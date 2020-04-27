package codec

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestABICache_AddAndFind(t *testing.T) {
	eosioTokenABI1 := readABI(t, "token.1.abi.json")
	eosioTestABI1 := readABI(t, "test.1.abi.json")
	eosioTestABI2 := readABI(t, "test.2.abi.json")
	eosioNekotABI1 := readABI(t, "nekot.1.abi.json")

	cache := newABICache()
	err := cache.addABI("token", 0, eosioTokenABI1)
	require.NoError(t, err)

	err = cache.addABI("test", 5, eosioTestABI1)
	require.NoError(t, err)

	err = cache.addABI("test", 15, eosioTestABI2)
	require.NoError(t, err)

	err = cache.addABI("test", 12, eosioTestABI1)
	require.Equal(t, errors.New("abi is not sequential against latest ABI's global sequence, latest is 15 and trying to add 12 which is in the past"), err)

	err = cache.addABI("nekot", 12, eosioNekotABI1)
	require.NoError(t, err)

	assert.Equal(t, eosioTokenABI1, cache.findABI("token", 0))
	assert.Equal(t, eosioTokenABI1, cache.findABI("token", 10))
	assert.Equal(t, eosioTokenABI1, cache.findABI("token", 50))

	assert.Nil(t, cache.findABI("test", 0))
	assert.Nil(t, cache.findABI("test", 4))
	assert.Equal(t, eosioTestABI1, cache.findABI("test", 5))
	assert.Equal(t, eosioTestABI1, cache.findABI("test", 14))
	assert.Equal(t, eosioTestABI2, cache.findABI("test", 15))
	assert.Equal(t, eosioTestABI2, cache.findABI("test", 16))
	assert.Equal(t, eosioTestABI2, cache.findABI("test", 50))

	assert.Nil(t, cache.findABI("nekot", 0))
	assert.Equal(t, eosioNekotABI1, cache.findABI("nekot", 12))
	assert.Equal(t, eosioNekotABI1, cache.findABI("nekot", 13))
}

func TestABICache_Truncate(t *testing.T) {
	eosioTestABI1 := readABI(t, "test.1.abi.json")
	eosioTestABI2 := readABI(t, "test.2.abi.json")
	eosioTestABI3 := readABI(t, "test.3.abi.json")
	eosioTokenABI1 := readABI(t, "token.1.abi.json")
	eosioTokenABI2 := readABI(t, "token.2.abi.json")
	eosioNekotABI1 := readABI(t, "nekot.1.abi.json")

	type abiAdder func(cache *ABICache)

	addAbi := func(contract string, globalSequence uint64, abi *eos.ABI) abiAdder {
		return func(cache *ABICache) {
			err := cache.addABI(contract, globalSequence, abi)
			require.NoError(t, err)
		}
	}

	type expectFindAbi struct {
		contract       string
		globalSequence uint64
		abi            *eos.ABI
	}

	tests := []struct {
		name         string
		addAbis      []abiAdder
		truncateAt   uint64
		expectedAbis []expectFindAbi
	}{
		// Empty

		{
			name:         "empty",
			addAbis:      nil,
			truncateAt:   14,
			expectedAbis: nil,
		},

		// Single Contract, Single ABI

		{
			name: "single contract, single abi, truncating exactly on it",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
			},
			truncateAt: 14,
			expectedAbis: []expectFindAbi{
				{"test", 14, nil},
				{"test", 15, nil},
			},
		},
		{
			name: "single contract, single abi, truncating before it",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
			},
			truncateAt: 13,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 14, nil},
				{"test", 15, nil},
			},
		},
		{
			name: "single contract, single abi, truncating after it",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
			},
			truncateAt: 15,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 14, eosioTestABI1},
				{"test", 15, eosioTestABI1},
			},
		},

		// Single Contract, Multiple ABIs

		{
			name: "single contract, multiple abi, truncating none",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
				addAbi("test", 16, eosioTestABI2),
				addAbi("test", 18, eosioTestABI3),
			},
			truncateAt: 19,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 15, eosioTestABI1},
				{"test", 17, eosioTestABI2},
				{"test", 19, eosioTestABI3},
			},
		},
		{
			name: "single contract, multiple abi, truncating all, exactly on",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
				addAbi("test", 16, eosioTestABI2),
				addAbi("test", 18, eosioTestABI3),
			},
			truncateAt: 14,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 15, nil},
				{"test", 17, nil},
				{"test", 19, nil},
			},
		},
		{
			name: "single contract, multiple abi, truncating all, before",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
				addAbi("test", 16, eosioTestABI2),
				addAbi("test", 18, eosioTestABI3),
			},
			truncateAt: 13,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 15, nil},
				{"test", 17, nil},
				{"test", 19, nil},
			},
		},
		{
			name: "single contract, multiple abi, truncating before half",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
				addAbi("test", 16, eosioTestABI2),
				addAbi("test", 18, eosioTestABI3),
			},
			truncateAt: 16,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 15, eosioTestABI1},
				{"test", 17, eosioTestABI1},
				{"test", 19, eosioTestABI1},
			},
		},
		{
			name: "single contract, multiple abi, truncating after half",
			addAbis: []abiAdder{
				addAbi("test", 14, eosioTestABI1),
				addAbi("test", 16, eosioTestABI2),
				addAbi("test", 18, eosioTestABI3),
			},
			truncateAt: 17,
			expectedAbis: []expectFindAbi{
				{"test", 13, nil},
				{"test", 15, eosioTestABI1},
				{"test", 17, eosioTestABI2},
				{"test", 19, eosioTestABI2},
			},
		},

		// Multiple Contracts, Multiple ABIs

		{
			name: "multiple contract, multiple abi, truncate middle",
			addAbis: []abiAdder{
				addAbi("test", 10, eosioTestABI1),
				addAbi("test", 20, eosioTestABI2),
				addAbi("test", 30, eosioTestABI3),

				addAbi("token", 15, eosioTokenABI1),
				addAbi("token", 25, eosioTokenABI2),

				addAbi("nekot", 21, eosioNekotABI1),
			},
			truncateAt: 20,
			expectedAbis: []expectFindAbi{
				{"test", 5, nil},
				{"test", 10, eosioTestABI1},
				{"test", 15, eosioTestABI1},
				{"test", 20, eosioTestABI1},
				{"test", 25, eosioTestABI1},
				{"test", 30, eosioTestABI1},
				{"test", 35, eosioTestABI1},

				{"token", 10, nil},
				{"token", 15, eosioTokenABI1},
				{"token", 20, eosioTokenABI1},
				{"token", 25, eosioTokenABI1},
				{"token", 30, eosioTokenABI1},

				{"nekot", 15, nil},
				{"nekot", 20, nil},
				{"nekot", 25, nil},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := newABICache()

			for _, in := range test.addAbis {
				in(cache)
			}

			cache.truncateAfterOrEqualTo(test.truncateAt)

			for _, expect := range test.expectedAbis {
				if expect.abi == nil {
					assert.Nil(t, cache.findABI(expect.contract, expect.globalSequence))
				} else {
					assert.Equal(t, expect.abi, cache.findABI(expect.contract, expect.globalSequence))
				}
			}
		})
	}
}

func readABI(t *testing.T, abiFile string) (out *eos.ABI) {
	path := path.Join("testdata", "abi", abiFile)
	abiJSON, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	out = new(eos.ABI)
	err = json.Unmarshal(abiJSON, out)
	require.NoError(t, err)

	return
}
