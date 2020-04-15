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

package resolvers

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/search/eos/v1"
	"github.com/dfuse-io/dgraphql"
	"github.com/dfuse-io/dtracing"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSearchMatchArchive(trxID string) *pbsearch.SearchMatch {
	data, err := proto.Marshal(&pbsearcheos.Match{})
	if err != nil {
		panic(err)
	}
	return &pbsearch.SearchMatch{
		TrxIdPrefix: trxID,
		BlockNum:    0,
		Index:       0,
		Cursor:      "",
		ChainSpecific: &any.Any{
			TypeUrl: "dfuse://eos.search.match",
			Value:   data,
		},
		Undo:        false,
		IrrBlockNum: 0,
	}
}

func newSearchMatchLive(trxID string, idx int) *pbsearch.SearchMatch {
	data, err := proto.Marshal(&pbsearcheos.Match{
		Block: &pbsearcheos.BlockTrxPayload{
			Trace: &pbeos.TransactionTrace{Index: uint64(idx)},
		},
	})
	if err != nil {
		panic(err)
	}

	return &pbsearch.SearchMatch{
		TrxIdPrefix: trxID,
		ChainSpecific: &any.Any{
			TypeUrl: "dfuse://eos.search.match",
			Value:   data,
		},
	}
}

func newDgraphqlResponse(trxID string, idx int) *SearchTransactionForwardResponse {
	return &SearchTransactionForwardResponse{
		SearchTransactionBackwardResponse: SearchTransactionBackwardResponse{
			trxIDPrefix: trxID,
			trxTrace: &pbeos.TransactionTrace{
				Index: uint64(idx),
			},
		},
	}
}

func TestSubscriptionSearchForward(t *testing.T) {
	ctx := dtracing.NewFixedTraceIDInContext(context.Background(), "00000000000000000000000000000000")

	tests := []struct {
		name        string
		fromRouter  []interface{}
		fromDB      map[string][]*pbeos.TransactionEvent
		expect      []*SearchTransactionForwardResponse
		expectError error
	}{
		{
			name: "simple",
			fromRouter: []interface{}{
				newSearchMatchArchive("trx123"),
				fmt.Errorf("failed"),
			},
			fromDB: map[string][]*pbeos.TransactionEvent{
				"trx123": {
					{Id: "trx12399999999999999999", Event: pbeos.NewTestExecEvent(5)},
				},
			},
			expect: []*SearchTransactionForwardResponse{
				newDgraphqlResponse("trx123", 5),
				{
					err: dgraphql.Errorf(ctx, "failed"),
				},
			},

			expectError: nil,
		},
		{
			name: "hammered",
			fromRouter: []interface{}{
				newSearchMatchArchive("trx000"),
				newSearchMatchArchive("trx001"),
				newSearchMatchArchive("trx002"),
				newSearchMatchArchive("trx022"),
				newSearchMatchLive("trx003", 8),
				newSearchMatchLive("trx004", 9),
				newSearchMatchLive("trx005", 10),
			},
			fromDB: map[string][]*pbeos.TransactionEvent{
				"trx000": {
					{Id: "trx000boo", Event: pbeos.NewTestExecEvent(5)},
				},
				"trx001": {
					{Id: "trx001boo", Event: pbeos.NewTestExecEvent(6)},
				},
				"trx002": {
					{Id: "trx002boo", Event: pbeos.NewTestExecEvent(7)},
				},
				"trx022": {
					{Id: "trx022boo", Event: pbeos.NewTestExecEvent(11)},
				},
			},
			expect: []*SearchTransactionForwardResponse{
				newDgraphqlResponse("trx000", 5),
				newDgraphqlResponse("trx001", 6),
				newDgraphqlResponse("trx002", 7),
				newDgraphqlResponse("trx022", 11),
				newDgraphqlResponse("trx003", 8),
				newDgraphqlResponse("trx004", 9),
				newDgraphqlResponse("trx005", 10),
			},

			expectError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			root := &Root{
				searchClient: pbsearch.NewTestRouterClient(test.fromRouter),
				trxsReader:   eosdb.NewTestTransactionsReader(test.fromDB),
			}

			res, err := root.streamSearchTracesBoth(true, ctx, StreamSearchArgs{})
			if test.expectError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				var expect []*SearchTransactionForwardResponse
				for el := range res {
					if el.err != nil {

					}
					expect = append(expect, el)
				}

				assert.Equal(t, test.expect, expect)
			}
		})
	}
}

