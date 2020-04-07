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

package eosws

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/dfuse-io/validator"
	"github.com/thedevsaddam/govalidator"
)

func init() {
	govalidator.AddCustomRule("eos.blockNum", validator.EOSBlockNumRule)
	govalidator.AddCustomRule("eos.name", validator.EOSNameRule)
	govalidator.AddCustomRule("eos.trxID", validator.EOSTrxIDRule)

	govalidator.AddCustomRule("eosws.cursor", validator.CursorRule)
	govalidator.AddCustomRule("eosws.search.sortOrder", sortOrderRule)
}

func ValidateBlocksRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"skip":  []string{"numeric"},
		"limit": []string{"required", "numeric_between:1,100"},
	})
}

func ValidateListRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"limit":  []string{"required", "numeric_between:1,100"},
		"cursor": []string{"eosws.cursor"},
	})
}

func validateSearchTransactionsRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"q":               []string{"required", "min:5"},
		"start_block":     []string{"eos.blockNum", fmt.Sprintf("numeric_between:0,%d", math.MaxUint32)},
		"block_count":     []string{"numeric", "numeric_between:1,"},
		"limit":           []string{"numeric", "numeric_between:1,100"},
		"cursor":          []string{"eosws.cursor"},
		"sort":            []string{"eosws.search.sortOrder"},
		"with_reversible": []string{"in:true,false"},
		"format":          []string{},
	})
}

func sortOrderRule(field string, rule string, message string, value interface{}) error {
	val, ok := value.(string)
	if !ok {
		return fmt.Errorf("The %s field must be a string", field)
	}

	loweredVal := strings.ToLower(val)
	if loweredVal == "desc" || loweredVal == "asc" {
		return nil
	}

	return fmt.Errorf("The %s field must be one of desc, asc", field)
}
