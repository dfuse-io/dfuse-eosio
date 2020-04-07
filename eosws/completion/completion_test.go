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

package completion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"

	"github.com/stretchr/testify/assert"
)

func TestComplete(t *testing.T) {
	section := func(name string, suggestionLabels ...string) *mdl.SuggestionSection {
		suggestions := make([]*mdl.Suggestion, len(suggestionLabels))
		for i, suggestionLabel := range suggestionLabels {
			parts := strings.Split(suggestionLabel, "|")
			summary := ""
			if len(parts) > 1 {
				summary = parts[1]
			}

			suggestions[i] = &mdl.Suggestion{Key: parts[0], Label: parts[0], Summary: summary}
		}

		return &mdl.SuggestionSection{ID: name, Suggestions: suggestions}
	}

	defaultSQESection := func(prefix string) *mdl.SuggestionSection {
		return section("query",
			fmt.Sprintf("(auth:%s OR receiver:%s)|account_history", prefix, prefix),
			fmt.Sprintf("auth:%s|signed_by", prefix),
			fmt.Sprintf("receiver:eosio.token account:eosio.token action:transfer (data.from:%s OR data.to:%s)|eos_token_transfer", prefix, prefix),
			fmt.Sprintf("data.to:%s|fuzzy_token_search", prefix),
		)
	}

	var emptySections []*mdl.SuggestionSection

	tests := []struct {
		name          string
		prefix        string
		limit         int
		accountNames  []string
		expectedError error
		expected      []*mdl.SuggestionSection
	}{
		// Account like
		{
			"only eos account matches", "eosio.bp", 5, []string{"eosio.bpay"}, nil, []*mdl.SuggestionSection{
				section("accounts", "eosio.bpay"),
				defaultSQESection("eosio.bp"),
			},
		},
		{
			"account like, no match shows sqe fields only", "woot", 5, nil, nil, []*mdl.SuggestionSection{
				defaultSQESection("woot"),
			},
		},
		{
			"account like, followed by space shows nothing", "eosio.bp ", 5, []string{"eosio.bpay"}, nil, emptySections,
		},
		{
			"account like, followed by something else", "eosio.bp testing", 5, []string{"eosio.bpay"}, nil, emptySections,
		},

		// SQE
		{
			"sqe completed field followed by space", "account:value ", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "account:value receiver:", "account:value action:", "account:value data.to:"),
			},
		},

		{
			"sqe complete field name when not account like", "data.producer_", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.producer_key:"),
			},
		},

		// SQE Value Account
		{
			"sqe complete field value account, empty", "account:", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "account:other"),
			},
		},
		{
			"sqe complete field value account, partial match", "account:oth", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "account:other"),
			},
		},
		{
			"sqe complete field value account, no match", "account:aaa", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "account:aaa"),
			},
		},

		// SQE Value Boolean
		{
			"sqe complete field value boolean, empty", "data.is_active:", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.is_active:true", "data.is_active:false"),
			},
		},
		{
			"sqe complete field value boolean, partial true", "data.is_active:tr", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.is_active:true"),
			},
		},
		{
			"sqe complete field value boolean, partial false", "data.is_active:fa", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.is_active:false"),
			},
		},
		{
			"sqe complete field value boolean, no match", "data.is_active:gg", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.is_active:true", "data.is_active:false"),
			},
		},

		// SQE Multi fields
		{
			"sqe multi fields, complete field name", "data.from:eosio data.prox", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "data.from:eosio data.proxy:"),
			},
		},
		{
			"sqe multi fields, complete field value, empty", "data.from:eosio account:", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "data.from:eosio account:other"),
			},
		},
		{
			"sqe multi fields, complete field value, partial", "data.from:eosio account:oth", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "data.from:eosio account:other"),
			},
		},
		{
			"sqe multi fields, complete field value, no match", "data.from:eosio account:aaa", 3, []string{"other"}, nil, []*mdl.SuggestionSection{
				section("query", "data.from:eosio account:aaa"),
			},
		},
		{
			"sqe multi fields, suggest non present field", "account:aaa receiver:bbb ", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "account:aaa receiver:bbb action:", "account:aaa receiver:bbb data.to:", "account:aaa receiver:bbb data.from:"),
			},
		},
		{
			"sqe multi fields, suggest non present field with OR", "(account:a OR auth:b) ", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "(account:a OR auth:b) receiver:", "(account:a OR auth:b) action:", "(account:a OR auth:b) data.to:"),
			},
		},

		// SQE with special expressions
		{
			"sqe special exprs, after (", "(", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "(account:", "(receiver:", "(action:"),
			},
		},
		{
			"sqe special exprs, after OR", "(account:a OR", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "(account:a OR receiver:", "(account:a OR action:", "(account:a OR data.to:"),
			},
		},
		{
			"sqe special exprs, after )", "(account:a)", 3, nil, nil, []*mdl.SuggestionSection{
				section("query", "(account:a) receiver:", "(account:a) action:", "(account:a) data.to:"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var accountNames []string
			if len(test.accountNames) > 0 {
				accountNames = test.accountNames
			}

			convertedAccountNames := make([]string, len(accountNames))
			for i, accountName := range accountNames {
				convertedAccountNames[i] = accountName
			}

			completion := newFromData(convertedAccountNames)
			sections, err := completion.Complete(test.prefix, test.limit)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, test.expected, sections)
		})
	}
}

func TestAddAccount(t *testing.T) {
	completion := newFromData([]string{"eosio"})
	sections, err := completion.Complete("test", 1)
	require.NoError(t, err)

	require.Len(t, sections, 1)
	assert.Truef(t, sections[0].ID != "accounts", "first section should not have been accounts")

	completion.AddAccount("testing")
	sections, err = completion.Complete("test", 1)
	require.NoError(t, err)

	require.Len(t, sections, 2)
	assert.Truef(t, sections[0].ID == "accounts", "first section should have been accounts")
	require.Len(t, sections[0].Suggestions, 1)
	assert.Equal(t, "testing", sections[0].Suggestions[0].Label)
}

func TestSearchAccountNameByPrefix(t *testing.T) {
	tests := []struct {
		name         string
		accountNames []string
		prefix       string
		limit        int
		expected     []string
	}{
		{
			"empty",
			[]string{},
			"any",
			100,
			[]string{},
		},
		{
			"single matching partially",
			[]string{"eos"},
			"e",
			100,
			[]string{"eos"},
		},
		{
			"single matching fully",
			[]string{"eos"},
			"eos",
			100,
			[]string{"eos"},
		},
		{
			"single not matching",
			[]string{"eos"},
			"other",
			100,
			[]string{},
		},
		{
			"single not matching/more than prefix",
			[]string{"eos"},
			"eosa",
			100,
			[]string{},
		},
		{
			"multiple matching",
			[]string{"eos", "abc", "eosio"},
			"eos",
			100,
			[]string{"eos", "eosio"},
		},
		{
			"multiple matching/flush on limit",
			[]string{"eos", "abc", "eosio"},
			"eos",
			2,
			[]string{"eos", "eosio"},
		},
		{
			"multiple matching/more than limit",
			[]string{"eos", "abc", "eosio"},
			"eos",
			1,
			[]string{"eos"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			completion := newFromData(test.accountNames)
			matchingAccountNames := completion.searchAccountNamesByPrefix(test.prefix, test.limit)

			assert.Equal(t, test.expected, matchingAccountNames)
		})
	}
}