//
//import (
//	"context"
//	"fmt"
//	"testing"
//
//	"github.com/sergi/go-diff/diffmatchpatch"
//
//	"github.com/graph-gophers/graphql-go"
//	"github.com/tidwall/gjson"
//
//	"github.com/stretchr/testify/require"
//
//	test_schema "github.com/dfuse-io/dgraphql/schema/test"
//
//	"github.com/stretchr/testify/assert"
//)
//
//type TestRoot struct {
//	Root
//}
//
//func (r *TestRoot) QueryTestSearch() (*SearchTransactionForwardResponse, error) {
//	resp := &SearchTransactionForwardResponse{
//		SearchTransactionBackwardResponse: SearchTransactionBackwardResponse{
//			cursor:                "cursor.1",
//			irreversibleBlockNum:  99,
//			matchingActionIndexes: []uint32{0},
//			dbops:                 []byte(`[{"op":"INS","action_idx":0,"npayer":"laulaulau123","path":"eosio/laulaulau123/userres/laulaulau123","new":"3044d0266a13b589000000000000000004454f5300000000000000000000000004454f53000000000000000000000000"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/rammarket/cpd4ykuhc5d.4","old":"00407a10f35a00000452414d434f5245fe2d1cac0c0000000052414d00000000000000000000e03f6a5495f00200000004454f5300000000000000000000e03f","new":"00407a10f35a00000452414d434f5245fe2d1cac0c0000000052414d00000000000000000000e03f6a5495f00200000004454f5300000000000000000000e03f"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/rexpool/","old":"00e03273470300000004454f5300000000b83f447b0c00000004454f53000000001bbb204b0000000004454f53000000009872b7c20f00000004454f5300000000c8c58b3ce1e800000452455800000000000000000000000004454f53000000002702000000000000","new":"00e03273470300000004454f5300000000bd3f447b0c00000004454f53000000001bbb204b0000000004454f53000000009d72b7c20f00000004454f5300000000c8c58b3ce1e800000452455800000000000000000000000004454f53000000002702000000000000"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/rammarket/cpd4ykuhc5d.4","old":"00407a10f35a00000452414d434f5245fe2d1cac0c0000000052414d00000000000000000000e03f6a5495f00200000004454f5300000000000000000000e03f","new":"00407a10f35a00000452414d434f5245151e1cac0c0000000052414d00000000000000000000e03f1b5895f00200000004454f5300000000000000000000e03f"},{"op":"UPD","action_idx":1,"opayer":"laulaulau123","npayer":"laulaulau123","path":"eosio/laulaulau123/userres/laulaulau123","old":"3044d0266a13b589000000000000000004454f5300000000000000000000000004454f53000000000000000000000000","new":"3044d0266a13b589000000000000000004454f5300000000000000000000000004454f5300000000e90f000000000000"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global/global","old":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","new":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b0000100000040006000000000010000000ebe1e35303000000db66869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global2/global2","old":"00009a86cd480087cd48ecc60b5659d7e24401","new":"00000187cd480087cd48ecc60b5659d7e24401"},{"op":"UPD","action_idx":1,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global3/global3","old":"809029437288050014705804f791cd43","new":"809029437288050014705804f791cd43"},{"op":"UPD","action_idx":2,"opayer":"junglefaucet","npayer":"junglefaucet","path":"eosio.token/junglefaucet/accounts/........ehbo5","old":"f5b41d9f1109000004454f5300000000","new":"44b11d9f1109000004454f5300000000"},{"op":"UPD","action_idx":2,"opayer":"eosio.ram","npayer":"eosio.ram","path":"eosio.token/eosio.ram/accounts/........ehbo5","old":"2a63869c0000000004454f5300000000","new":"db66869c0000000004454f5300000000"},{"op":"UPD","action_idx":5,"opayer":"junglefaucet","npayer":"junglefaucet","path":"eosio.token/junglefaucet/accounts/........ehbo5","old":"44b11d9f1109000004454f5300000000","new":"3fb11d9f1109000004454f5300000000"},{"op":"UPD","action_idx":5,"opayer":"eosio.ramfee","npayer":"eosio.ramfee","path":"eosio.token/eosio.ramfee/accounts/........ehbo5","old":"24abd0000000000004454f5300000000","new":"29abd0000000000004454f5300000000"},{"op":"UPD","action_idx":8,"opayer":"eosio.ramfee","npayer":"eosio.ramfee","path":"eosio.token/eosio.ramfee/accounts/........ehbo5","old":"29abd0000000000004454f5300000000","new":"24abd0000000000004454f5300000000"},{"op":"UPD","action_idx":8,"opayer":"eosio.rex","npayer":"eosio.rex","path":"eosio.token/eosio.rex/accounts/........ehbo5","old":"aa7e0d1f1b00000004454f5300000000","new":"af7e0d1f1b00000004454f5300000000"},{"op":"INS","action_idx":11,"npayer":"laulaulau123","path":"eosio/laulaulau123/delband/laulaulau123","new":"3044d0266a13b5893044d0266a13b589102700000000000004454f5300000000102700000000000004454f5300000000"},{"op":"UPD","action_idx":11,"opayer":"laulaulau123","npayer":"laulaulau123","path":"eosio/laulaulau123/userres/laulaulau123","old":"3044d0266a13b589000000000000000004454f5300000000000000000000000004454f5300000000e90f000000000000","new":"3044d0266a13b589102700000000000004454f5300000000102700000000000004454f5300000000e90f000000000000"},{"op":"INS","action_idx":11,"npayer":"laulaulau123","path":"eosio/eosio/voters/laulaulau123","new":"3044d0266a13b589000000000000000000204e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},{"op":"UPD","action_idx":11,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global/global","old":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b0000100000040006000000000010000000ebe1e35303000000db66869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","new":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b0000100000040006000000000010000000ebe1e35303000000db66869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48"},{"op":"UPD","action_idx":11,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global2/global2","old":"00000187cd480087cd48ecc60b5659d7e24401","new":"00000187cd480087cd48ecc60b5659d7e24401"},{"op":"UPD","action_idx":11,"opayer":"eosio","npayer":"eosio","path":"eosio/eosio/global3/global3","old":"809029437288050014705804f791cd43","new":"809029437288050014705804f791cd43"},{"op":"UPD","action_idx":12,"opayer":"junglefaucet","npayer":"junglefaucet","path":"eosio.token/junglefaucet/accounts/........ehbo5","old":"3fb11d9f1109000004454f5300000000","new":"1f631d9f1109000004454f5300000000"},{"op":"UPD","action_idx":12,"opayer":"eosio.stake","npayer":"eosio.stake","path":"eosio.token/eosio.stake/accounts/........ehbo5","old":"c59eb750920c000004454f5300000000","new":"e5ecb750920c000004454f5300000000"}]`),
//			blockHeader:           []byte(`{"timestamp":"2019-05-09T10:54:56.500","producer":"eosdacserval","confirmed":0,"previous":"01a7dc74fe39f798f33e3ab8b1382c8fa2b79cea7a828bb33aee8387b9cbe85f","transaction_mroot":"ce1ef6dc2f0bb511a8b20b5cde4b9091c6c975efefa805511dfdf9e1cb9792ed","action_mroot":"1b639c974b0f4fba0ef36a9644e41b2ef24bc126b42aef8140838c2ad9b45e7a","schedule_version":178,"header_extensions":[],"producer_signature":"SIG_K1_KhmxyeAgYEUriXYNGaKoK8d8nHMmEpGN5xNg2xZTzFXNZb3eyTuAJkohkhBAuCBD3GBUvWSRTpVeCBQXXoVojCyFF4GsL6"}`),
//			trxTraces:             []byte(`{"id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","receipt":{"status":"executed","cpu_usage_us":1360,"net_usage_words":42},"elapsed":35605,"net_usage":336,"scheduled":false,"action_traces":[{"receipt":{"receiver":"eosio","act_digest":"e519e7da08910c3127fa8347a3cd128afca1fb2c6ec871f832eb97cf7fc57246","global_sequence":398248105,"recv_sequence":31715389,"auth_sequence":[["junglefaucet",322466]],"code_sequence":7,"abi_sequence":7},"act":{"account":"eosio","name":"newaccount","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"creator":"junglefaucet","name":"laulaulau123","owner":{"threshold":1,"keys":[{"key":"EOS7RdKLHvWkS6y46UxLWEn6jzzkUwiCn8rpHyNutG6qpTe3dF3ga","weight":1}],"accounts":[],"waits":[]},"active":{"threshold":1,"keys":[{"key":"EOS7j3SCLpSpq1pPXajb71L4nzj1KUPnMmMJ3hzPhcAu8ViDRuUHh","weight":1}],"accounts":[],"waits":[]}},"hex_data":"9015d266a9c8a67e3044d0266a13b589010000000100034e17de2b351f0c853e2ed02a68e37f858c2896da7c5fb96b17b1700703c3d8bf010000000100000001000375a354dc4cfbb457e078e01b7f2fc8b2a58d4f4f2e3373c9ae7f069b5467b50301000000"},"context_free":false,"elapsed":27233,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[{"account":"laulaulau123","delta":2996}],"except":null,"inline_traces":[]},{"receipt":{"receiver":"eosio","act_digest":"dd2e40946e93b51725f983992f58f064cf05d8266c77b3d634536225e4985bd9","global_sequence":398248106,"recv_sequence":31715390,"auth_sequence":[["junglefaucet",322467]],"code_sequence":7,"abi_sequence":7},"act":{"account":"eosio","name":"buyrambytes","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"payer":"junglefaucet","receiver":"laulaulau123","bytes":4096},"hex_data":"9015d266a9c8a67e3044d0266a13b58900100000"},"context_free":false,"elapsed":6651,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[{"receipt":{"receiver":"eosio.token","act_digest":"0ea67fd9c19d29f0907423ad20169f7ba7a0affc0ee27bd5fcf65dc7f97fa3ca","global_sequence":398248107,"recv_sequence":72508439,"auth_sequence":[["eosio.ram",153681],["junglefaucet",322468]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"},{"actor":"eosio.ram","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ram","quantity":"0.0945 EOS","memo":"buy ram"},"hex_data":"9015d266a9c8a67e000090e602ea3055b10300000000000004454f5300000000076275792072616d"},"context_free":false,"elapsed":251,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[{"receipt":{"receiver":"junglefaucet","act_digest":"0ea67fd9c19d29f0907423ad20169f7ba7a0affc0ee27bd5fcf65dc7f97fa3ca","global_sequence":398248108,"recv_sequence":95312,"auth_sequence":[["eosio.ram",153682],["junglefaucet",322469]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"},{"actor":"eosio.ram","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ram","quantity":"0.0945 EOS","memo":"buy ram"},"hex_data":"9015d266a9c8a67e000090e602ea3055b10300000000000004454f5300000000076275792072616d"},"context_free":false,"elapsed":18,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]},{"receipt":{"receiver":"eosio.ram","act_digest":"0ea67fd9c19d29f0907423ad20169f7ba7a0affc0ee27bd5fcf65dc7f97fa3ca","global_sequence":398248109,"recv_sequence":215189,"auth_sequence":[["eosio.ram",153683],["junglefaucet",322470]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"},{"actor":"eosio.ram","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ram","quantity":"0.0945 EOS","memo":"buy ram"},"hex_data":"9015d266a9c8a67e000090e602ea3055b10300000000000004454f5300000000076275792072616d"},"context_free":false,"elapsed":14,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]}]},{"receipt":{"receiver":"eosio.token","act_digest":"5ab1c77b4f08b230b0b256790d35f2453afd2f51d8d9e5a7214d9ac07d5a9986","global_sequence":398248110,"recv_sequence":72508440,"auth_sequence":[["junglefaucet",322471]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ramfee","quantity":"0.0005 EOS","memo":"ram fee"},"hex_data":"9015d266a9c8a67ea0d492e602ea3055050000000000000004454f53000000000772616d20666565"},"context_free":false,"elapsed":97,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[{"receipt":{"receiver":"junglefaucet","act_digest":"5ab1c77b4f08b230b0b256790d35f2453afd2f51d8d9e5a7214d9ac07d5a9986","global_sequence":398248111,"recv_sequence":95313,"auth_sequence":[["junglefaucet",322472]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ramfee","quantity":"0.0005 EOS","memo":"ram fee"},"hex_data":"9015d266a9c8a67ea0d492e602ea3055050000000000000004454f53000000000772616d20666565"},"context_free":false,"elapsed":34,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]},{"receipt":{"receiver":"eosio.ramfee","act_digest":"5ab1c77b4f08b230b0b256790d35f2453afd2f51d8d9e5a7214d9ac07d5a9986","global_sequence":398248112,"recv_sequence":258268,"auth_sequence":[["junglefaucet",322473]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.ramfee","quantity":"0.0005 EOS","memo":"ram fee"},"hex_data":"9015d266a9c8a67ea0d492e602ea3055050000000000000004454f53000000000772616d20666565"},"context_free":false,"elapsed":12,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]}]},{"receipt":{"receiver":"eosio.token","act_digest":"ebe2e402b101a8c73d118ef7fa86fabdb0900ff61631b8869d92f9c313d92a4e","global_sequence":398248113,"recv_sequence":72508441,"auth_sequence":[["eosio.ramfee",129243]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"eosio.ramfee","permission":"active"}],"data":{"from":"eosio.ramfee","to":"eosio.rex","quantity":"0.0005 EOS","memo":"transfer from eosio.ramfee to eosio.rex"},"hex_data":"a0d492e602ea30550000e8ea02ea3055050000000000000004454f5300000000277472616e736665722066726f6d20656f73696f2e72616d66656520746f20656f73696f2e726578"},"context_free":false,"elapsed":113,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[{"receipt":{"receiver":"eosio.ramfee","act_digest":"ebe2e402b101a8c73d118ef7fa86fabdb0900ff61631b8869d92f9c313d92a4e","global_sequence":398248114,"recv_sequence":258269,"auth_sequence":[["eosio.ramfee",129244]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"eosio.ramfee","permission":"active"}],"data":{"from":"eosio.ramfee","to":"eosio.rex","quantity":"0.0005 EOS","memo":"transfer from eosio.ramfee to eosio.rex"},"hex_data":"a0d492e602ea30550000e8ea02ea3055050000000000000004454f5300000000277472616e736665722066726f6d20656f73696f2e72616d66656520746f20656f73696f2e726578"},"context_free":false,"elapsed":5,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]},{"receipt":{"receiver":"eosio.rex","act_digest":"ebe2e402b101a8c73d118ef7fa86fabdb0900ff61631b8869d92f9c313d92a4e","global_sequence":398248115,"recv_sequence":46240,"auth_sequence":[["eosio.ramfee",129245]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"eosio.ramfee","permission":"active"}],"data":{"from":"eosio.ramfee","to":"eosio.rex","quantity":"0.0005 EOS","memo":"transfer from eosio.ramfee to eosio.rex"},"hex_data":"a0d492e602ea30550000e8ea02ea3055050000000000000004454f5300000000277472616e736665722066726f6d20656f73696f2e72616d66656520746f20656f73696f2e726578"},"context_free":false,"elapsed":11,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]}]}]},{"receipt":{"receiver":"eosio","act_digest":"277b27049902dfa6804c311c618ccdd49c9db418556b86a600c69f00928d8e21","global_sequence":398248116,"recv_sequence":31715391,"auth_sequence":[["junglefaucet",322474]],"code_sequence":7,"abi_sequence":7},"act":{"account":"eosio","name":"delegatebw","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","receiver":"laulaulau123","stake_net_quantity":"1.0000 EOS","stake_cpu_quantity":"1.0000 EOS","transfer":1},"hex_data":"9015d266a9c8a67e3044d0266a13b589102700000000000004454f5300000000102700000000000004454f530000000001"},"context_free":false,"elapsed":583,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[{"account":"laulaulau123","delta":450}],"except":null,"inline_traces":[{"receipt":{"receiver":"eosio.token","act_digest":"8d3cea2c340b4db23b96b79cec7bf9c3f2bb4ff13be31116014f585b0ea73e84","global_sequence":398248117,"recv_sequence":72508442,"auth_sequence":[["junglefaucet",322475]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.stake","quantity":"2.0000 EOS","memo":"stake bandwidth"},"hex_data":"9015d266a9c8a67e0014341903ea3055204e00000000000004454f53000000000f7374616b652062616e647769647468"},"context_free":false,"elapsed":120,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[{"receipt":{"receiver":"junglefaucet","act_digest":"8d3cea2c340b4db23b96b79cec7bf9c3f2bb4ff13be31116014f585b0ea73e84","global_sequence":398248118,"recv_sequence":95314,"auth_sequence":[["junglefaucet",322476]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.stake","quantity":"2.0000 EOS","memo":"stake bandwidth"},"hex_data":"9015d266a9c8a67e0014341903ea3055204e00000000000004454f53000000000f7374616b652062616e647769647468"},"context_free":false,"elapsed":6,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]},{"receipt":{"receiver":"eosio.stake","act_digest":"8d3cea2c340b4db23b96b79cec7bf9c3f2bb4ff13be31116014f585b0ea73e84","global_sequence":398248119,"recv_sequence":278094,"auth_sequence":[["junglefaucet",322477]],"code_sequence":3,"abi_sequence":2},"act":{"account":"eosio.token","name":"transfer","authorization":[{"actor":"junglefaucet","permission":"active"}],"data":{"from":"junglefaucet","to":"eosio.stake","quantity":"2.0000 EOS","memo":"stake bandwidth"},"hex_data":"9015d266a9c8a67e0014341903ea3055204e00000000000004454f53000000000f7374616b652062616e647769647468"},"context_free":false,"elapsed":72,"console":"","trx_id":"fb611c6e6be3282a5a1d4b7f0f62e2c078d5e0a55bdb944bb400da0f118f1c6c","block_num":27778165,"block_time":"2019-05-09T10:54:56.500","producer_block_id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","account_ram_deltas":[],"except":null,"inline_traces":[]}]}]}],"except":null}`),
//			tableops:              []byte(`[{"op":"INS","action_idx":0,"payer":"laulaulau123","path":"eosio/laulaulau123/userres"},{"op":"INS","action_idx":11,"payer":"laulaulau123","path":"eosio/laulaulau123/delband"}]`),
//			ramops:                []byte(`[{"op":"newaccount","action_idx":0,"payer":"laulaulau123","delta":2724,"usage":2724},{"op":"create_table","action_idx":0,"payer":"laulaulau123","delta":112,"usage":2836},{"op":"primary_index_add","action_idx":0,"payer":"laulaulau123","delta":160,"usage":2996},{"op":"create_table","action_idx":11,"payer":"laulaulau123","delta":112,"usage":3108},{"op":"primary_index_add","action_idx":11,"payer":"laulaulau123","delta":160,"usage":3268},{"op":"primary_index_add","action_idx":11,"payer":"laulaulau123","delta":178,"usage":3446}]`),
//			dtrxops:               []byte(``),
//			creationTree:          []byte(`[[0,-1,0],[1,-1,1],[2,1,2],[3,2,3],[4,2,4],[5,1,5],[6,5,6],[7,5,7],[8,1,8],[9,8,9],[10,8,10],[11,-1,11],[12,11,12],[13,12,13],[14,12,14]]`),
//		},
//		undo: true,
//	}
//
//	return resp, nil
//}
//
//func TestResolverV1Payload(t *testing.T) {
//
//	s, err := graphql.ParseSchema(
//		test_schema.String(),
//		&TestRoot{Root: Root{}},
//		graphql.PrefixRootFunctions(),
//		graphql.UseStringDescriptions(), graphql.UseFieldResolvers(),
//		graphql.MaxDepth(24), // this is good for at least 6 levels of `inlineTraces`, fetching its data, etc..
//	)
//
//	assert.NoError(t, err)
//
//	q := `
//{
//    TestSearch {
//      cursor
//      undo
//      isIrreversible
//      irreversibleBlockNum
//      trace {
//        id
//        block {
//          ...block
//        }
//        status
//        receipt {
//          status
//          cpuUsageMicroSeconds
//          netUsageWords
//        }
//        elapsed
//        netUsage
//        scheduled
//        executedActions {
//          ...actionTrace
//        }
//        matchingActionIndexes{
//          ...actionTrace
//        }
//        topLevelActions{
//          ...actionTrace
//        }
//        exceptJSON
//      }
//    }
//  }
//
//fragment block on BlockHeader {
//  id
//  num
//  timestamp
//  producer
//  confirmed
//  previous
//  transactionMRoot
//  actionMRoot
//  scheduleVersion
//  newProducers {
//    version
//    producers {
//      producerName
//      blockSigningKey
//    }
//  }
//}
//
//fragment transaction on SignedTransaction {
//  expiration
//  refBlockNum
//  refBlockPrefix
//  maxNetUsageWords
//  maxCPUUsageMS
//  delaySec
//  contextFreeActions {
//    account
//    name
//    authorization {
//      actor
//      permission
//    }
//    json
//    data
//    hexData
//  }
//  actions {
//    ...action
//  }
//}
//
//fragment action on Action {
//  account
//  name
//  authorization {
//    actor
//    permission
//  }
//  json
//  data
//  hexData
//}
//
//fragment actionReceipt on ActionReceipt {
//  receiver
//  digest
//  globalSequence
//  codeSequence
//  abiSequence
//}
//
//fragment authorization on PermissionLevel {
//  actor
//  permission
//}
//
//fragment ramOps on RAMOp {
//  operation
//  payer
//  delta
//  usage
//}
//
//fragment dtrxOps on DTrxOp {
//  operation
//  sender
//  senderID
//  payer
//  publishedAt
//  delayUntil
//  expirationAt
//  trxID
//  transaction {
//    ...transaction
//  }
//}
//
//fragment tableOps on TableOp {
//
//    operation
//    table {
//      code
//      table
//      scope
//    }
//
//}
//
//fragment dbOps on DBOp {
//      operation
//    oldPayer
//    newPayer
//    key {
//      code
//      table
//      scope
//      key
//    }
//    oldData
//    newData
//    oldJSON
//    newJSON
//}
//
//fragment baseActionTrace on ActionTrace {
//    seq
//  receiver
//  account
//  name
//  data
//  json
//  hexData
//  receipt {
//    ...actionReceipt
//  }
//  authorization {
//    ...authorization
//  }
//  ramOps {
//    ...ramOps
//  }
//  dtrxOps {
//    ...dtrxOps
//  }
//  tableOps {
//    ...tableOps
//  }
//  dbOps {
//    ...dbOps
//  }
//  console
//  contextFree
//  elapsed
//  exceptJSON
//  isNotify
//  isMatchingQuery
//}
//
//fragment actionTrace on ActionTrace {
//  ...baseActionTrace
//  createdActions{
//    ...baseActionTrace
//  }
//  creatorAction{
//    ...baseActionTrace
//  }
//  closestUnnotifiedAncestorAction{
//    ...baseActionTrace
//  }
//}
//	`
//
//	expected := `{"data":{"searchTransactionsForward":{"results":[{"cursor":"Exu5wgkZmwO_II01DdCfYfe7JpE_AVJuUw7vIBkV0Yrz9yOUj5T3CA==","undo":false,"isIrreversible":true,"irreversibleBlockNum":27778165,"trace":{"id":"8dd157aab9a882c168f29db5b3d46043e84f5a220cce0bba266f8a626a962ab8","block":{"id":"01a7dc756cc6f1397b3efafe3433dd815651212f56cf7e6ab11cebd1b65044f7","num":27778165,"timestamp":"2019-05-09T10:54:56.5Z","producer":"eosdacserval","confirmed":0,"previous":"01a7dc74fe39f798f33e3ab8b1382c8fa2b79cea7a828bb33aee8387b9cbe85f","transactionMRoot":"ce1ef6dc2f0bb511a8b20b5cde4b9091c6c975efefa805511dfdf9e1cb9792ed","actionMRoot":"1b639c974b0f4fba0ef36a9644e41b2ef24bc126b42aef8140838c2ad9b45e7a","scheduleVersion":178,"newProducers":null},"status":"EXECUTED","receipt":{"status":"EXECUTED","cpuUsageMicroSeconds":100,"netUsageWords":0},"elapsed":"407","netUsage":"0","scheduled":false,"executedActions":[{"seq":"398248104","receiver":"eosio","account":"eosio","name":"onblock","data":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"json":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"hexData":"0087cd4810cdbe0a23933055000001a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f950000000000000000000000000000000000000000000000000000000000000000ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054b20000000000","receipt":{"receiver":"eosio","digest":"da84489f49b49d492a7b9d9f453443db039960b7ce3460d9289e024b4bd3b1af","globalSequence":"398248104","codeSequence":"7","abiSequence":"7"},"authorization":[{"actor":"eosio","permission":"active"}],"ramOps":null,"dtrxOps":null,"tableOps":null,"dbOps":[{"operation":"UPD","oldPayer":"eosdacserval","newPayer":"eosdacserval","key":{"code":"eosio","table":"producers","scope":"eosio","key":"eosdacserval"},"oldData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f751500004036faa2638805003a03","newData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f761500004036faa2638805003a03","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global","scope":"eosio","key":"global"},"oldData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000008a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","newData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global2","scope":"eosio","key":"global2"},"oldData":"00009a86cd48ff86cd48ecc60b5659d7e24401","newData":"00009a86cd480087cd48ecc60b5659d7e24401","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global3","scope":"eosio","key":"global3"},"oldData":"809029437288050014705804f791cd43","newData":"809029437288050014705804f791cd43","oldJSON":null,"newJSON":null}],"console":"","contextFree":false,"elapsed":"281","exceptJSON":null,"isNotify":false,"isMatchingQuery":true,"createdActions":[],"creatorAction":null,"closestUnnotifiedAncestorAction":null}],"matchingActionIndexes":[{"seq":"398248104","receiver":"eosio","account":"eosio","name":"onblock","data":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"json":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"hexData":"0087cd4810cdbe0a23933055000001a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f950000000000000000000000000000000000000000000000000000000000000000ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054b20000000000","receipt":{"receiver":"eosio","digest":"da84489f49b49d492a7b9d9f453443db039960b7ce3460d9289e024b4bd3b1af","globalSequence":"398248104","codeSequence":"7","abiSequence":"7"},"authorization":[{"actor":"eosio","permission":"active"}],"ramOps":null,"dtrxOps":null,"tableOps":null,"dbOps":[{"operation":"UPD","oldPayer":"eosdacserval","newPayer":"eosdacserval","key":{"code":"eosio","table":"producers","scope":"eosio","key":"eosdacserval"},"oldData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f751500004036faa2638805003a03","newData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f761500004036faa2638805003a03","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global","scope":"eosio","key":"global"},"oldData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000008a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","newData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global2","scope":"eosio","key":"global2"},"oldData":"00009a86cd48ff86cd48ecc60b5659d7e24401","newData":"00009a86cd480087cd48ecc60b5659d7e24401","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global3","scope":"eosio","key":"global3"},"oldData":"809029437288050014705804f791cd43","newData":"809029437288050014705804f791cd43","oldJSON":null,"newJSON":null}],"console":"","contextFree":false,"elapsed":"281","exceptJSON":null,"isNotify":false,"isMatchingQuery":true,"createdActions":[],"creatorAction":null,"closestUnnotifiedAncestorAction":null}],"topLevelActions":[{"seq":"398248104","receiver":"eosio","account":"eosio","name":"onblock","data":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"json":{"h":{"timestamp":1221428992,"producer":"eosdacserval","confirmed":0,"previous":"01a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f95","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054","schedule_version":178,"new_producers":null}},"hexData":"0087cd4810cdbe0a23933055000001a7dc73386679e426a66c50a31702c10e37c1b4215e0a3cd598f23c39c35f950000000000000000000000000000000000000000000000000000000000000000ab4d5a6811722690612b7466d50bd6445afbeee970fb74922bf0a964f5320054b20000000000","receipt":{"receiver":"eosio","digest":"da84489f49b49d492a7b9d9f453443db039960b7ce3460d9289e024b4bd3b1af","globalSequence":"398248104","codeSequence":"7","abiSequence":"7"},"authorization":[{"actor":"eosio","permission":"active"}],"ramOps":null,"dtrxOps":null,"tableOps":null,"dbOps":[{"operation":"UPD","oldPayer":"eosdacserval","newPayer":"eosdacserval","key":{"code":"eosio","table":"producers","scope":"eosio","key":"eosdacserval"},"oldData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f751500004036faa2638805003a03","newData":"10cdbe0a23933055048a38058cb18c430002287039ea488ae1398c60a5e66350dfbdadf59faeffce2be7fccfafd5c30bedcf011168747470733a2f2f656f736461632e696f761500004036faa2638805003a03","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global","scope":"eosio","key":"global"},"oldData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000008a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","newData":"0000080000000000e8030000ffff07000c000000f40100001400000064000000400d0300c4090000f049020064000000100e00005802000080533b000010000004000600000000001000000002d2e353030000002a63869c00000000e486cd4880eae0bd6d880500e09f04f20000000005ba09850500000009a26f00e20f54aab501000040f90d35587b05001500b637373a4605d5434193cb48","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global2","scope":"eosio","key":"global2"},"oldData":"00009a86cd48ff86cd48ecc60b5659d7e24401","newData":"00009a86cd480087cd48ecc60b5659d7e24401","oldJSON":null,"newJSON":null},{"operation":"UPD","oldPayer":"eosio","newPayer":"eosio","key":{"code":"eosio","table":"global3","scope":"eosio","key":"global3"},"oldData":"809029437288050014705804f791cd43","newData":"809029437288050014705804f791cd43","oldJSON":null,"newJSON":null}],"console":"","contextFree":false,"elapsed":"281","exceptJSON":null,"isNotify":false,"isMatchingQuery":true,"createdActions":[],"creatorAction":null,"closestUnnotifiedAncestorAction":null}],"exceptJSON":null}}]}}}`
//
//	resp := s.Exec(context.Background(), q, "", make(map[string]interface{}))
//	fmt.Println(resp.Errors)
//	require.Len(t, resp.Errors, 0)
//
//	out := gjson.GetBytes(resp.Data, "TestSearch").Str
//
//	dmp := diffmatchpatch.New()
//
//	diffs := dmp.DiffMain(out, expected, false)
//
//	fmt.Println("diff", dmp.DiffPrettyText(diffs))
//}
