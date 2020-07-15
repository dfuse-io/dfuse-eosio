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
