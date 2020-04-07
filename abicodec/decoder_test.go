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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dfuse-io/dstore"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDecoder_DecodeAction(t *testing.T) {

	var abi *eos.ABI
	err := json.Unmarshal([]byte(ABI_TRANSFER), &abi)
	require.NoError(t, err)

	store, err := dstore.NewSimpleStore("file:///tmp/cache/")
	require.NoError(t, err)
	cache, err := NewABICache(store, "test_cache.bin")
	require.NoError(t, err)
	cache.SetABIAtBlockNum("eosio.token", 100, abi)

	transferHex := "7015345262aaba4a90558c8663aaba4a853300000000000004454f53000000006d7b2274797065223a22627579222c226d61726b6574223a22454f53222c227175616e74697479223a22312e33313839222c227072696365223a22302e3130343334393137222c22636f6465223a22656f7364747374746f6b656e222c2273796d626f6c223a22454f534454227d"
	data, err := hex.DecodeString(transferHex)
	require.NoError(t, err)

	decoder := NewDecoder(cache)
	out, abiBlockNum, err := decoder.decodeAction("eosio.token", "transfer", data, 101)
	require.NoError(t, err)

	require.Equal(t, uint32(100), abiBlockNum)
	require.Equal(t, "dexeosmmaker", gjson.GetBytes(out, "from").Str)
	require.Equal(t, "dexeoswallet", gjson.GetBytes(out, "to").Str)
	require.Equal(t, "1.3189 EOS", gjson.GetBytes(out, "quantity").Str)

}

func TestDecoder_DecodeTable(t *testing.T) {

	var abi *eos.ABI
	err := json.Unmarshal([]byte(ABI_TRANSFER), &abi)
	require.NoError(t, err)

	store, err := dstore.NewSimpleStore("file:///tmp/cache/")
	require.NoError(t, err)
	cache, err := NewABICache(store, "test_cache.bin")
	require.NoError(t, err)

	cache.SetABIAtBlockNum("eosio.token", 100, abi)

	data, err := hex.DecodeString("2ef204000000000004454f5300000000")
	require.NoError(t, err)

	decoder := NewDecoder(cache)
	out, abiBlockNum, err := decoder.decodeTable("eosio.token", "accounts", data, 100)
	fmt.Println(string(out))
	require.NoError(t, err)

	require.Equal(t, uint32(100), abiBlockNum)
	require.Equal(t, "32.4142 EOS", gjson.GetBytes(out, "balance").Str)

}
