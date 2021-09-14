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

package abicodec

import (
	"context"
	"encoding/hex"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/logging"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func NewTestABI(version string) *eos.ABI {
	return &eos.ABI{Version: version}
}

func TestABICache_SetABIAtBlockNum(t *testing.T) {
	testCases := []struct {
		name               string
		items              map[string][]*ABICacheItem
		account            string
		version            string
		blockNum           uint32
		expectedABIAtIndex int
		expectedVersion    string
		expectedCacheSize  int
	}{
		{
			name:               "first one",
			items:              map[string][]*ABICacheItem{},
			account:            "account.1",
			version:            "version.1",
			blockNum:           2,
			expectedABIAtIndex: 0,
			expectedVersion:    "version.1",
			expectedCacheSize:  1,
		},
		{
			name: "Second one",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{
						BlockNum: 1,
						ABI:      NewTestABI("version.1"),
					},
				},
			},
			account:            "account.1",
			version:            "version.2",
			blockNum:           2,
			expectedABIAtIndex: 1,
			expectedVersion:    "version.2",
			expectedCacheSize:  2,
		},
		{
			name: "To middle",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{
						BlockNum: 1,
						ABI:      NewTestABI("version.1"),
					},
					{
						BlockNum: 3,
						ABI:      NewTestABI("version.3"),
					},
				},
			},
			account:            "account.1",
			version:            "version.2",
			blockNum:           2,
			expectedABIAtIndex: 1,
			expectedVersion:    "version.2",
			expectedCacheSize:  3,
		},
		{
			name: "To the end",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{
						BlockNum: 1,
						ABI:      NewTestABI("version.1"),
					},
					{
						BlockNum: 3,
						ABI:      NewTestABI("version.2"),
					},
				},
			},
			account:            "account.1",
			version:            "version.3",
			blockNum:           10,
			expectedABIAtIndex: 2,
			expectedVersion:    "version.3",
			expectedCacheSize:  3,
		},
		{
			name: "Replace",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{
						BlockNum: 1,
						ABI:      NewTestABI("version.1"),
					},
					{
						BlockNum: 3,
						ABI:      NewTestABI("version.2"),
					},
				},
			},
			account:            "account.1",
			version:            "version.3",
			blockNum:           3,
			expectedABIAtIndex: 1,
			expectedVersion:    "version.3",
			expectedCacheSize:  2,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			store, err := dstore.NewSimpleStore("file:///tmp/cache")
			require.NoError(t, err)
			cache, err := NewABICache(store, "test_cache.bin")
			require.NoError(t, err)

			cache.Abis = c.items
			cache.SetABIAtBlockNum(c.account, c.blockNum, NewTestABI(c.version))
			assert.Equal(t, c.expectedVersion, cache.Abis[c.account][c.expectedABIAtIndex].ABI.Version)
			assert.Equal(t, c.expectedCacheSize, len(cache.Abis[c.account]))
		})
	}

}

func TestDefaultCache_ABIAtBlockNum(t *testing.T) {
	testCases := []struct {
		name            string
		items           map[string][]*ABICacheItem
		expectedVersion string
		expectNil       bool
		atBlock         uint32
	}{
		{
			name: "Last",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(400),
			expectedVersion: "version.3",
		},
		{
			name: "right on",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(100),
			expectedVersion: "version.1",
		},
		{
			name: "between",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(250),
			expectedVersion: "version.2",
		},
		{
			name: "missing",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(99),
			expectNil:       true,
			expectedVersion: "version.2",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			store, err := dstore.NewSimpleStore("file:///tmp/cache")
			require.NoError(t, err)
			cache, err := NewABICache(store, "test_cache.bin")
			require.NoError(t, err)

			cache.Abis = c.items
			abiItem := cache.ABIAtBlockNum("account.1", c.atBlock)
			if c.expectNil {
				require.Nil(t, abiItem)
				return
			}
			require.Equal(t, c.expectedVersion, abiItem.ABI.Version)
		})
	}

}

func TestDefaultCache_RemoveABIAtBlockNum(t *testing.T) {
	testCases := []struct {
		name            string
		items           map[string][]*ABICacheItem
		expectedVersion string
		expectNil       bool
		atBlock         uint32
	}{
		{
			name: "Last",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(300),
			expectedVersion: "version.2",
		},
		{
			name: "middle",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(200),
			expectedVersion: "version.1",
		},
		{
			name: "first",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:   uint32(100),
			expectNil: true,
		},
		{
			name: "over",
			items: map[string][]*ABICacheItem{
				"account.1": {
					{BlockNum: 100, ABI: NewTestABI("version.1")},
					{BlockNum: 200, ABI: NewTestABI("version.2")},
					{BlockNum: 300, ABI: NewTestABI("version.3")},
				},
			},
			atBlock:         uint32(400),
			expectedVersion: "version.3",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			store, err := dstore.NewSimpleStore("file:///tmp/cache")
			require.NoError(t, err)
			cache, err := NewABICache(store, "test_cache.bin")
			require.NoError(t, err)
			cache.Abis = c.items
			cache.RemoveABIAtBlockNum("account.1", c.atBlock)
			abiItem := cache.ABIAtBlockNum("account.1", c.atBlock)
			if c.expectNil {
				require.Nil(t, abiItem)
				return
			}
			require.Equal(t, c.expectedVersion, abiItem.ABI.Version)
		})
	}

}

