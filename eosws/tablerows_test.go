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
	"encoding/json"
	"strings"
	"testing"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	fluxdb "github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/jsonpb"
	eos "github.com/eoscanada/eos-go"
	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_onGetTableRows(t *testing.T) {

	archiveStore := dstore.NewMockStore(nil)

	abiGetter := NewTestABIGetter()

	cases := []struct {
		name              string
		lib               uint32
		archiveFiles      []archiveFiles
		fluxDBResponse    string
		abiForAccountName *ABIAccountName
		msg               string
		expectedOutput    []string
	}{
		{
			name: "sunny path",
			lib:  1,
			archiveFiles: []archiveFiles{{name: "0000000000", content: pbeosBlockToFile(t, pbeosBlockFromString(t, `{
    "header": {
        "previous": "00000001a",
        "timestamp": "2019-09-09T00:00:00Z"
    },
    "id": "00000002a",
    "number": 2,
    "transaction_traces": [
        {
            "db_ops": [
                {
                    "action_index": 0,
                    "new_data": "096e65772e76616c7565",
                    "old_data": "096f6c642e76616c7565",
                    "new_payer": "new_payer",
                    "old_payer": "old.payer",
                    "code": "eosio",
                    "scope": "eosio",
                    "table_name": "table_name_1",
                    "primary_key": "key_name_1"
                }
            ],
            "id": "trx.1"
        }
    ]
}
`))}},
			fluxDBResponse:    `{"last_irreversible_block_id":"00000001a","last_irreversible_block_num":1,"up_to_block_id":"00000001a","up_to_block_num":1,"rows":{"foo":"bar"}}`,
			abiForAccountName: &ABIAccountName{accountName: eos.AccountName("account.1"), abiString: `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`},
			msg:               `{"type":"get_table_rows","req_id":"abc","listen":false,"fetch":true,"data":{"code":"account.1","scope":"scope.1","table":"table.name.1","json":true}}`,
			expectedOutput:    []string{`{"type":"table_snapshot","req_id":"abc","data":{"rows":{"foo":"bar"}}}`},
		},
	}

	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			for _, f := range c.archiveFiles {
				archiveStore.SetFile(f.name, f.content)
			}
			subscriptionHub := newTestSubscriptionHub(t, c.lib, archiveStore)
			fluxClient := fluxdb.NewTestFluxClient()

			handler := NewWebsocketHandler(
				abiGetter,
				nil,
				nil,
				subscriptionHub,
				fluxClient,
				nil,
				nil,
				nil,
				NewTestIrreversibleFinder("00000001a", nil),
				0,
			)

			conn, closer := newTestConnection(t, handler)
			defer closer()

			fluxClient.SetGetTableResponse(c.fluxDBResponse, nil)

			abiGetter.SetABIForAccount(c.abiForAccountName.abiString, c.abiForAccountName.accountName)

			err := conn.WriteMessage(1, []byte(c.msg))
			require.NoError(t, err)
			validateOutput(t, "", c.expectedOutput, conn)
		})
	}
}

