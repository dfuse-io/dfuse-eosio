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

package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeEvent(t *testing.T) {
	tests := []struct {
		name         string
		unrestricted bool
		key          string
		data         string
		expect       string
	}{
		{
			name:         "valid 2 kv",
			unrestricted: false,
			data:         "key1=value&key2=value2",
			expect:       "key1=value&key2=value2",
		},
		{
			name:         "valid with 2 empty values",
			unrestricted: false,
			data:         "key1=value&key2&key3",
			expect:       "key1=value&key2=&key3=",
		},
		{
			name:         "key longer than 16 chars",
			unrestricted: false,
			data:         "keyislongerthan16characters=value",
			expect:       "",
		},
		{
			name:         "value longer than 64 chars, ditch it all",
			unrestricted: false,
			data:         "key=fieldislongerwaylongerthan64characterswhichishardtypeandensurewearegood&key2=ok",
			expect:       "",
		},
		{
			name:         "more than 3 fields, ditch it all",
			unrestricted: false,
			data:         "key1=value1&key2=value2&key3=value3&key4=value4",
			expect:       "",
		},
		{
			name:         "spaces in key is not ditched?",
			unrestricted: false,
			data:         "spaced key=value",
			expect:       "spaced+key=value", // hmm.. perhaps we can revise this.. and allow only restricted characters in the field name.
		},

		// Unrestricted tests
		{
			name:         "unrestricted, key longer than 16 chars",
			unrestricted: true,
			data:         "keyislongerthan16characters=value",
			expect:       "keyislongerthan16characters=value",
		},
		{
			name:         "unrestricted, more than 3 fields",
			unrestricted: true,
			data:         "key1=value1&key2=value2&key3=value3&key4=value4",
			expect:       "key1=value1&key2=value2&key3=value3&key4=value4",
		},
		{
			name:         "unrestricted, value longer than 64 chars",
			unrestricted: true,
			data:         "key=fieldislongerwaylongerthan64characterswhichishardtypeandensurewearegood&key2=ok",
			expect:       "key=fieldislongerwaylongerthan64characterswhichishardtypeandensurewearegood&key2=ok",
		},
	}

	tokenizer := tokenizer{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := tokenizer.tokenizeEvent(eventsConfig{actionName: "", unrestricted: test.unrestricted}, test.key, test.data)
			assert.Equal(t, test.expect, res.Encode())
		})
	}
}
