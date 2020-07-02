// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filtering

import (
	"encoding/json"
	"fmt"
	"strings"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/interpreter"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type CELFilter struct {
	name          string
	code          string
	program       cel.Program
	valueWhenNoop bool
}

func newCELFilterInclude(code string) (*CELFilter, error) {
	return newCELFilter("inclusion", code, []string{"", "true", "*"}, true)
}

func newCELFilterExclude(code string) (*CELFilter, error) {
	return newCELFilter("exclusion", code, []string{"", "false"}, false)
}

func newCELFilter(name string, code string, noopPrograms []string, valueWhenNoop bool) (*CELFilter, error) {
	stripped := strings.TrimSpace(code)
	for _, noopProgram := range noopPrograms {
		if stripped == noopProgram {
			return &CELFilter{
				name:          name,
				code:          stripped,
				valueWhenNoop: valueWhenNoop,
			}, nil
		}
	}

	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("receiver", decls.String, nil),
			decls.NewIdent("account", decls.String, nil),
			decls.NewIdent("action", decls.String, nil),
			decls.NewIdent("data", decls.NewMapType(decls.String, decls.Any), nil),
			decls.NewIdent("auth", decls.NewListType(decls.String), nil),
			decls.NewIdent("input", decls.Bool, nil),
			decls.NewIdent("notif", decls.Bool, nil),
			decls.NewIdent("scheduled", decls.Bool, nil),

			// Those are not supported right now, so they are commented out for now to generate an error when using them
			// decls.NewIdent("db", decls.NewMapType(decls.String, decls.String), nil),
			// decls.NewIdent("ram", decls.NewMapType(decls.String, decls.String), nil),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("new env: %w", err)
	}

	exprAst, issues := env.Compile(stripped)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("parse filter: %w", issues.Err())
	}

	if exprAst.ResultType() != decls.Bool {
		return nil, fmt.Errorf("invalid return type %q", exprAst.ResultType())
	}

	prg, err := env.Program(exprAst)
	if err != nil {
		return nil, fmt.Errorf("program: %w", err)
	}

	return &CELFilter{
		name:          name,
		code:          code,
		program:       prg,
		valueWhenNoop: valueWhenNoop,
	}, nil
}

func (f *CELFilter) match(activation interpreter.Activation) (matched bool) {
	if f.program == nil {
		return f.valueWhenNoop
	}

	res, _, err := f.program.Eval(activation)
	if err != nil {
		if traceEnabled {
			zlog.Debug("filter program failed", zap.String("name", f.name), zap.Error(err))
		}
		return
	}

	retval, valid := res.(types.Bool)
	if !valid {
		// We've checked the return value should be a Bool previously, so it's safe to panic here
		panic("return value of our cel program isn't of type bool")
	}

	if traceEnabled {
		zlog.Debug("filter program executed correctly", zap.String("name", f.name), zap.Bool("matched", bool(retval)))
	}

	return bool(retval)
}

type actionTraceActivation struct {
	trace *pbcodec.ActionTrace

	trxScheduled bool
}

func (a *actionTraceActivation) Parent() interpreter.Activation {
	return nil
}

func (a *actionTraceActivation) ResolveName(name string) (interface{}, bool) {
	if traceEnabled {
		zlog.Debug("trying to resolve activation name", zap.String("name", name))
	}

	switch name {
	case "receiver":
		if a.trace.Receipt != nil {
			return a.trace.Receipt.Receiver, true
		}
		return a.trace.Receiver, true
	case "account":
		return a.trace.Account(), true
	case "action":
		return a.trace.Name(), true
	case "auth":
		return tokenizeEOSAuthority(a.trace.Action.Authorization), true
	case "data":
		if len(a.trace.Action.JsonData) == 0 {
			return nil, false
		}
		var out map[string]interface{}
		err := json.Unmarshal([]byte(a.trace.Action.JsonData), &out)
		if err != nil {
			if traceEnabled {
				zlog.Warn("invalid json data", zap.Error(err), zap.String("json", a.trace.Action.JsonData))
			}

			return nil, false
		}

		return out, true
	case "notif":
		receiver := a.trace.Receiver
		if a.trace.Receipt != nil {
			receiver = a.trace.Receipt.Receiver
		}
		return a.trace.Account() != receiver, true
	case "scheduled":
		return a.trxScheduled, true
	case "input":
		return a.trace.IsInput(), true

	// Those are actually commented out from the valid list of names that is possible to use in our CEL program,
	// so it's not a big deal to have them currently panicking here as it no code path can reach this point.
	case "ram":
		panic("CEL filtering does not yet support ram.consumed nor ram.released")
	case "db":
		panic("CEL filtering does not yet support db.table, db.key, etc..")
	}

	return nil, false
}

// This must follow rules taken in `search/tokenization.go`, ideally we would share this, maybe would be a good idea to
// put the logic in an helper method on type `pbcodec.PermissionLevel` directly.
func tokenizeEOSAuthority(authorizations []*pbcodec.PermissionLevel) (out []string) {
	for _, auth := range authorizations {
		actor := auth.Actor
		perm := auth.Permission
		out = append(out, actor, fmt.Sprintf("%s@%s", actor, perm))
	}

	return
}

type dataActivation struct {
	parent actionTraceActivation
	gjson.Result
}

func (a *dataActivation) Parent() interpreter.Activation {
	return &a.parent
}

func (a *dataActivation) ResolveName(name string) (interface{}, bool) {
	if traceEnabled {
		zlog.Debug("trying to resolve activation name", zap.String("name", name))
	}

	res := a.Get(name)
	if len(res.Raw) == 0 {
		return nil, false
	}

	return res.Value(), true
}
