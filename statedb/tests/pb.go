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

package tests

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testBlock(t *testing.T, blkID string, previousBlkID string, trxTraceJSONs ...string) *pbcodec.Block {
	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbcodec.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace), "actual string:\n"+trxTraceJSON)

		trxTraces[i] = trxTrace
	}

	pbblock := &pbcodec.Block{
		Id:                          blkID,
		Number:                      eos.BlockNum(blkID),
		UnfilteredTransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbcodec.BlockHeader{
		Previous:  previousBlkID,
		Producer:  "tester",
		Timestamp: blockTimestamp,
	}

	if os.Getenv("DEBUG") != "" {
		marshaler := &jsonpb.Marshaler{}
		out, err := marshaler.MarshalToString(pbblock)
		require.NoError(t, err)

		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
		normalizedOut := map[string]interface{}{}
		require.NoError(t, json.Unmarshal([]byte(out), &normalizedOut))

		zlog.Debug("created test block", zap.Any("block", normalizedOut))
	}

	return pbblock
}

func bstreamBlocks(t *testing.T, pbBlocks ...*pbcodec.Block) []*bstream.Block {
	blocks := make([]*bstream.Block, len(pbBlocks))
	for i, pbBlock := range pbBlocks {
		block, err := codec.BlockFromProto(pbBlock)
		require.NoError(t, err)

		blocks[i] = block
	}

	return blocks
}

func toTimestamp(t time.Time) *tspb.Timestamp {
	el, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return el
}

func dbOp(t *testing.T, abi *eos.ABI, op string, path string, payer string, data string) *pbcodec.DBOp {
	paths := strings.Split(path, "/")

	// Split those with → instead, will probably improve readability
	payers := strings.Split(payer, "/")
	datas := strings.Split(data, "/")

	dbOp := &pbcodec.DBOp{
		Operation:  pbcodec.DBOp_Operation(pbcodec.DBOp_Operation_value["OPERATION_"+strings.ToUpper(op)]),
		Code:       paths[0],
		TableName:  paths[1],
		Scope:      paths[2],
		PrimaryKey: paths[3],
	}

	if payers[0] != "" {
		dbOp.OldPayer = payers[0]
	}

	if payers[1] != "" {
		dbOp.NewPayer = payers[1]
	}

	dataToBinary := func(jsonContent string) []byte {
		data, err := abi.EncodeTable(eos.TableName(dbOp.TableName), []byte(jsonContent))
		require.NoError(t, err)

		return data
	}

	if datas[0] != "" {
		dbOp.OldData = dataToBinary(datas[0])
	}

	if datas[1] != "" {
		dbOp.NewData = dataToBinary(datas[1])
	}

	return dbOp
}

func tableOp(t *testing.T, op string, path string, payer string) *pbcodec.TableOp {
	paths := strings.Split(path, "/")

	return &pbcodec.TableOp{
		Operation: pbcodec.TableOp_Operation(pbcodec.TableOp_Operation_value["OPERATION_"+strings.ToUpper(op)]),
		Code:      paths[0],
		TableName: paths[1],
		Scope:     paths[2],
		Payer:     payer,
	}
}

func trxTrace(t *testing.T, elements ...proto.Message) string {
	trace := &pbcodec.TransactionTrace{}
	for _, element := range elements {
		switch v := element.(type) {
		case *pbcodec.ActionTrace:
			trace.ActionTraces = append(trace.ActionTraces, v)
		case *pbcodec.DBOp:
			trace.DbOps = append(trace.DbOps, v)
		case *pbcodec.TableOp:
			trace.TableOps = append(trace.TableOps, v)
		}
	}

	out, err := jsonpb.MarshalIndentToString(trace, "")
	require.NoError(t, err)

	return out
}

func actionSetABI(t *testing.T, account string, abi *eos.ABI) *pbcodec.ActionTrace {
	packedABI, err := eos.MarshalBinary(abi)
	require.NoError(t, err)

	return &pbcodec.ActionTrace{
		Receiver: "eosio",
		Receipt: &pbcodec.ActionReceipt{
			Receiver: "eosio",
		},
		Action: &pbcodec.Action{
			Account:  "eosio",
			Name:     "setabi",
			JsonData: str(`{"account":"%s","abi":"%s"}`, account, hex.EncodeToString(packedABI)),
		},
	}
}
