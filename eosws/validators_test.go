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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noErrors = url.Values{}

type queryValidatorTestCase struct {
	name   string
	query  string
	errors url.Values
}

type cursorRuleTestCase struct {
	name   string
	cursor string
	error  error
}

func TestValidateSearchTransactionsRequest(t *testing.T) {
	validQuery := func(rest string) string {
		return "q=abcdef&" + rest
	}

	tests := []queryValidatorTestCase{
		{"q valid simple", "q=account:test", noErrors},
		{"q valid multiple simples", "q=account:test receiver:eos.msig", noErrors},
		{"q valid multiple with parenthesis", "q=account:test (data.from:test or data.to:other)", noErrors},
		{"q valid multiple with double quotes", `q=account:test data.memo:"test"`, noErrors},
		{"q valid multiple with single boolean", "q=account:test scheduled:true notif:false", noErrors},
		{"q valid on exact min length", "q=abcdef", noErrors},

		{"q not long enough, 0", "q=", url.Values{
			"q": []string{"The q field is required", "The q field must be minimum 5 char"},
		}},

		{"q not long enough, flush on min", "q=auth", url.Values{
			"q": []string{"The q field must be minimum 5 char"},
		}},

		{"start_block zero valid", validQuery("start_block=0"), noErrors},
		{"start_block one valid", validQuery("start_block=1"), noErrors},
		{"start_block big valid", validQuery("start_block=1000000"), noErrors},

		{"start_block negative invalid", validQuery("start_block=-1"), url.Values{
			"start_block": []string{"The start_block field must be numeric value between 0 and 4294967295"},
		}},
		{"start_block non numeric invalid", validQuery("start_block=a"), url.Values{
			"start_block": []string{"The start_block field must be a valid EOS block num", "The start_block field must be numeric value between 0 and 4294967295"},
		}},

		{"block_count on lower limit valid", validQuery("block_count=1"), noErrors},
		{"block_count higher than lower limit valid", validQuery("block_count=1000"), noErrors},

		{"block_count below lower limit invalid", validQuery("block_count=0"), url.Values{
			"block_count": []string{"The block_count field value can not be less than 1"},
		}},
		{"block_count non numeric invalid", validQuery("block_count=a"), url.Values{
			"block_count": []string{"The block_count field must be numeric", "The block_count field value can not be less than 1"},
		}},

		{"limit on lower limit valid", validQuery("limit=1"), noErrors},
		{"limit on upper limit valid", validQuery("limit=100"), noErrors},

		{"limit below lower limit invalid", validQuery("limit=0"), url.Values{
			"limit": []string{"The limit field must be numeric value between 1 and 100"},
		}},
		{"limit above upper limit invalid", validQuery("limit=101"), url.Values{
			"limit": []string{"The limit field must be numeric value between 1 and 100"},
		}},
		{"limit non numeric invalid", validQuery("limit=a"), url.Values{
			"limit": []string{"The limit field must be numeric", "The limit field must be numeric value between 1 and 100"},
		}},

		{"cursor empty valid", validQuery("cursor="), url.Values{}},
		{"cursor invalid format", validQuery("cursor=---"), url.Values{"cursor": []string{"The cursor field is not a valid cursor"}}},

		{"sort desc is valid", validQuery("sort=desc"), noErrors},
		{"sort asc is valid", validQuery("sort=asc"), noErrors},
		{"sort DESC is valid", validQuery("sort=DESC"), noErrors},
		{"sort ASC is valid", validQuery("sort=ASC"), noErrors},
		{"sort dESc is valid", validQuery("sort=dESc"), noErrors},
		{"sort AsC is valid", validQuery("sort=AsC"), noErrors},

		{"sort empty is invalid", validQuery("sort="), url.Values{
			"sort": []string{"The sort field must be one of desc, asc"},
		}},
		{"sort anything else is invalid", validQuery("sort=bla"), url.Values{
			"sort": []string{"The sort field must be one of desc, asc"},
		}},

		{"with_reversible true is valid", validQuery("with_reversible=true"), noErrors},
		{"with_reversible false is valid", validQuery("with_reversible=false"), noErrors},

		{"with_reversible empty is invalid", validQuery("with_reversible="), url.Values{
			"with_reversible": []string{"The with_reversible field must be one of true, false"},
		}},
		{"with_reversible anything else is invalid", validQuery("with_reversible=bla"), url.Values{
			"with_reversible": []string{"The with_reversible field must be one of true, false"},
		}},
	}

	runQueryValidatorTests(t, "search/transactions", tests, validateSearchTransactionsRequest)
}

func TestValidateListRequest(t *testing.T) {
	validQuery := func(rest string) string {
		return "cursor=_R0R0k5_kFS28uXkTwahGPazJ8Q9BFNmBw60LBITgIL0piaTi8mmUmN2aEmGk_r43BPoGgz63tvFQHgu9JJX6oTux79kvyZpFyh6wN-7_7HlePPzOw%3D%3D&limit=100"
	}

	tests := []queryValidatorTestCase{
		{"happy path", validQuery(""), url.Values{}},
		{"cursor empty valid", "cursor=&limit=25", url.Values{}},
		{"wrong cursor error", "cursor=---&limit=40", url.Values{"cursor": []string{"The cursor field is not a valid cursor"}}},
		{"limit is required error", "cursor=", url.Values{"limit": []string{"The limit field is required", "The limit field must be numeric value between 1 and 100"}}},
		{"limit is too big error", "cursor=&limit=101", url.Values{"limit": []string{"The limit field must be numeric value between 1 and 100"}}},
	}

	runQueryValidatorTests(t, "/things", tests, ValidateListRequest)
}

func TestValidateBlocksRequest(t *testing.T) {
	validQuery := func(rest string) string {
		return "skip=60000&limit=100"
	}

	tests := []queryValidatorTestCase{
		{"happy path", validQuery(""), url.Values{}},
		{"skip empty valid", "limit=25", url.Values{}},
		{"wrong skip error", "skip=refre&limit=40", url.Values{"skip": []string{"The skip field must be numeric"}}},
		{"limit is required error", "skip=300", url.Values{"limit": []string{"The limit field is required", "The limit field must be numeric value between 1 and 100"}}},
		{"limit is too big error", "skip=90&limit=101", url.Values{"limit": []string{"The limit field must be numeric value between 1 and 100"}}},
	}

	runQueryValidatorTests(t, "/blocks", tests, ValidateBlocksRequest)
}

func runQueryValidatorTests(t *testing.T, tag string, tests []queryValidatorTestCase, validator func(r *http.Request) url.Values) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", tag, test.name), func(t *testing.T) {
			request, err := http.NewRequest("GET", "/?"+test.query, nil)
			require.NoError(t, err)

			errors := validator(request)
			assert.Equal(t, test.errors, errors)
		})
	}
}
