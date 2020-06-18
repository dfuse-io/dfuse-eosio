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
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
)

func buildCELProgram(noopProgram string, programString string) (cel.Program, error) {
	stripped := strings.TrimSpace(programString)
	if stripped == "" || stripped == noopProgram {
		return nil, nil
	}

	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("db", decls.NewMapType(decls.String, decls.String), nil), // "table", "key" => string
			// TODO: Eventually, build a sub-struct based on the fields declared in `indexed terms specs`
			decls.NewIdent("data", decls.NewMapType(decls.String, decls.Any), nil),
			// TODO: Conditionally add when `ram.consumed`, and `ram.released`
			decls.NewIdent("ram", decls.NewMapType(decls.String, decls.String), nil),
			decls.NewIdent("receiver", decls.String, nil),
			decls.NewIdent("account", decls.String, nil),
			decls.NewIdent("action", decls.String, nil),
			decls.NewIdent("auth", decls.NewListType(decls.String), nil),
			decls.NewIdent("input", decls.Bool, nil),
			decls.NewIdent("notif", decls.Bool, nil),
			decls.NewIdent("scheduled", decls.Bool, nil),
		),

		// Search AND trxdb:
		//   receiver == "eosio.token" || action == "bobbob"
		// FluxDB:
		//   (code == "eosio" && scope.startswith("eosio.")) || code == "mycontract || key == "4,EOS"
		//   (code == "eosio" && scope.startswith("eosio.")) || code == "mycontract || table == "stat" || key == "4,EOS"
		//
	)
	if err != nil {
		return nil, err
	}

	exprAst, issues := env.Compile(programString)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("filter expression parse/check error: %w", issues.Err())
	}

	if exprAst.ResultType() != decls.Bool {
		return nil, fmt.Errorf("filter expression should return a boolean, returned %s", exprAst.ResultType())
	}

	prg, err := env.Program(exprAst)
	if err != nil {
		return nil, fmt.Errorf("cel program construction error: %w", err)
	}

	return prg, nil
}

func (m *BlockMapper) shouldIndexAction(doc map[string]interface{}) bool {
	filterOnResult := m.filterMatches(m.filterOnProgram, true, doc)
	filterOutResult := m.filterMatches(m.filterOutProgram, false, doc)
	return filterOnResult && !filterOutResult
}

func (m *BlockMapper) filterMatches(program cel.Program, defaultVal bool, doc map[string]interface{}) bool {
	if program == nil {
		return defaultVal
	}

	res, _, err := program.Eval(doc)
	if err != nil {
		//fmt.Printf("filter program: %s\n", err.Error())
		return false
	}
	retval, valid := res.(types.Bool)
	if !valid {
		// TODO: use logger, we've checked the return value should be a Bool previously, so
		// it's even safe to panic here
		panic("return value of our cel program isn't of type bool")
	}
	return bool(retval)
}
