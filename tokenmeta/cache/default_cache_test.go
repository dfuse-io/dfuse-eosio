package cache

import (
	"testing"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/bstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCache_AccountBalances(t *testing.T) {
	tests := []*struct {
		name              string
		accountName       eos.AccountName
		balances          map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		eosStake          map[eos.AccountName]*EOSStake
		expectOwnedAssets []*OwnedAsset
		options           []AccountBalanceOption
	}{
		{
			name:        "owner with one token in one contract",
			accountName: eos.AccountName("eoscanadadad"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
					eos.AccountName("johndoemyhero"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(29371, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:        "owner with one token in two contract",
			accountName: eos.AccountName("eoscanadadad"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
				eos.AccountName("abababababa"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(200, "WALL"),
								Contract: "abababababa",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(200, "WALL"),
						Contract: "abababababa",
					},
				},
			},
		},
		{
			name:        "owner with two tokens in one contract",
			accountName: eos.AccountName("eoscanadadad"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(3, "WALL"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(3, "WALL"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:        "poor owner without any assets",
			accountName: eos.AccountName("johndoemyone"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
		},
		{
			name:        "owner with staked EOS",
			accountName: eos.AccountName("eoscanadadad"),
			eosStake: map[eos.AccountName]*EOSStake{
				eos.AccountName("eoscanadadad"): {
					TotalNet: eos.Int64(24),
					TotalCpu: eos.Int64(14),
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("eoscanadadad"): {
							To:   "eoscanadadad",
							From: "eoscanadadad",
							Net:  eos.Int64(24),
							Cpu:  eos.Int64(14),
						},
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(111, "WAX"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			options: []AccountBalanceOption{
				EOSIncludeStakedAccOpt,
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(111, "WAX"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(138, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:        "owner without staked EOS, requested",
			accountName: eos.AccountName("eoscanadadad"),
			eosStake:    map[eos.AccountName]*EOSStake{},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			options: []AccountBalanceOption{
				EOSIncludeStakedAccOpt,
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:        "owner with unwanted staked EOS",
			accountName: eos.AccountName("eoscanadadad"),
			eosStake: map[eos.AccountName]*EOSStake{
				eos.AccountName("eoscanadadad"): {

					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("eoscanadadad"): {
							To:   "eoscanadad",
							From: "eoscanadadad",
							Net:  eos.Int64(24),
							Cpu:  eos.Int64(14),
						},
					},
				},
			},

			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				Balances: test.balances,
				EOSStake: test.eosStake,
			}
			ownedAssets := cache.AccountBalances(test.accountName, test.options...)
			assert.ElementsMatch(t, test.expectOwnedAssets, ownedAssets)
		})
	}
}

func TestDefaultCache_TokenBalances(t *testing.T) {
	tests := []*struct {
		name              string
		contract          eos.AccountName
		balances          map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		expectOwnedAssets []*OwnedAsset
		eosStake          map[eos.AccountName]*EOSStake
		options           []TokenBalanceOption
	}{
		{
			name:     "contract with multiple users and tokens",
			contract: eos.AccountName("eosio.token"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(3, "WALL"),
								Contract: "eosio.token",
							},
						},
					},
					eos.AccountName("johndoeonecoin"): {
						{
							Owner: eos.AccountName("johndoeonecoin"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(23927, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(100, "EOS"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(3, "WALL"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("johndoeonecoin"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(23927, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:     "eosio.token with staked",
			contract: eos.AccountName("eosio.token"),
			options: []TokenBalanceOption{
				EOSIncludeStakedTokOpt,
			},
			eosStake: map[eos.AccountName]*EOSStake{
				eos.AccountName("eoscanadadad"): {
					TotalCpu: eos.Int64(200000000),
					TotalNet: eos.Int64(100000000),
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("eoscanadadad"): {
							To:   "eoscanadad",
							From: "eoscanadadad",
							Net:  eos.Int64(100000000),
							Cpu:  eos.Int64(200000000),
						},
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(3, "WALL"),
								Contract: "eosio.token",
							},
						},
					},
					eos.AccountName("johndoeonecoin"): {
						{
							Owner: eos.AccountName("johndoeonecoin"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(23927, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
			expectOwnedAssets: []*OwnedAsset{
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(300000100, "EOS"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("eoscanadadad"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(3, "WALL"),
						Contract: "eosio.token",
					},
				},
				{
					Owner: eos.AccountName("johndoeonecoin"),
					Asset: &eos.ExtendedAsset{
						Asset:    generateTestAsset(23927, "EOS"),
						Contract: "eosio.token",
					},
				},
			},
		},
		{
			name:     "contract does not exists",
			contract: eos.AccountName("eidoeonecoin"),
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				eos.AccountName("eosio.token"): {
					eos.AccountName("eoscanadadad"): {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: "eosio.token",
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				Balances: test.balances,
				EOSStake: test.eosStake,
			}
			assert.ElementsMatch(t, test.expectOwnedAssets, cache.TokenBalances(test.contract, test.options...))
		})
	}
}

func TestDefaultCache_IsTokenContract(t *testing.T) {
	tests := []struct {
		name        string
		contract    eos.AccountName
		tokens      map[eos.AccountName][]*pbtokenmeta.Token
		expectValue bool
	}{
		{
			name:        "contract is not cached",
			contract:    eos.AccountName("eosio.token"),
			tokens:      map[eos.AccountName][]*pbtokenmeta.Token{},
			expectValue: false,
		},
		{
			name:     "contract is cached",
			contract: eos.AccountName("eosio.token"),
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {},
			},
			expectValue: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				TokensInContract: test.tokens,
			}
			assert.Equal(t, test.expectValue, cache.IsTokenContract(test.contract))
		})
	}

}

func TestDefaultCache_hasSymbolForContract(t *testing.T) {
	tests := []struct {
		name        string
		contract    eos.AccountName
		symbol      string
		tokens      map[eos.AccountName][]*pbtokenmeta.Token
		expectValue bool
	}{
		{
			name:     "contract and symbol exists",
			contract: eos.AccountName("eosio.token"),
			symbol:   "EOS",
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol: "EOS",
					},
				},
			},
			expectValue: true,
		},
		{
			name:        "contract does not exists",
			contract:    eos.AccountName("eosio.token"),
			tokens:      map[eos.AccountName][]*pbtokenmeta.Token{},
			expectValue: false,
		},
		{
			name:     "contract exists but symbol does not exists",
			contract: eos.AccountName("eosio.token"),
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol: "WAX",
					},
				},
			},
			expectValue: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				TokensInContract: test.tokens,
			}
			assert.Equal(t, test.expectValue, cache.hasSymbolForContract(test.contract, test.symbol))
		})
	}

}

func TestDefaultCache_setBalance(t *testing.T) {
	tests := []struct {
		name           string
		asset          *OwnedAsset
		tokens         map[eos.AccountName][]*pbtokenmeta.Token
		balances       map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		expectBalances map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		expectTokens   map[eos.AccountName][]*pbtokenmeta.Token
		expectError    bool
	}{
		{
			name: "sunny path",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol: "EOS",
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
		},
		{
			name: "sunny path when account already seen",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 0,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
		},
		{
			name: "set a new balance for a non existing contract",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens:         map[eos.AccountName][]*pbtokenmeta.Token{},
			balances:       map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{},
			expectTokens:   map[eos.AccountName][]*pbtokenmeta.Token{},
			expectError:    true,
		},
		{
			name: "set a new balance for a non-existing token",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {},
			},
			balances:       map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {},
			},
			expectError: true,
		},
		{
			name: "change an existing balance a new balance",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(20, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
		},
		{
			name: "should only change the specfic contract",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(100, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{Symbol: "EOS", Holders: 1},
				},
				eos.AccountName("eidosonecoin"): {
					{Symbol: "EOS", Holders: 1},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(20, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
				"eidosonecoin": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(30, "EOS"),
								Contract: eos.AccountName("eidosonecoin"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
				"eidosonecoin": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(30, "EOS"),
								Contract: eos.AccountName("eidosonecoin"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{Symbol: "EOS", Holders: 1},
				},
				eos.AccountName("eidosonecoin"): {
					{Symbol: "EOS", Holders: 1},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				TokensInContract: test.tokens,
				Balances:         test.balances,
			}
			err := cache.setBalance(test.asset)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectBalances, cache.Balances)
			assert.Equal(t, test.expectTokens, cache.TokensInContract)

		})
	}

}

func TestDefaultCache_removeBalance(t *testing.T) {
	tests := []struct {
		name           string
		asset          *OwnedAsset
		tokens         map[eos.AccountName][]*pbtokenmeta.Token
		balances       map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		expectBalances map[eos.AccountName]map[eos.AccountName][]*OwnedAsset
		expectTokens   map[eos.AccountName][]*pbtokenmeta.Token
		expectError    bool
	}{
		{
			name: "remove an existing balance",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 0,
					},
				},
			},
		},
		{
			name: "remove an existing balance while maintaining another token",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
					{
						Symbol:  "WALL",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(65, "WALL"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(65, "WALL"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 0,
					},
					{
						Symbol:  "WALL",
						Holders: 1,
					},
				},
			},
		},
		{
			name: "remove a non existing balance",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "EOS"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"lelapinblanc": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"lelapinblanc": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			expectError: true,
		},
		{
			name: "should only change the specific contract",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "EOS"),
					Contract: eos.AccountName("eidosonecoin"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
				eos.AccountName("eidosonecoin"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(20, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
				"eidosonecoin": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(30, "EOS"),
								Contract: eos.AccountName("eidosonecoin"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(20, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
				"eidosonecoin": {},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
				eos.AccountName("eidosonecoin"): {
					{
						Symbol:  "EOS",
						Holders: 0,
					},
				},
			},
		},
		{
			name: "remove balance for a non cached contract",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "EOS"),
					Contract: eos.AccountName("abababababa"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			expectError: true,
		},
		{
			name: "remove balance for non existing token symbol",
			asset: &OwnedAsset{
				Owner: eos.AccountName("eoscanadadad"),
				Asset: &eos.ExtendedAsset{
					Asset:    generateTestAsset(0, "WAL"),
					Contract: eos.AccountName("eosio.token"),
				},
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			balances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectBalances: map[eos.AccountName]map[eos.AccountName][]*OwnedAsset{
				"eosio.token": {
					"eoscanadadad": {
						{
							Owner: eos.AccountName("eoscanadadad"),
							Asset: &eos.ExtendedAsset{
								Asset:    generateTestAsset(100, "EOS"),
								Contract: eos.AccountName("eosio.token"),
							},
						},
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Symbol:  "EOS",
						Holders: 1,
					},
				},
			},
			expectError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				TokensInContract: test.tokens,
				Balances:         test.balances,
			}
			err := cache.removeBalance(test.asset)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectBalances, cache.Balances)
			assert.Equal(t, test.expectTokens, cache.TokensInContract)
		})
	}

}

func TestDefaultCache_setToken(t *testing.T) {
	asset := generateTestAsset(1000000, "EOS")
	biggerAsset := generateTestAsset(20000000, "EOS")
	tests := []struct {
		name         string
		token        *pbtokenmeta.Token
		tokens       map[eos.AccountName][]*pbtokenmeta.Token
		expectTokens map[eos.AccountName][]*pbtokenmeta.Token
	}{
		{
			name: "sunny path",
			token: &pbtokenmeta.Token{
				Contract:      "eosio.token",
				Symbol:        "EOS",
				Issuer:        "eosio.token",
				MaximumSupply: uint64(asset.Amount),
				Precision:     4,
				TotalSupply:   uint64(asset.Amount),
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
					},
				},
			},
		},
		{
			name: "update token",
			token: &pbtokenmeta.Token{
				Contract:      "eosio.token",
				Symbol:        "EOS",
				Issuer:        "eosio.token",
				MaximumSupply: uint64(biggerAsset.Amount),
				Precision:     4,
				TotalSupply:   uint64(biggerAsset.Amount),
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
						Holders:       13,
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(biggerAsset.Amount),
						Precision:     4,
						TotalSupply:   uint64(biggerAsset.Amount),
						Holders:       13,
					},
				},
			},
		},
		{
			name: "add token to existing contract",
			token: &pbtokenmeta.Token{
				Contract:      "eosio.token",
				Symbol:        "WALL",
				Issuer:        "eosio.token",
				MaximumSupply: uint64(biggerAsset.Amount),
				Precision:     4,
				TotalSupply:   uint64(biggerAsset.Amount),
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
						Holders:       13,
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
						Holders:       13,
					},
					{
						Contract:      "eosio.token",
						Symbol:        "WALL",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(biggerAsset.Amount),
						Precision:     4,
						TotalSupply:   uint64(biggerAsset.Amount),
						Holders:       0,
					},
				},
			},
		},
		{
			name: "add token and contract",
			token: &pbtokenmeta.Token{
				Contract:      "eidosonecoin",
				Symbol:        "EIDOS",
				Issuer:        "eidosonecoin",
				MaximumSupply: uint64(biggerAsset.Amount),
				Precision:     4,
				TotalSupply:   uint64(biggerAsset.Amount),
			},
			tokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
						Holders:       7,
					},
				},
			},
			expectTokens: map[eos.AccountName][]*pbtokenmeta.Token{
				eos.AccountName("eidosonecoin"): {
					{
						Contract:      "eidosonecoin",
						Symbol:        "EIDOS",
						Issuer:        "eidosonecoin",
						MaximumSupply: uint64(biggerAsset.Amount),
						Precision:     4,
						TotalSupply:   uint64(biggerAsset.Amount),
						Holders:       0,
					},
				},
				eos.AccountName("eosio.token"): {
					{
						Contract:      "eosio.token",
						Symbol:        "EOS",
						Issuer:        "eosio.token",
						MaximumSupply: uint64(asset.Amount),
						Precision:     4,
						TotalSupply:   uint64(asset.Amount),
						Holders:       7,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				TokensInContract: test.tokens,
			}
			err := cache.setToken(test.token)
			require.NoError(t, err)
			assert.Equal(t, test.expectTokens, cache.TokensInContract)
		})
	}

}

