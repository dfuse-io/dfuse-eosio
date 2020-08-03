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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bodyValidatorTestCase struct {
	name   string
	body   string
	errors url.Values
}

type queryValidatorTestCase struct {
	name   string
	query  string
	errors url.Values
}

type ruleTestCase struct {
	name          string
	value         interface{}
	expectedError string
}

func TestValidateDecodeABIRequest(t *testing.T) {
	tests := []bodyValidatorTestCase{
		{"all valid", `{"account":"c", "table": "t", "hex_rows":["abcf"], "block_num":11}`, url.Values{}},

		{"account required", `{"table": "t", "hex_rows":["abcf"]}`, url.Values{
			"account": []string{"The account field is required"},
		}},

		{"account not name", `{"account":"9", "table": "t", "hex_rows":["abcf"]}`, url.Values{
			"account": []string{"The account field must be a valid EOS name"},
		}},

		{"table required", `{"account":"a", "hex_rows":["abcf"]}`, url.Values{
			"table": []string{"The table field is required"},
		}},

		{"table not name", `{"account":"a", "table": "9999", "hex_rows":["abcf"]}`, url.Values{
			"table": []string{"The table field must be a valid EOS name"},
		}},

		{"hex_data required", `{"account":"a", "table": "t"}`, url.Values{
			"hex_rows": []string{"The hex_rows field is required", "The hex_rows field must have at least 1 element"},
		}},

		{"hex_data empty", `{"account":"a", "table": "t", "hex_rows": []}`, url.Values{
			"hex_rows": []string{"The hex_rows field is required", "The hex_rows field must have at least 1 element"},
		}},

		{"hex_data invalid format", `{"account":"a", "table": "t", "hex_rows": "abc"}`, url.Values{
			"_error": []string{"json: cannot unmarshal string into Go struct field decodeABIRequest.hex_rows of type []string"},
		}},

		{"hex_data invalid hexadeciaml", `{"account":"a", "table": "t", "hex_rows": ["abc", "zzz"]}`, url.Values{
			"hex_rows": []string{"The hex_rows[0] field must be a valid hexadecimal"},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := &decodeABIRequest{}

			req, err := http.NewRequest("POST", "/", strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}

			errors := validateDecodeABIRequest(req, request)
			assert.Equal(t, test.errors, errors)
		})
	}
}

func TestValidateGetABIRequest(t *testing.T) {
	validQuery := func(rest string) string {
		return "account=a&" + rest
	}

	tests := []queryValidatorTestCase{
		{"account required", "", url.Values{
			"account": []string{"The account field is required"},
		}},

		{"account not name", "account=9", url.Values{
			"account": []string{"The account field must be a valid EOS name"},
		}},

		{"block_num valid", validQuery("block_num=1"), url.Values{}},

		{"block_num not valid", validQuery("block_num=a"), url.Values{
			"block_num": []string{"The block_num field must be a valid EOS block num"},
		}},

		{"json valid 0", validQuery("json=0"), url.Values{}},
		{"json valid true", validQuery("json=true"), url.Values{}},
		{"json not boolean", validQuery("json=a"), url.Values{
			"json": []string{"The json may only contain boolean value, string or int 0, 1"},
		}},
	}

	runQueryValidatorTests(t, "TestValidateGetABIRequest", tests, validateGetABIRequest)
}

func TestValidateGetTableRequest(t *testing.T) {
	validateCommonReadRequest(t, "single_read", "table=a&account=c&scope=b", validateGetTableRequest)

	tests := []queryValidatorTestCase{
		{"table required", "account=c&scope=b", url.Values{
			"table": []string{"The table field is required"},
		}},

		{"table not name", "account=c&scope=b&table=9", url.Values{
			"table": []string{"The table field must be a valid EOS name"},
		}},

		{"account required", "table=c&scope=b", url.Values{
			"account": []string{"The account field is required"},
		}},

		{"account not name", "account=9&scope=b&table=a", url.Values{
			"account": []string{"The account field must be a valid EOS name"},
		}},

		{"scope empty", "account=c&scope=&table=a", url.Values{}},

		{"scope name", "account=c&scope=s&table=a", url.Values{}},

		{"scope symbol", "account=c&scope=4,EOS&table=a", url.Values{}},

		{"scope symbol code", "account=c&scope=EOS&table=a", url.Values{}},

		{"scope required", "account=c&table=a", url.Values{
			"scope": []string{"The scope field is required"},
		}},

		{"scope not name", "account=c&scope=0&table=a", url.Values{
			"scope": []string{"The scope field must be a valid EOS name"},
		}},
	}

	runQueryValidatorTests(t, "TestValidateGetTableRequest", tests, validateGetTableRequest)
}