func TestTableDeltaHandler_ProcessBlock(t *testing.T) {
	scope := eos.Name("eosio")

	msg := &wsmsg.GetTableRows{
		CommonIn: wsmsg.CommonIn{
			Fetch:      false,
			Listen:     true,
			StartBlock: 3,
		},
		Data: wsmsg.GetTableRowsData{
			Code:      "eosio",
			Scope:     &scope,
			TableName: "table_name_1",
			JSON:      true,
		},
	}

	abiString := `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`
	abi, err := eos.NewABI(strings.NewReader(abiString))
	require.NoError(t, err)

	cases := []struct {
		name                 string
		block                *bstream.Block
		step                 forkable.StepType
		expectedMessageCount int
	}{
		{
			name: "no delta matching",
			block: testBlock(t, "00000002a", "00000001a", "eosio", 1, `{"id":"trx.1","db_ops":[{
				"old_payer": "old.payer",
				"new_payer": "new_payer",
                "code": "eosio",
                "scope": "eosio",
                "table_name": "table_name_1",
                "primary_key": "key_name",
				"old_data": "096f6c642e76616c7565",
				"new_data": "096e65772e76616c7565"
			}]}`),
			step:                 forkable.StepNew,
			expectedMessageCount: 1,
		},
		{
			name: "no delta none matching",
			block: testBlock(t, "00000003a", "00000002a", "eosio", 2, `{"id":"trx.1","db_ops":[{
				"old_payer": "old.payer",
				"new_payer": "new_payer",
                "code": "eosio",
                "scope": "eosio",
                "table_name": "table_name_1",
                "primary_key": "key_name",
				"old_data": "096f6c642e76616c7565",
				"new_data": "096e65772e76616c7565"
			}]}`),
			expectedMessageCount: 0,
			step:                 forkable.StepHandoff,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fobj := &forkable.ForkableObject{
				Step: c.step,
			}

			require.NoError(t, err)
			emitter := NewTestEmitter(context.Background(), nil)
			handler := NewTableDeltaHandler(msg, emitter, context.Background(), zlog, func() *eos.ABI {
				return abi
			})
			err = handler.ProcessBlock(c.block, fobj)
			require.NoError(t, err)
			assert.Equal(t, c.expectedMessageCount, len(emitter.messages))
		})
	}
}

func Test_dbopsFromBlock(t *testing.T) {

	abiString := `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`
	abi, err := eos.NewABI(strings.NewReader(abiString))
	assert.NoError(t, err)

	goodBlock := testBlock(t, "00000002a0", "00000001a0", "eosio", 1, `{"id": "abcd","db_ops":[{
		"operation": "OPERATION_UPDATE",
		"action_index": 0,
		"old_payer": "old.payer",
		"new_payer": "new_payer",
        "code": "eosio",
        "scope": "eosio",
        "table_name": "table_name_1",
        "primary_key": "key_name",
		"old_data": "096f6c642e76616c7565",
		"new_data": "096e65772e76616c7565"
	}]}`)

	badOldDataHEX := testBlock(t, "00000002a0", "00000001a0", "eosio", 1, `{"id": "abcd","db_ops":[{
		"operation": "OPERATION_UPDATE",
		"action_index": 0,
		"old_payer": "old.payer",
		"new_payer": "new_payer",
        "code": "eosio",
        "scope": "eosio",
        "table_name": "table_name_1",
        "primary_key": "key_name",
		"old_data": "096f6c642e76616c7565",
		"new_data": "096e65772e76616c7565"
	}]}`)

	scope := eos.Name("eosio")

	msg := &wsmsg.GetTableRows{
		Data: wsmsg.GetTableRowsData{
			Code:      "eosio",
			Scope:     &scope,
			TableName: "table_name_1",
			JSON:      true,
		},
	}
	msgWrongCode := &wsmsg.GetTableRows{
		Data: wsmsg.GetTableRowsData{
			Code:      "wrong",
			Scope:     &scope,
			TableName: "table_name_1",
			JSON:      true,
		},
	}

	cases := []struct {
		name                    string
		block                   *bstream.Block
		msg                     *wsmsg.GetTableRows
		step                    forkable.StepType
		expectedError           error
		expectedOldData         interface{}
		expectedNewData         interface{}
		expectedStep            string
		expectedTableDeltaCount int
	}{
		{
			name:                    "sunny path",
			step:                    forkable.StepNew,
			block:                   goodBlock,
			msg:                     msg,
			expectedStep:            "new",
			expectedTableDeltaCount: 1,
			expectedError:           nil,
			expectedOldData:         json.RawMessage([]byte(`{"struct_1_field_1":"old.value"}`)),
			expectedNewData:         json.RawMessage([]byte(`{"struct_1_field_1":"new.value"}`)),
		},
		{
			name:                    "sunny path with undo",
			block:                   goodBlock,
			msg:                     msg,
			step:                    forkable.StepUndo,
			expectedStep:            "undo",
			expectedTableDeltaCount: 1,
			expectedError:           nil,
			expectedOldData:         json.RawMessage([]byte(`{"struct_1_field_1":"new.value"}`)),
			expectedNewData:         json.RawMessage([]byte(`{"struct_1_field_1":"old.value"}`)),
		},
		{
			name: "Bad old data hex",
			step: forkable.StepNew,

			block:                   badOldDataHEX,
			msg:                     msg,
			expectedStep:            "new",
			expectedTableDeltaCount: 1,
			expectedOldData:         nil,
			expectedNewData:         json.RawMessage([]byte(`{"struct_1_field_1":"new.value"}`)),
		},
		{
			name:                    "wrong code",
			block:                   goodBlock,
			msg:                     msgWrongCode,
			expectedOldData:         nil,
			expectedTableDeltaCount: 0,
			expectedNewData:         nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tableDeltas := tableDeltasFromBlock(c.block, c.msg, abi, c.step, zlog)

			require.Equal(t, c.expectedTableDeltaCount, len(tableDeltas))

			if c.expectedTableDeltaCount > 0 {
				require.NotNil(t, tableDeltas[0].Data)
				require.Equal(t, c.expectedStep, tableDeltas[0].Data.Step)

				if c.expectedOldData != nil {
					require.NotNil(t, tableDeltas[0].Data.DBOp)
					require.NotNil(t, tableDeltas[0].Data.DBOp.Old)
					require.Equal(t, c.expectedOldData, tableDeltas[0].Data.DBOp.Old.JSON)
				}

				if c.expectedNewData != nil {
					require.NotNil(t, tableDeltas[0].Data.DBOp)
					require.NotNil(t, tableDeltas[0].Data.DBOp.New)
					require.Equal(t, c.expectedNewData, tableDeltas[0].Data.DBOp.New.JSON)
				}
			}
		})
	}

}

