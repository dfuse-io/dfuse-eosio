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

package server

import (
	"fmt"

	"github.com/dfuse-io/validator"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"
)

const maxAccountCount = 1500
const maxScopeCount = 1500

func init() {
	govalidator.AddCustomRule("fluxdb.eos.accountsList", validator.EOSNamesListRuleFactory("|", maxAccountCount))
	govalidator.AddCustomRule("fluxdb.eos.blockNum", validator.EOSBlockNumRule)
	govalidator.AddCustomRule("fluxdb.eos.hexRows", validator.HexRowsRule)
	govalidator.AddCustomRule("fluxdb.eos.name", validator.EOSNameRule)
	govalidator.AddCustomRule("fluxdb.eos.extendedName", validator.EOSExtendedNameRule)
	govalidator.AddCustomRule("fluxdb.eos.publicKey", eosPublicKeyRule)
	govalidator.AddCustomRule("fluxdb.eos.scopesList", validator.EOSExtendedNamesListRuleFactory("|", maxScopeCount))
}

// FIXME: Extract to `github.com/dfuse-io/validator` library (with associated tests)
func eosPublicKeyRule(field string, rule string, message string, value interface{}) error {
	switch v := value.(type) {
	case string:
		_, err := ecc.NewPublicKey(v)
		if err != nil {
			zlog.Info("The public key was not parseable.", zap.String("public_key", v), zap.String("error", err.Error()))
			return fmt.Errorf("The %s field must be a valid EOS public key", field)
		}

		return nil
	case ecc.PublicKey, *ecc.PublicKey:
		return nil
	default:
		return fmt.Errorf("The %s field is not a known type for an EOS public key", field)
	}
}

func withCommonValidationRules(extraRules validator.Rules) validator.Rules {
	rules := commonReadValidationRules()
	for key, validators := range extraRules {
		rules[key] = validators
	}

	return rules
}

func commonReadValidationRules() validator.Rules {
	return validator.Rules{
		"block_num":      []string{"fluxdb.eos.blockNum"},
		"offset":         []string{"numeric"},
		"limit":          []string{"numeric"},
		"key_type":       []string{"in:hex,hex_be,uint64,name,symbol,symbol_code"},
		"json":           []string{"bool"},
		"with_abi":       []string{"bool"},
		"with_block_num": []string{"bool"},
	}
}
