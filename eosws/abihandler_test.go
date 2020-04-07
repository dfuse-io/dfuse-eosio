// Copyright 2020 dfuse Platform Inc.
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

package eosws

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func TestABIChangeHandler_ProcessBlock(t *testing.T) {
	abiString1 := `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`
	abiString2 := `{"version":"eosio::abi/2.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`
	abiString3 := `{"version":"eosio::abi/3.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`

	abiGetter := NewTestABIGetter()
	abiGetter.SetABIForAccount(abiString1, eos.AccountName("eosio"))
	handler, err := NewABIChangeHandler(abiGetter, 2, eos.AccountName("eosio"), bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
		return nil
	}), context.Background())

	require.NoError(t, err)
	require.Equal(t, "eosio::abi/1.0", handler.CurrentABI().Version)

	//blkWithABI1 := newBlockWithAbi(t, abiString1)
	blkWithABI2 := newBlockWithAbi(t, abiString2)
	blkWithABI3 := newBlockWithAbi(t, abiString3)

	err = handler.ProcessBlock(blkWithABI2, &forkable.ForkableObject{Step: forkable.StepNew})
	require.NoError(t, err)
	require.Equal(t, "eosio::abi/2.0", handler.CurrentABI().Version)

	err = handler.ProcessBlock(blkWithABI3, &forkable.ForkableObject{Step: forkable.StepIrreversible})
	require.NoError(t, err)
	require.Equal(t, "eosio::abi/2.0", handler.CurrentABI().Version)

	err = handler.ProcessBlock(blkWithABI2, &forkable.ForkableObject{Step: forkable.StepUndo})
	require.NoError(t, err)
	require.Equal(t, "eosio::abi/1.0", handler.CurrentABI().Version)

	err = handler.ProcessBlock(blkWithABI2, &forkable.ForkableObject{Step: forkable.StepRedo})
	require.NoError(t, err)
	require.Equal(t, "eosio::abi/2.0", handler.CurrentABI().Version)

	err = handler.ProcessBlock(blkWithABI3, &forkable.ForkableObject{Step: forkable.StepNew})
	require.NoError(t, err)
	require.Equal(t, "eosio::abi/3.0", handler.CurrentABI().Version)

}

func TestABIChangeHandler_ProcesswithError(t *testing.T) {
	abiString := `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`

	abiGetter := NewTestABIGetter()
	abiGetter.SetABIForAccount(abiString, eos.AccountName("eosio"))
	handler, err := NewABIChangeHandler(abiGetter, 2, eos.AccountName("eosio"), bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
		return nil
	}), context.Background())

	require.NoError(t, err)
	require.Equal(t, "eosio::abi/1.0", handler.CurrentABI().Version)

	blkWithBadABI := testBlock(t, "00000002a", "00000001a", "eosio", 1, `{
		"id":"trx.1",
		"action_traces":[{
            "receiver": "eosio",
			"action": {
				"account": "eosio",
				"name": "setabi",
				"json_data": "{\"account\":\"eosio\",\"abi\":\"bad.data.here\"}"
			}
		}]
	}`)

	err = handler.ProcessBlock(blkWithBadABI, &forkable.ForkableObject{Step: forkable.StepNew})
	require.Error(t, err)
}

func newBlockWithAbi(t *testing.T, abiString string) *bstream.Block {
	t.Helper()

	abi, err := eos.NewABI(strings.NewReader(abiString))
	data, err := eos.MarshalBinary(abi)
	require.NoError(t, err)

	return testBlock(t, "00000002a", "00000001a", "eosio", 1, fmt.Sprintf(`{
		"id":"trx.1",
		"action_traces":[{
            "receiver": "eosio",
			"action": {
				"account": "eosio",
				"name": "setabi",
				"json_data": "{\"account\":\"eosio\",\"abi\":\"%s\"}"
			}
		}]
	}`, hex.EncodeToString(data)))
}