func TestToV1DBOp(t *testing.T) {

	abiString := `{"version":"eosio::abi/1.0","structs":[{"name":"struct_name_1","fields":[{"name":"struct_1_field_1","type":"string"}]}],"tables":[{"name":"table_name_1","index_type":"i64","key_names":["key_name_1"],"key_types":["string"],"type":"struct_name_1"}]}`
	abi, err := eos.NewABI(strings.NewReader(abiString))
	assert.NoError(t, err)

	cases := []struct {
		name                string
		rowHexData          string
		asJSON              bool
		expectedJSON        interface{}
		expectedHex         string
		expectedErrorPrefix string
	}{
		{
			name:         "sunny path",
			rowHexData:   "096e65772e76616c7565",
			asJSON:       true,
			expectedJSON: json.RawMessage(`{"struct_1_field_1":"new.value"}`),
		},
		{
			name:         "sunny hex path",
			rowHexData:   "096e65772e76616c7565",
			asJSON:       false,
			expectedJSON: nil,
			expectedHex:  "096e65772e76616c7565",
		},
		{
			name:                "abi error",
			rowHexData:          "1027000000000000",
			asJSON:              true,
			expectedJSON:        nil,
			expectedErrorPrefix: "Couldn't json decode ROW:",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rowData, err := hex.DecodeString(c.rowHexData)
			require.NoError(t, err)
			v1DBRow := newDBRow(rowData, eos.TableName("table_name_1"), abi, "payer.1", c.asJSON, zlog)
			require.Equal(t, c.expectedJSON, v1DBRow.JSON)
			require.Equal(t, c.expectedHex, v1DBRow.Hex)
			require.True(t, strings.HasPrefix(v1DBRow.Error, c.expectedErrorPrefix))
		})
	}
}

func protoJSONUnmarshal(t *testing.T, data []byte, into proto.Message) {
	require.NoError(t, jsonpb.UnmarshalString(string(data), into))
}

func pbeosBlockFromString(t *testing.T, in string) (out *pbeos.Block) {
	out = &pbeos.Block{}
	require.NoError(t, jsonpb.UnmarshalString(in, out))
	return
}
