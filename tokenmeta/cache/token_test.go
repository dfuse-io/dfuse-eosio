package cache

import (
	"testing"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/stretchr/testify/assert"
)

func Test_SortTokensSymbolAlpha(t *testing.T) {
	tests := []struct {
		name         string
		tokens       []*pbtokenmeta.Token
		order        SortingOrder
		expectTokens []*pbtokenmeta.Token
	}{
		{
			name:  "ascending sort",
			order: ASC,
			tokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
			expectTokens: []*pbtokenmeta.Token{
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
			},
		},
		{
			name:  "descending sort",
			order: DESC,
			tokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
			expectTokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectTokens, SortTokensBySymbolAlpha(test.tokens, test.order))
		})
	}
}

func Test_SortTokensHolderCount(t *testing.T) {
	tests := []struct {
		name         string
		tokens       []*pbtokenmeta.Token
		order        SortingOrder
		expectTokens []*pbtokenmeta.Token
	}{
		{
			name:  "ascending sort",
			order: ASC,
			tokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
			expectTokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
		},
		{
			name:  "descending sort",
			order: DESC,
			tokens: []*pbtokenmeta.Token{
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
			},
			expectTokens: []*pbtokenmeta.Token{
				{Contract: "eidosonecoin", Symbol: "EIDOS", Holders: 15},
				{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
				{Contract: "eosio.token", Symbol: "WAL", Holders: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectTokens, SortTokensByHolderCount(test.tokens, test.order))
		})
	}
}

// TODO: implement me
func Test_SortTokensMarketCap(t *testing.T) {
}

func Test_compareTokenBySymbol(t *testing.T) {
	tests := []struct {
		name        string
		a           *pbtokenmeta.Token
		b           *pbtokenmeta.Token
		order       SortingOrder
		expectValue bool
	}{
		{
			name:        "ascending a before b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			expectValue: true,
		},
		{
			name:        "ascending a after b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			expectValue: false,
		},
		{
			name:        "ascending a equal b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "TNT", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "TNT", Holders: 8},
			expectValue: true,
		},
		{
			name:        "descending a before b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			expectValue: true,
		},
		{
			name:        "descending a after b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			expectValue: false,
		},
		{
			name:        "descending a equal b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "TNT", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "TNT", Holders: 8},
			expectValue: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectValue, compareTokenBySymbol(test.a, test.b, test.order))
		})
	}
}

func Test_compareTokenByContract(t *testing.T) {
	tests := []struct {
		name        string
		a           *pbtokenmeta.Token
		b           *pbtokenmeta.Token
		order       SortingOrder
		expectValue bool
	}{
		{
			name:        "ascending a before b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			expectValue: true,
		},
		{
			name:        "ascending a after b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 8},
			expectValue: false,
		},
		{
			name:        "ascending a equal b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			expectValue: true,
		},
		{
			name:        "descending a before b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 8},
			expectValue: true,
		},
		{
			name:        "descending a after b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			expectValue: false,
		},
		{
			name:        "descending a equal b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			expectValue: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectValue, compareTokenByContract(test.a, test.b, test.order))
		})
	}
}

func Test_compareTokenByHolders(t *testing.T) {
	tests := []struct {
		name        string
		a           *pbtokenmeta.Token
		b           *pbtokenmeta.Token
		order       SortingOrder
		expectValue bool
	}{
		{
			name:        "ascending a before b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 3},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			expectValue: true,
		},
		{
			name:        "ascending a after b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 4},
			expectValue: false,
		},
		{
			name:        "ascending a equal b",
			order:       ASC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "WAL", Holders: 8},
			expectValue: true,
		},
		{
			name:        "descending a before b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 3},
			expectValue: true,
		},
		{
			name:        "descending a after b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 3},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 8},
			expectValue: false,
		},
		{
			name:        "descending a equal b",
			order:       DESC,
			a:           &pbtokenmeta.Token{Contract: "eosio.token", Symbol: "EOS", Holders: 8},
			b:           &pbtokenmeta.Token{Contract: "bonecoin.en", Symbol: "WAL", Holders: 8},
			expectValue: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectValue, compareTokenByHolders(test.a, test.b, test.order))
		})
	}
}
