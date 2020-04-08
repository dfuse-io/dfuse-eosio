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

package resolvers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetToString(t *testing.T) {
	tests := []struct {
		name      string
		amount    uint64
		precision uint32
		symbol    string
		args      *AssetArgs
		expectStr string
	}{
		{
			name:      "type asset",
			amount:    1234567897382938,
			precision: 4,
			symbol:    "EOS",
			args: &AssetArgs{
				Format: AssetFormatAsset,
			},
			expectStr: "123456789738.2938 EOS",
		},
		{
			name:      "type integer",
			amount:    1234567897382938,
			precision: 4,
			symbol:    "EOS",
			args: &AssetArgs{
				Format: AssetFormatInteger,
			},
			expectStr: "1234567897382938",
		},
		{
			name:      "type decimal",
			amount:    1234567897382938,
			precision: 4,
			symbol:    "EOS",
			args: &AssetArgs{
				Format: AssetFormatDecimal,
			},
			expectStr: "123456789738.2938",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			str := assetToString(test.amount, test.precision, test.symbol, test.args)
			assert.Equal(t, test.expectStr, str)
		})
	}

}
