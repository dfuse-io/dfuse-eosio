package tokenmeta

import (
	"encoding/json"
	"testing"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbi_getAccountBalanceFromDBRow(t *testing.T) {
	tests := []struct {
		name        string
		scope       string
		contract    eos.AccountName
		symbol      *eos.Symbol
		dbRow       json.RawMessage
		expectValue *pbtokenmeta.AccountBalance
		expectError bool
	}{
		{
			name:     "simple raw message",
			scope:    "eoscanadacom",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow: []byte("{\"balance\":\"0.0064 EOS\"}"),
			expectValue: &pbtokenmeta.AccountBalance{
				TokenContract: "eosio.token",
				Account:       "eoscanadacom",
				Amount:        64,
				Precision:     4,
				Symbol:        "EOS",
			},
		},
		{
			name:     "invalid raw message",
			scope:    "eoscanadacom",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow:       []byte("{\"bbbbalances\":\"0.0064 EOS\"}"),
			expectError: true,
		},
		{
			name:     "invalid balance in raw message",
			scope:    "eoscanadacom",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow:       []byte("{\"balance\":\"0.0064EOS\"}"),
			expectError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := getAccountBalanceFromDBRow(test.contract, test.symbol, test.scope, test.dbRow)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectValue, value)
			}

		})
	}
}

func TestAbi_getTokenFromDBRow(t *testing.T) {
	tests := []struct {
		name        string
		contract    eos.AccountName
		symbol      *eos.Symbol
		dbRow       json.RawMessage
		expectValue *pbtokenmeta.Token
		expectError bool
	}{
		{
			name:     "updating an existing token",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow: []byte("{\"supply\":\"1074146295.0178 EOS\",\"max_supply\":\"10000000000.0000 EOS\",\"issuer\":\"eosio\"}"),
			expectValue: &pbtokenmeta.Token{
				Contract:      "eosio.token",
				Symbol:        "EOS",
				Precision:     4,
				Issuer:        "eosio",
				MaximumSupply: 100000000000000,
				TotalSupply:   10741462950178,
				Holders:       0,
			},
		},
		{
			name:     "creating a new token",
			contract: "eoscanadaca",
			dbRow:    []byte("{\"supply\":\"52000.00000 WAL\",\"max_supply\":\"100000.00000 WAL\",\"issuer\":\"eoscanadacom\"}"),
			expectValue: &pbtokenmeta.Token{
				Contract:      "eoscanadaca",
				Symbol:        "WAL",
				Precision:     5,
				Issuer:        "eoscanadacom",
				MaximumSupply: 10000000000,
				TotalSupply:   5200000000,
				Holders:       0,
			},
		},
		{
			name:     "invalid raw message",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow:       []byte("{\"balance\":\"234\"}"),
			expectError: true,
		},
		{
			name:     "invalid balance in raw message",
			contract: "eosio.token",
			symbol: &eos.Symbol{
				Precision: 4,
				Symbol:    "EOS",
			},
			dbRow:       []byte("{\"supply\":\"1074146295.0178EOS\",\"max_supply\":\"10000000000.0000 EOS\",\"issuer\":\"eosio\"}"),
			expectError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := getTokenFromDBRow(test.contract, test.symbol, test.dbRow)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectValue, value)
			}

		})
	}
}