func TestValidateListTablesRowsForAccountsRequest(t *testing.T) {
	validateCommonReadRequest(t, "multi_accounts", "accounts=a&table=t&scope=s", validateListTablesRowsForAccountsRequest)

	accountsAboveMax := strings.Repeat("a|", maxAccountCount) + "a"

	tests := []queryValidatorTestCase{
		{"table required", "accounts=c&scope=b", url.Values{
			"table": []string{"The table field is required"},
		}},

		{"table not name", "accounts=c&scope=b&table=9", url.Values{
			"table": []string{"The table field must be a valid EOS name"},
		}},

		{"accounts required", "table=c&scope=b", url.Values{
			"accounts": []string{"The accounts field is required", "The accounts field must have at least 1 element"},
		}},

		{"accounts invalid name", "accounts=9&scope=b&table=a", url.Values{
			"accounts": []string{`The accounts[0] field must be a valid EOS name`},
		}},

		{"accounts above max", fmt.Sprintf("accounts=%s&scope=b&table=a", accountsAboveMax), url.Values{
			"accounts": []string{"The accounts field must have at most 1500 elements"},
		}},

		{"scope empty", "accounts=c&scope=&table=a", url.Values{}},

		{"scope name", "accounts=c&scope=s&table=a", url.Values{}},

		{"scope symbol", "accounts=c&scope=4,EOS&table=a", url.Values{}},

		{"scope symbol code", "accounts=c&scope=EOS&table=a", url.Values{}},

		{"scope required", "accounts=c&table=a", url.Values{
			"scope": []string{"The scope field is required"},
		}},

		{"scope not name", "accounts=c&scope=0&table=a", url.Values{
			"scope": []string{"The scope field must be a valid EOS name"},
		}},
	}

	runQueryValidatorTests(t, "TestValidateListTablesRowsForAccountsRequest", tests, validateListTablesRowsForAccountsRequest)
}

func TestValidateListTablesRowsForScopesRequest(t *testing.T) {
	validateCommonReadRequest(t, "multi_scope", "account=c&table=t&scopes=b", validateListTablesRowsForScopesRequest)

	scopesAboveMax := strings.Repeat("s|", maxScopeCount) + "s"

	tests := []queryValidatorTestCase{
		{"all valid", "account=a&table=t&scopes=s|4,EOS|EOS", url.Values{}},

		{"table required", "account=c&scopes=b", url.Values{
			"table": []string{"The table field is required"},
		}},

		{"table not name", "account=c&scopes=b&table=9", url.Values{
			"table": []string{"The table field must be a valid EOS name"},
		}},

		{"account required", "scopes=c&table=b", url.Values{
			"account": []string{"The account field is required"},
		}},

		{"account not name", "account=9&scopes=s&table=a", url.Values{
			"account": []string{"The account field must be a valid EOS name"},
		}},

		{"scopes required", "table=c&account=b", url.Values{
			"scopes": []string{"The scopes field is required", "The scopes field must have at least 1 element"},
		}},

		{"scopes one element needed", "table=c&account=b&scopes=", url.Values{
			"scopes": []string{"The scopes field is required", "The scopes field must have at least 1 element"},
		}},

		{"scopes above max", fmt.Sprintf("account=a&scopes=%s&table=a", scopesAboveMax), url.Values{
			"scopes": []string{"The scopes field must have at most 1500 elements"},
		}},

		{"scopes invalid name", "account=a&scopes=9&table=a", url.Values{
			"scopes": []string{`The scopes[0] field must be a valid EOS name`},
		}},
	}

	runQueryValidatorTests(t, "TestValidateListTablesRowsForScopesRequest", tests, validateListTablesRowsForScopesRequest)
}

func TestValidateGetLinkedPermssionsRequest(t *testing.T) {
	validQuery := func(rest string) string {
		return "account=a&" + rest
	}

	tests := []queryValidatorTestCase{
		{"account required", "", url.Values{
			"account": []string{"The account field is required"},
		}},

		{"account not name", "account=9", url.Values{
			"account": []string{"The account field must be a valid EOS name"},
		}},

		{"block_num valid", validQuery("block_num=1"), url.Values{}},

		{"block_num not valid", validQuery("block_num=a"), url.Values{
			"block_num": []string{"The block_num field must be a valid EOS block num"},
		}},
	}

	runQueryValidatorTests(t, "TestValidateGetLinkedPermssionsRequest", tests, validateGetLinkedPermissionsRequest)
}

