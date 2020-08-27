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
	"go.uber.org/zap"
)

type CELFilter struct {
	name          string
	code          string
	program       cel.Program
	valueWhenNoop bool
}

func (f *CELFilter) IsNoop() bool {
	return f.program == nil
}

func newCELFilterInclude(code string) (*CELFilter, error) {
	return newCELFilter("inclusion", code, []string{"", "true", "*"}, true)
}

func newCELFilterSystemActionsInclude(code string) (*CELFilter, error) {
	return newCELFilter("system action inclusion", code, []string{"false", ""}, false)
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
			decls.NewIdent("trx_action_count", decls.Int, nil), // Amount of actions in the transaction in which this action is part of.

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
	if f.IsNoop() {
		return f.valueWhenNoop
	}

	res, _, err := f.program.Eval(activation)
	if err != nil {
		if traceEnabled {
			zlog.Debug("filter program failed", zap.String("name", f.name), zap.Error(err))
		}
		return f.valueWhenNoop
	}

	retval, valid := res.(types.Bool)
	if !valid {
		zlog.Error("return value of our cel program isn't of type bool, this should never happen since we've checked the return value type already")
		return f.valueWhenNoop
	}

	if traceEnabled {
		zlog.Debug("filter program executed correctly", zap.String("name", f.name), zap.Bool("matched", bool(retval)))
	}

	return bool(retval)
}

type actionTraceActivation struct {
	trace      *pbcodec.ActionTrace
	cachedData map[string]interface{}

	trxScheduled bool
	trxActionCount  int
}

func (a *actionTraceActivation) Parent() interpreter.Activation {
	return nil
}

func (a *actionTraceActivation) ResolveName(name string) (interface{}, bool) {
	if traceEnabled {
		zlog.Debug("trying to resolve activation name", zap.String("name", name))
	}

	switch name {
	case "trx_action_count":
		return a.trxActionCount, true
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
		if a.cachedData != nil {
			return a.cachedData, true
		}

		jsonData := a.trace.Action.JsonData
		if len(jsonData) == 0 || strings.IndexByte(jsonData, '{') == -1 {
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

		if a.cachedData == nil {
			a.cachedData = out
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
	out = make([]string, len(authorizations))
	for i, auth := range authorizations {

		out[i*2] = auth.Actor
		out[i*2+1] = auth.Authorization()
	}

	return
}
