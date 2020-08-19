package tokenmeta

import (
	"testing"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
)

func Test_tokenMatchFilters(t *testing.T) {
	tests := []struct {
		name           string
		token          *pbtokenmeta.Token
		contractFilter []string
		symbolFilter   []string
		expectMatch    bool
	}{
		{
			name: "without any filters",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			expectMatch: true,
		},
		{
			name: "with a non-matching contract filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin"},
			expectMatch:    false,
		},
		{
			name: "with a matching contract filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin", "eosio.token"},
			expectMatch:    true,
		},
		{
			name: "with a non-matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			symbolFilter: []string{"WAL"},
			expectMatch:  false,
		},
		{
			name: "with a matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			symbolFilter: []string{"WAL", "EOS"},
			expectMatch:  true,
		},
		{
			name: "with a non-matching contract filter & non-matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin"},
			symbolFilter:   []string{"WAL"},
			expectMatch:    false,
		},
		{
			name: "with a non-matching contract filter & matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin"},
			symbolFilter:   []string{"EOS", "WALL"},
			expectMatch:    false,
		},
		{
			name: "with a matching contract filter & non-matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin", "eosio.token"},
			symbolFilter:   []string{"WALL"},
			expectMatch:    false,
		},
		{
			name: "with a matching contract filter & matching symbol filter",
			token: &pbtokenmeta.Token{
				Contract:  "eosio.token",
				Symbol:    "EOS",
				Precision: 4,
			},
			contractFilter: []string{"eidosonecoin", "eosio.token"},
			symbolFilter:   []string{"WALL", "EOS"},
			expectMatch:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectMatch, matchFilters(eos.AccountName(test.token.Contract), test.token.Symbol, test.contractFilter, test.symbolFilter))
		})
	}
}

func Test_sortOrderMapper(t *testing.T) {
	tests := []struct {
		name        string
		order       pbtokenmeta.SortOrder
		expectOrder cache.SortingOrder
	}{
		{
			name:        "ascending",
			order:       pbtokenmeta.SortOrder_ASC,
			expectOrder: cache.ASC,
		},
		{
			name:        "descending",
			order:       pbtokenmeta.SortOrder_DESC,
			expectOrder: cache.DESC,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectOrder, sortOrderMapper(test.order))
		})
	}
}

func Test_limitResults(t *testing.T) {
	tests := []struct {
		name         string
		results      []*pbtokenmeta.Token
		limit        uint32
		expectResult []*pbtokenmeta.Token
		symbolFilter []string
		expectMatch  bool
	}{
		{
			name: "limit 0",
			results: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 8},
			},
			limit: 0,
			expectResult: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 8},
			},
		},
		{
			name: "limit less then results set length",
			results: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 8},
			},
			limit: 2,
			expectResult: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			},
		},
		{
			name: "limit greater then results set length",
			results: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 8},
			},
			limit: 10,
			expectResult: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 8},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectResult, limitTokenResults(test.results, test.limit))
		})
	}

}
func generateTestSymbol(symbol string) *eos.Symbol {
	return &eos.Symbol{
		Precision: 4,
		Symbol:    symbol,
	}
}

func generateTestAsset(amount eos.Int64, symbol string) eos.Asset {
	return eos.Asset{
		Amount: amount,
		Symbol: *generateTestSymbol(symbol),
	}
}