func TestDefaultCache_Save_Load(t *testing.T) {
	cacheName := "test_cache.bin"
	ctx := context.Background()

	store, err := dstore.NewSimpleStore("file:///tmp")
	require.NoError(t, err)

	exist, err := store.FileExists(ctx, cacheName)
	require.NoError(t, err)
	if exist {
		err := store.DeleteObject(ctx, cacheName)
		require.NoError(t, err)
	}

	require.NoError(t, err)
	cache, err := NewABICache(store, cacheName)
	require.NoError(t, err)

	abiData, err := hex.DecodeString("0e656f73696f3a3a6162692f312e300110657468657265756d5f6164647265737306737472696e6702076164647265737300030269640675696e74363410657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e636505617373657403616464000210657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e63650561737365740100000000000052320361646400010000c00a637553320369363401026964010675696e7436340761646472657373000000")
	require.NoError(t, err)

	var abi *eos.ABI
	err = eos.UnmarshalBinary(abiData, &abi)
	require.NoError(t, err)

	spew.Dump(abi)

	cache.SetABIAtBlockNum("account.1", 2, abi)
	err = cache.Save("cursor.1", "not.used.1")
	require.NoError(t, err)

	err = cache.SaveState()
	require.NoError(t, err)

	loadedCache, err := NewABICache(store, cacheName)
	require.NoError(t, err)

	require.Equal(t, "cursor.1", loadedCache.GetCursor())
	require.Equal(t, 1, len(loadedCache.Abis))

	accountABIS := loadedCache.Abis["account.1"]
	require.Equal(t, 1, len(accountABIS))

	a := accountABIS[0]
	require.Equal(t, uint32(2), a.BlockNum)
	require.Equal(t, "eosio::abi/1.0", a.ABI.Version)

}

func TestDefaultCache_Large_Save_Load(t *testing.T) {
	cacheName := "test_cache.bin"
	ctx := context.Background()

	store, err := dstore.NewSimpleStore("file:///tmp")
	require.NoError(t, err)

	exist, err := store.FileExists(ctx, cacheName)
	require.NoError(t, err)
	if exist {
		err := store.DeleteObject(ctx, cacheName)
		require.NoError(t, err)
	}

	require.NoError(t, err)
	cache, err := NewABICache(store, cacheName)
	require.NoError(t, err)

	abiData, err := hex.DecodeString("0e656f73696f3a3a6162692f312e300110657468657265756d5f6164647265737306737472696e6702076164647265737300030269640675696e74363410657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e636505617373657403616464000210657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e63650561737365740100000000000052320361646400010000c00a637553320369363401026964010675696e7436340761646472657373000000")
	require.NoError(t, err)

	var abi *eos.ABI
	err = eos.UnmarshalBinary(abiData, &abi)
	require.NoError(t, err)

	for i := 1; i < 10000; i++ {
		cache.SetABIAtBlockNum("account.1", uint32(i), abi)
	}

	err = cache.Save("cursor.1", "not.used.1")
	require.NoError(t, err)

	err = cache.SaveState()
	require.NoError(t, err)

	_, err = NewABICache(store, cacheName)
	require.NoError(t, err)

	//require.Equal(t, "cursor.1", loadedCache.GetCursor())
	//require.Equal(t, 1, len(loadedCache.Abis))
	//
	//accountABIS := loadedCache.Abis["account.1"]
	//require.Equal(t, 1, len(accountABIS))
	//
	//a := accountABIS[0]
	//require.Equal(t, uint32(2), a.BlockNum)
	//require.Equal(t, "eosio::abi/1.0", a.ABI.Version)

}

func Test_Upload(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectBaseURL  string
		expectFileName string
		expectError    bool
	}{
		{
			name:           "golden path",
			url:            "gs://dfuseio-global-abicache-us/eos-dev1-v3.json.zst",
			expectBaseURL:  "gs://dfuseio-global-abicache-us",
			expectFileName: "eos-dev1-v3.json.zst",
			expectError:    false,
		},
		{
			name:           "does not add a compression extension, it expect it to already be configured",
			url:            "gs://dfuseio-global-abicache-us/eos-dev1-v3.json",
			expectBaseURL:  "gs://dfuseio-global-abicache-us",
			expectFileName: "eos-dev1-v3.json",
			expectError:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseURL, filename, err := getStoreInfo(test.url)
			if test.expectError {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectBaseURL, baseURL)
				assert.Equal(t, test.expectFileName, filename)
			}
		})
	}

}