func TestDefaultCache_Stake(t *testing.T) {
	tests := []*struct {
		name           string
		stakeEntries   []*EOSStakeEntry
		expectStakeMap map[eos.AccountName]*EOSStake
	}{
		{
			name: "golden",
			stakeEntries: []*EOSStakeEntry{
				{
					To:   "b1",
					From: "b1",
					Net:  10000,
					Cpu:  15000,
				},
				{
					To:   "b2",
					From: "b2",
					Net:  20000,
					Cpu:  25000,
				},
				{
					To:   "b3",
					From: "b1",
					Net:  7000,
					Cpu:  13000,
				},
			},
			expectStakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 17000,
					TotalCpu: 28000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  10000,
							Cpu:  15000,
						},
						eos.AccountName("b3"): {
							To:   "b3",
							From: "b1",
							Net:  7000,
							Cpu:  13000,
						},
					},
				},
				eos.AccountName("b2"): {
					TotalNet: 20000,
					TotalCpu: 25000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b2",
							Net:  20000,
							Cpu:  25000,
						},
					},
				},
			},
		},
		{
			name: "modify",
			stakeEntries: []*EOSStakeEntry{
				{
					To:   "b1",
					From: "b1",
					Net:  10000,
					Cpu:  15000,
				},
				{
					To:   "b2",
					From: "b2",
					Net:  20000,
					Cpu:  25000,
				},
				{
					To:   "b1",
					From: "b1",
					Net:  0,
					Cpu:  13000,
				},
			},
			expectStakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 0,
					TotalCpu: 13000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  0,
							Cpu:  13000,
						},
					},
				},
				eos.AccountName("b2"): {
					TotalNet: 20000,
					TotalCpu: 25000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b2",
							Net:  20000,
							Cpu:  25000,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			muts := &MutationsBatch{}
			for _, stakeEntry := range test.stakeEntries {
				muts.SetStake(stakeEntry)
			}
			cache := &DefaultCache{
				EOSStake: make(map[eos.AccountName]*EOSStake),
			}
			cache.Apply(muts, bstream.NewBlockRef("10a", 10))
			assert.EqualValues(t, test.expectStakeMap, cache.EOSStake)
		})
	}
}