func validateCommonReadRequest(t *testing.T, tag string, validQueryPrefix string, validator func(r *http.Request) url.Values) {
	validQuery := func(rest string) string {
		return validQueryPrefix + "&" + rest
	}

	tests := []queryValidatorTestCase{
		{"block_num valid", validQuery("block_num=1"), url.Values{}},
		{"offset valid", validQuery("offset=0"), url.Values{}},
		{"limit valid", validQuery("limit=0"), url.Values{}},
		{"key_type hex", validQuery("key_type=hex"), url.Values{}},
		{"key_type hex_be", validQuery("key_type=hex_be"), url.Values{}},
		{"key_type name", validQuery("key_type=name"), url.Values{}},
		{"key_type uint64", validQuery("key_type=uint64"), url.Values{}},
		{"json valid 0", validQuery("json=0"), url.Values{}},
		{"json valid true", validQuery("json=true"), url.Values{}},
		{"with_abi valid 0", validQuery("with_abi=0"), url.Values{}},
		{"with_abi valid true", validQuery("with_abi=true"), url.Values{}},
		{"with_block_num valid 0", validQuery("with_block_num=0"), url.Values{}},
		{"with_block_num valid true", validQuery("with_block_num=true"), url.Values{}},

		{"block_num not valid", validQuery("block_num=a"), url.Values{
			"block_num": []string{"The block_num field must be a valid EOS block num"},
		}},

		{"offset not a number", validQuery("offset=a"), url.Values{
			"offset": []string{"The offset field must be numeric"},
		}},

		{"limit not a number", validQuery("limit=a"), url.Values{
			"limit": []string{"The limit field must be numeric"},
		}},

		{"key_type invalid", validQuery("key_type=a"), url.Values{
			"key_type": []string{"The key_type field must be one of hex, hex_be, uint64, name, symbol, symbol_code"},
		}},

		{"json not boolean", validQuery("json=a"), url.Values{
			"json": []string{"The json may only contain boolean value, string or int 0, 1"},
		}},

		{"with_abi not boolean", validQuery("with_abi=a"), url.Values{
			"with_abi": []string{"The with_abi may only contain boolean value, string or int 0, 1"},
		}},

		{"with_block_num not boolean", validQuery("with_block_num=a"), url.Values{
			"with_block_num": []string{"The with_block_num may only contain boolean value, string or int 0, 1"},
		}},
	}

	runQueryValidatorTests(t, tag, tests, validator)
}

func Test_eosPublicKeyRule(t *testing.T) {
	tag := "public_key"
	validator := func(field string, value interface{}) error {
		return eosPublicKeyRule(field, tag, "", value)
	}

	validKey, err := ecc.NewPublicKey("EOS7YNS1swh6QWANkzGgFrjiX8E3u8WK5CK9GMAb6EzKVNZMYhCH3")
	require.NoError(t, err)

	tests := []ruleTestCase{
		{"should be a string", true, "The test field is not a known type for an EOS public key"},
		{"should be well formed", "EOS7YNS1swh6QWANkzGgFrjiX8E3u8WK5CK9GMAb6EzKVNZMYhCH4", "The test field must be a valid EOS public key"},

		{"valid EOS string", "EOS7YNS1swh6QWANkzGgFrjiX8E3u8WK5CK9GMAb6EzKVNZMYhCH3", ""},
		{"valid R1 string", "PUB_R1_78rbUHSk87e7eCBoccgWUkhNTCZLYdvJzerDRHg6fxj2SQy6Xm", ""},
		{"valid K1 string", "PUB_K1_6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV", ""},
		{"valid eos.PublicKey", validKey, ""},
	}

	runRuleTestCases(t, tag, tests, validator)
}

func runQueryValidatorTests(t *testing.T, tag string, tests []queryValidatorTestCase, validator func(r *http.Request) url.Values) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", tag, test.name), func(t *testing.T) {
			req, err := http.NewRequest("GET", "/?"+test.query, nil)
			if err != nil {
				t.Fatal(err)
			}

			errors := validator(req)
			assert.Equal(t, test.errors, errors)
		})
	}
}

func runRuleTestCases(t *testing.T, tag string, tests []ruleTestCase, validator func(field string, value interface{}) error) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", tag, test.name), func(t *testing.T) {
			err := validator("test", test.value)

			if test.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, errors.New(test.expectedError), err)
			}
		})
	}
}
