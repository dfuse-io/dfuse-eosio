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
	"testing"

	"github.com/dfuse-io/dstore"
	"github.com/stretchr/testify/require"
)

func TestSearchWatcherPreset_Handle(t *testing.T) {
	t.Skip()
	testCases := []struct {
		name              string
		json              string
		blockID           string
		trxID             string
		account           string
		expectedBlockNum  uint32
		expectedItemCount int
	}{
		{
			name:              "sunny path",
			json:              `{"json": {"account": "account.1", "abi": "0e656f73696f3a3a6162692f312e300110657468657265756d5f6164647265737306737472696e6702076164647265737300030269640675696e74363410657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e636505617373657403616464000210657468657265756d5f6164647265737310657468657265756d5f616464726573730762616c616e63650561737365740100000000000052320361646400010000c00a637553320369363401026964010675696e7436340761646472657373000000"}}`,
			blockID:           "0000044b582e4ffdb68c84e1a4379327736c3f85ac2317d9ad92156b1e9a4617",
			trxID:             "tx.id.1",
			account:           "account.1",
			expectedBlockNum:  uint32(1099),
			expectedItemCount: 1,
		},
		{
			name:              "hex decode error",
			json:              `{"json": {"account": "account.1", "abi": "bad.hex"}}`,
			blockID:           "0000044b582e4ffdb68c84e1a4379327736c3f85ac2317d9ad92156b1e9a4617",
			trxID:             "tx.id.1",
			account:           "account.1",
			expectedBlockNum:  uint32(1099),
			expectedItemCount: 0,
		},
		{
			name:              "invalid abi data",
			json:              `{"json": {"account": "account.1", "abi": "6261642E616269"}}`, //bad.abi in hex
			blockID:           "0000044b582e4ffdb68c84e1a4379327736c3f85ac2317d9ad92156b1e9a4617",
			trxID:             "tx.id.1",
			account:           "account.1",
			expectedBlockNum:  uint32(1099),
			expectedItemCount: 0,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			store, err := dstore.NewSimpleStore("file:///tmp/cache")
			require.NoError(t, err)
			cache, err := NewABICache(store, "test_cache.bin")
			require.NoError(t, err)

			/*			watcher := NewSearchWatcherHandler(cache, nil, false, "")

						err = watcher.Handle(c.blockID, c.trxID, c.json, false)
						require.NoError(t, err)
			*/
			items := cache.Abis[c.account]
			require.Len(t, items, c.expectedItemCount)

			if c.expectedItemCount == 1 {
				require.Equal(t, uint32(c.expectedBlockNum), items[0].BlockNum)
			}
		})
	}
}