func TestDefaultCache_getStakeForAccount(t *testing.T) {
	tests := []*struct {
		name             string
		account          eos.AccountName
		stakeMap         map[eos.AccountName]*EOSStake
		expectStakeValue int64
	}{
		{
			name:    "golden",
			account: eos.AccountName("b1"),
			stakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 1700,
					TotalCpu: 2800,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  10000,
							Cpu:  15000,
						},
						eos.AccountName("b3"): {
							To:   "b3",
							From: "b1",
							Net:  7000,
							Cpu:  13000,
						},
					},
				},
				eos.AccountName("b2"): {
					TotalNet: 20000,
					TotalCpu: 25000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b2",
							Net:  20000,
							Cpu:  25000,
						},
					},
				},
			},
			expectStakeValue: 4500,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				EOSStake: test.stakeMap,
			}
			assert.EqualValues(t, test.expectStakeValue, cache.getStakeForAccount(test.account))
		})
	}
}

func TestDefaultCache_setStake(t *testing.T) {
	tests := []*struct {
		name           string
		stakeEntry     *EOSStakeEntry
		stakeMap       map[eos.AccountName]*EOSStake
		expectStakeMap map[eos.AccountName]*EOSStake
	}{
		{
			name: "no stake entry present",
			stakeEntry: &EOSStakeEntry{
				To:   "b2",
				From: "b1",
				Net:  1200,
				Cpu:  2400,
			},
			stakeMap: map[eos.AccountName]*EOSStake{},
			expectStakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 1200,
					TotalCpu: 2400,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b1",
							Net:  1200,
							Cpu:  2400,
						},
					},
				},
			},
		},
		{
			name: "stake entry present with a different stake.to",
			stakeEntry: &EOSStakeEntry{
				To:   "b1",
				From: "b1",
				Net:  70000,
				Cpu:  140000,
			},
			stakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 1200,
					TotalCpu: 2400,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b1",
							Net:  1200,
							Cpu:  2400,
						},
					},
				},
			},
			expectStakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 71200,
					TotalCpu: 142400,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b2"): {
							To:   "b2",
							From: "b1",
							Net:  1200,
							Cpu:  2400,
						},
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  70000,
							Cpu:  140000,
						},
					},
				},
			},
		},
		{
			name: "stake entry present with an already exisiting stake.to",
			stakeEntry: &EOSStakeEntry{
				To:   "b1",
				From: "b1",
				Net:  70000,
				Cpu:  140000,
			},
			stakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 1200,
					TotalCpu: 2400,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  1200,
							Cpu:  2400,
						},
					},
				},
			},
			expectStakeMap: map[eos.AccountName]*EOSStake{
				eos.AccountName("b1"): {
					TotalNet: 70000,
					TotalCpu: 140000,
					Entries: map[eos.AccountName]*EOSStakeEntry{
						eos.AccountName("b1"): {
							To:   "b1",
							From: "b1",
							Net:  70000,
							Cpu:  140000,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &DefaultCache{
				EOSStake: test.stakeMap,
			}
			cache.setStake(test.stakeEntry)
			assert.EqualValues(t, test.expectStakeMap, cache.EOSStake)
		})
	}
}

func generateTestAsset(amount eos.Int64, symbol string) eos.Asset {
	return eos.Asset{
		Amount: amount,
		Symbol: *generateTestSymbol(symbol),
	}
}

func generateTestSymbol(symbol string) *eos.Symbol {
	return &eos.Symbol{
		Precision: 4,
		Symbol:    symbol,
	}
}
