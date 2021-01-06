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

func (bbcf blocknumBasedCELFilter) choose(blknum uint64) (out *CELFilter) {
	var highestMatchingKey uint64
	for k, v := range bbcf {
		if blknum >= k && k >= highestMatchingKey {
			highestMatchingKey = k
			out = v
		}
	}
	return
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
	if _, ok := filtersMap[0]; !ok { // create noop filtermap
		filtersMap[0] = &CELFilter{
			name:          name,
			code:          "",
			valueWhenNoop: valueWhenNoop,
		}
	}

	return
}

var ActionTraceDeclarations = cel.Declarations(
	decls.NewIdent("receiver", decls.String, nil), // eosio.account_name::receiver
	decls.NewIdent("account", decls.String, nil),  // eosio.account_name::account
	decls.NewIdent("action", decls.String, nil),   // eosio.name::action

	decls.NewIdent("block_num", decls.Uint, nil),    // uint32 block number
	decls.NewIdent("block_id", decls.String, nil),   // string block id (hash)
	decls.NewIdent("block_time", decls.String, nil), // string timestamp

	decls.NewIdent("step", decls.String, nil),            // one of: Irreversible, New, Undo, Redo, Unknown
	decls.NewIdent("transaction_id", decls.String, nil),  // string transaction id (hash)
	decls.NewIdent("transaction_index", decls.Uint, nil), // uint transaction position inside the block
	decls.NewIdent("global_seq", decls.Uint, nil),        // uint
	decls.NewIdent("execution_index", decls.Uint, nil),   // uint action position inside the transaction

	decls.NewIdent("data", decls.NewMapType(decls.String, decls.Any), nil),
	decls.NewIdent("auth", decls.NewListType(decls.String), nil),
	decls.NewIdent("input", decls.Bool, nil),
	decls.NewIdent("notif", decls.Bool, nil),
	decls.NewIdent("scheduled", decls.Bool, nil),

	decls.NewIdent("trx_action_count", decls.Int, nil), // Number of actions in the transaction in which this action is part of.
	decls.NewIdent("top5_trx_actors", decls.NewListType(decls.String), nil),

	// Those are not supported right now, so they are commented out for now to generate an error when using them
	// decls.NewIdent("db", decls.NewMapType(decls.String, decls.String), nil),
	// decls.NewIdent("ram", decls.NewMapType(decls.String, decls.String), nil),
)

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

	env, err := cel.NewEnv(ActionTraceDeclarations)
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

func NewActionTraceActivation(
	actionTrace *pbcodec.ActionTrace,
	trxTrace *MemoizableTrxTrace,
	stepName string,
) *ActionTraceActivation {
	activation := &ActionTraceActivation{
		Trace:    actionTrace,
		TrxTrace: trxTrace,
		StepName: shortStepName(stepName),
	}
	return activation
}

type MemoizableTrxTrace struct {
	TrxTrace   *pbcodec.TransactionTrace
	top5Actors []string
}

func (t *MemoizableTrxTrace) getTop5Actors() []string {
	if t.top5Actors == nil {
		t.top5Actors = getTop5ActorsForTrx(t.TrxTrace)
	}
	return t.top5Actors
}

type ActionTraceActivation struct {
	Trace      *pbcodec.ActionTrace
	TrxTrace   *MemoizableTrxTrace
	StepName   string
	cachedData map[string]interface{}
}

func shortStepName(in string) string {
	if in == "" {
		return "Unknown"
	}
	return strings.ToTitle(strings.TrimPrefix(in, "STEP_"))
}

func (a *ActionTraceActivation) Parent() interpreter.Activation {
	return nil
}

// exclude_filter: (trx_action_count > 200 && top5_trx_actors.exists(x in ['pizzapizza', 'eidosonecoin'])

func (a *ActionTraceActivation) ResolveName(name string) (interface{}, bool) {
	if traceEnabled {
		zlog.Debug("trying to resolve activation name", zap.String("name", name))
	}

	switch name {
	case "block_num":
		return a.TrxTrace.TrxTrace.BlockNum, true
	case "block_id":
		return a.TrxTrace.TrxTrace.ProducerBlockId, true
	case "block_time":
		return a.TrxTrace.TrxTrace.BlockTime.AsTime().Format("2006-01-02T15:04:05.0Z07:00"), true
	case "transaction_id":
		return a.TrxTrace.TrxTrace.Id, true
	case "transaction_index":
		return a.TrxTrace.TrxTrace.Index, true
	case "step":
		return a.StepName, true

	case "global_seq":
		return a.Trace.Receipt.GlobalSequence, true
	case "execution_index":
		return a.Trace.ExecutionIndex, true

	case "top5_trx_actors":
		return a.TrxTrace.getTop5Actors(), true
	case "trx_action_count":
		return len(a.TrxTrace.TrxTrace.ActionTraces), true

	case "receiver":
		if a.Trace.Receipt != nil {
			return a.Trace.Receipt.Receiver, true
		}
		return a.Trace.Receiver, true
	case "account":
		return a.Trace.Account(), true
	case "action":
		return a.Trace.Name(), true
	case "auth":
		return tokenizeEOSAuthority(a.Trace.Action.Authorization), true
	case "data":
		if a.cachedData != nil {
			return a.cachedData, true
		}

		jsonData := a.Trace.Action.JsonData
		if len(jsonData) == 0 || strings.IndexByte(jsonData, '{') == -1 {
			return nil, false
		}

		var out map[string]interface{}
		err := json.Unmarshal([]byte(a.Trace.Action.JsonData), &out)
		if err != nil {
			if traceEnabled {
				zlog.Warn("invalid json data", zap.Error(err), zap.String("json", a.Trace.Action.JsonData))
			}

			return nil, false
		}

		if a.cachedData == nil {
			a.cachedData = out
		}

		return out, true
	case "notif":
		receiver := a.Trace.Receiver
		if a.Trace.Receipt != nil {
			receiver = a.Trace.Receipt.Receiver
		}
		return a.Trace.Account() != receiver, true
	case "scheduled":
		return a.TrxTrace.TrxTrace.Scheduled, true
	case "input":
		return a.Trace.IsInput(), true

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
