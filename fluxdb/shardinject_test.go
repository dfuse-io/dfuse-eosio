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

package fluxdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseFilename(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expectFirst uint32
		expectLast  uint32
		expectError bool
	}{
		{
			name:        "simple",
			in:          "0103500000-0103999999",
			expectFirst: 103500000,
			expectLast:  103999999,
			expectError: false,
		},
		{
			name:        "not-numbers",
			in:          "0103500000x-0234252444y",
			expectFirst: 0,
			expectLast:  0,
			expectError: true,
		},
		{
			name:        "not-enough",
			in:          "0103500000",
			expectFirst: 0,
			expectLast:  0,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, l, e := parseFileName(test.in)
			assert.Equal(t, test.expectFirst, f)
			assert.Equal(t, test.expectLast, l)
			if test.expectError {
				assert.Error(t, e)
			} else {
				assert.NoError(t, e)
			}
		})
	}

}
