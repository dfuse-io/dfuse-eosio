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
	"sort"
	"strconv"
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

type blocknumBasedCELFilter map[uint64]*CELFilter

func (bbcf blocknumBasedCELFilter) String() (out string) {
	if len(bbcf) == 1 {
		for _, v := range bbcf {
			return v.code
		}
	}
	var arr []uint64
	for k := range bbcf {
		arr = append(arr, k)
	}
	sort.Slice(arr, func(i int, j int) bool { return arr[i] < arr[j] })
	for _, k := range arr {
		out += fmt.Sprintf("#%d;%s", k, bbcf[k].code)
	}
	return
}

func (bbcf blocknumBasedCELFilter) choose(blknum uint64) *CELFilter {
	var highestMatchingKey uint64
	for k := range bbcf {
		if blknum >= k && k > highestMatchingKey {
			highestMatchingKey = k
		}
	}
	if v, ok := bbcf[highestMatchingKey]; ok {
		return v
	}
	return nil
}

func (f *CELFilter) IsNoop() bool {
	return f.program == nil
}

func newCELFiltersInclude(codes []string) (blocknumBasedCELFilter, error) {
	return newCELFilters("inclusion", codes, []string{"", "true", "*"}, true)
}

func newCELFiltersSystemActionsInclude(codes []string) (blocknumBasedCELFilter, error) {
	return newCELFilters("system action inclusion", codes, []string{"false", ""}, false)
}

func newCELFiltersExclude(codes []string) (blocknumBasedCELFilter, error) {
	return newCELFilters("exclusion", codes, []string{"", "false"}, false)
}

func parseBlocknumBasedCode(code string) (out string, blocknum uint64, err error) {
	parts := strings.SplitN(code, ";", 2)
	if len(parts) == 1 {
		return parts[0], 0, nil
	}
	if !strings.HasPrefix(parts[0], "#") {
		return "", 0, fmt.Errorf("invalid block num part")
	}
	blocknum, err = strconv.ParseUint(strings.TrimLeft(parts[0], "#"), 10, 64)
	out = strings.Trim(parts[1], " ")
	return
}

func newCELFilters(name string, codes []string, noopPrograms []string, valueWhenNoop bool) (filtersMap blocknumBasedCELFilter, err error) {
	filtersMap = make(map[uint64]*CELFilter)
	for _, code := range codes {
		parsedCode, blockNum, err := parseBlocknumBasedCode(code)
		if err != nil {
			return nil, err
		}

		filter, err := newCELFilter(name, parsedCode, noopPrograms, valueWhenNoop)
		if err != nil {
			return nil, err
		}
		if _, ok := filtersMap[blockNum]; ok {
			return nil, fmt.Errorf("blocknum %d declared twice in filter", blockNum)
		}

		filtersMap[blockNum] = filter
	}
	return
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
			decls.NewIdent("top5_trx_actors", decls.NewListType(decls.String), nil),

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

	trxTop5ActorsGetter func() []string
	trxScheduled        bool
	trxActionCount      int
}

func (a *actionTraceActivation) Parent() interpreter.Activation {
	return nil
}

// exclude_filter: (trx_action_count > 200 && top5_trx_actors.exists(x in ['pizzapizza', 'eidosonecoin'])

func (a *actionTraceActivation) ResolveName(name string) (interface{}, bool) {
	if traceEnabled {
		zlog.Debug("trying to resolve activation name", zap.String("name", name))
	}

	switch name {
	case "top5_trx_actors":
		return a.trxTop5ActorsGetter(), true
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
	out = make([]string, len(authorizations)*2)
	for i, auth := range authorizations {
		out[i*2] = auth.Actor
		out[i*2+1] = auth.Authorization()
	}

	return
}
