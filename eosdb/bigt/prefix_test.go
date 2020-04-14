// Copyright 2019 dfuse Platform Inc.
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

package bigt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefix_Get(t *testing.T) {
	type call struct {
		id       string
		expected string
		err      error
	}

	tests := []struct {
		prefixes []string
		calls    []call
	}{
		{[]string{"123", "456"}, []call{
			{id: "123456", expected: "123", err: nil},
			{id: "123456", expected: "123", err: nil},
			{id: "456789", expected: "456", err: nil},
			{id: "456789", expected: "456", err: nil},
		}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			m := idToPrefix{}

			for j, call := range test.calls {
				actual, err := m.prefix(test.prefixes, call.id)
				if call.err == nil {
					require.NoError(t, err, "call index #%d", j)
					assert.Equal(t, call.expected, actual, "call index #%d", j)
				} else {
					assert.Equal(t, call.err, err, "call index #%d", j)
				}
			}
		})
	}
}
