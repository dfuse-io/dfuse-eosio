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

package sql

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-eosio/codec"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func newWriter(t *testing.T) (*DB, func()) {
	rawdb, err := New("sqlite3:///tmp/mama.db?cache=shared&mode=memory")
	require.NoError(t, err)

	db := rawdb.(*DB)

	//_, _ = db.db.Exec("DROP DATABASE unittests")
	//_, err = db.db.Exec("CREATE DATABASE unittests")
	//require.NoError(t, err)

	rawdb, err = New("sqlite3:///tmp/mama.db?cache=shared&mode=memory&createTables=true")
	require.NoError(t, err)

	db = rawdb.(*DB)
	// db.SetWriterChainID(hexChainID)

	cleanup := func() {
		db.Close()
		err := os.Remove("/tmp/mama.db")
		require.NoError(t, err)
	}

	return db, cleanup
}

func testBlock1() *pbeos.Block {
	blockTime, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	blockTimestamp, _ := ptypes.TimestampProto(blockTime)

	trx := &eos.Transaction{
		TransactionHeader: eos.TransactionHeader{
			Expiration:     eos.JSONTime{blockTime},
			RefBlockNum:    123,
			RefBlockPrefix: 234,
		},
		Actions: []*eos.Action{
			{
				Account:    "some",
				Name:       "name",
				ActionData: eos.NewActionDataFromHexData([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}),
			},
		},
	}
	signedTrx := eos.NewSignedTransaction(trx)
	signedTrx.Signatures = append(signedTrx.Signatures, ecc.MustNewSignature("SIG_K1_K7kTcvsznS2pSQ2unjW9nduqHieWnc5B6rFdbVif4RM1DCTVhQUpzwng3XTGewDhVZqNvqSAEwHgB8yBnfDYAHquRX4fBo"))
	packed, err := signedTrx.Pack(eos.CompressionNone)
	if err != nil {
		panic(err)
	}
	trxID, _ := hex.DecodeString("00112233")
	receipt := &eos.TransactionReceipt{
		TransactionReceiptHeader: eos.TransactionReceiptHeader{
			Status:               eos.TransactionStatusExecuted,
			CPUUsageMicroSeconds: 32,
			NetUsageWords:        eos.Varuint32(32),
		},
		Transaction: eos.TransactionWithID{
			ID:     eos.Checksum256([]byte(trxID)),
			Packed: packed,
		},
	}

	pbblock := &pbeos.Block{
		Id:                       "00000002a",
		Number:                   2,
		DposIrreversibleBlocknum: 1,
		Header: &pbeos.BlockHeader{
			Previous:  "00000001a",
			Producer:  "tester",
			Timestamp: blockTimestamp,
		},
		Transactions: []*pbeos.TransactionReceipt{
			codec.TransactionReceiptToDEOS(receipt),
		},
		ImplicitTransactionOps: []*pbeos.TrxOp{
			{
				Operation:     pbeos.TrxOp_OPERATION_CREATE,
				Name:          "onblock",
				TransactionId: "abc999",
				Transaction: &pbeos.SignedTransaction{
					Transaction: &pbeos.Transaction{},
				},
			},
		},
		TransactionTraces: []*pbeos.TransactionTrace{
			{
				Id: "00112233",
				DtrxOps: []*pbeos.DTrxOp{
					{
						Operation:     pbeos.DTrxOp_OPERATION_CREATE,
						TransactionId: "trx777",
						Transaction: &pbeos.SignedTransaction{
							Transaction: &pbeos.Transaction{},
						},
					},
					{
						Operation:     pbeos.DTrxOp_OPERATION_CANCEL,
						TransactionId: "trx888",
						Transaction: &pbeos.SignedTransaction{
							Transaction: &pbeos.Transaction{},
						},
					},
				},
				ActionTraces: []*pbeos.ActionTrace{
					{
						Receiver: "eosio",
						Action: &pbeos.Action{
							Account:  "eosio",
							Name:     "newaccount",
							JsonData: `{"creator": "frankenstein", "name": "createdacct"}`,
						},
					},
				},
			},
		},
	}

	if os.Getenv("DEBUG") != "" {
		marshaler := &jsonpb.Marshaler{}
		out, err := marshaler.MarshalToString(pbblock)
		if err != nil {
			panic(err)
		}

		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
		normalizedOut := map[string]interface{}{}
		err = json.Unmarshal([]byte(out), &normalizedOut)
		if err != nil {
			panic(err)
		}

		//zlog.Debug("created test block", zap.Any("block", normalizedOut))
	}

	return pbblock
}

func TestReadBlock(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	ctx := context.Background()
	in := testBlock1()

	require.NoError(t, db.PutBlock(ctx, in))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, in))
	require.NoError(t, db.FlushAllMutations(ctx))

	_, err := codec.BlockFromProto(in)
	require.NoError(t, err)

	// Block data
	blockID := "00000002a"
	resp, err := db.GetBlock(ctx, blockID)
	require.NoError(t, err)
	assert.Equal(t, in.Id, resp.Block.Id)
	assert.True(t, resp.Irreversible)
	assert.Equal(t, 1, len(resp.TransactionTraceRefs.Hashes))
	assert.Equal(t, 1, len(resp.TransactionRefs.Hashes))
	assert.Equal(t, 1, len(resp.ImplicitTransactionRefs.Hashes))
	assert.Equal(t, in.Number, resp.Block.Number)
	assert.Equal(t, in.Header.Producer, resp.Block.Header.Producer)
	assert.Equal(t, in.DposIrreversibleBlocknum, resp.Block.DposIrreversibleBlocknum)
	assert.Nil(t, resp.Block.Transactions)
	assert.Nil(t, resp.Block.TransactionTraces)
	assert.Nil(t, resp.Block.ImplicitTransactionOps)

	resp2, err := db.GetBlockByNum(ctx, 2)
	require.NoError(t, err)
	require.Len(t, resp2, 1)
	assert.Equal(t, in.Id, resp2[0].Block.Id)
	assert.True(t, resp2[0].Irreversible)

	// Timeline written
	// respID, _, err := db.BlockIDBefore(ctx, in.MustTime(), true) // direct timestamp
	// assert.NoError(t, err)
	// assert.Equal(t, blockID, respID)

	// respID, _, err = db.BlockIDAfter(ctx, time.Time{}, true) // first block since epoch
	// assert.NoError(t, err)
	// assert.Equal(t, blockID, respID)

	// respID, _, err = db.BlockIDBefore(ctx, in.MustTime().Add(-time.Second), true) // nothing before
	// assert.Error(t, err)

	// respID, _, err = db.BlockIDAfter(ctx, in.MustTime().Add(time.Second), true) // nothing after
	// assert.Error(t, err)
}

func TestReadTransactions(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	in := testBlock1()

	require.NoError(t, db.PutBlock(ctx, in))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, in))

	// Block data
	evs, err := db.GetTransactionEvents(context.Background(), "00112233")
	require.NoError(t, err)
	assert.Len(t, evs, 2)

	ev1 := evs[0]
	assert.Equal(t, "00112233", ev1.Id)
	assert.Equal(t, "00000002a", ev1.BlockId)
	assert.True(t, ev1.Irreversible)
	add, found := ev1.Event.(*pbeos.TransactionEvent_Addition)
	assert.True(t, found)
	assert.Equal(t, 32, int(add.Addition.Receipt.NetUsageWords))
	assert.Equal(t, 32, int(add.Addition.Receipt.CpuUsageMicroSeconds))
	assert.Equal(t, []string{"SIG_K1_K7kTcvsznS2pSQ2unjW9nduqHieWnc5B6rFdbVif4RM1DCTVhQUpzwng3XTGewDhVZqNvqSAEwHgB8yBnfDYAHquRX4fBo"}, add.Addition.Transaction.Signatures)
	assert.Len(t, add.Addition.Transaction.Transaction.Actions, 1)
	assert.Equal(t, "name", add.Addition.Transaction.Transaction.Actions[0].Name)
	assert.Equal(t, []string{"EOS7T3GcBYpYf2D63HGDG7qB9TiD56XT4m1hAQfkHWuV9LhMoQ1ZY"}, add.Addition.PublicKeys.PublicKeys)

	ev2 := evs[1]
	assert.Equal(t, "00112233", ev2.Id)
	assert.Equal(t, "00000002a", ev2.BlockId)
	assert.True(t, ev2.Irreversible)

	exec, found := ev2.Event.(*pbeos.TransactionEvent_Execution)
	assert.True(t, found)
	assert.Equal(t, "00000001a", exec.Execution.BlockHeader.Previous)
	assert.Equal(t, "tester", exec.Execution.BlockHeader.Producer)
	assert.Len(t, exec.Execution.Trace.DtrxOps, 2)
	assert.Equal(t, "trx888", exec.Execution.Trace.DtrxOps[1].TransactionId)
	assert.Equal(t, "00112233", exec.Execution.Trace.Id)
}

func TestReadTimelineEqual(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	noon := time.Date(2020, time.February, 02, 12, 0, 0, 0, time.UTC)

	blkA := &pbeos.Block{
		Id:     "00000008a",
		Number: 8,
		Header: &pbeos.BlockHeader{
			Timestamp: toTimestamp(noon),
		},
	}
	require.NoError(t, db.PutBlock(ctx, blkA))

	res, err := db.BlockIDAt(context.Background(), noon)
	require.NoError(t, err)
	assert.Equal(t, "00000008a", res)

	blkB := &pbeos.Block{
		Id:     "00000008b",
		Number: 8,
		Header: &pbeos.BlockHeader{
			Timestamp: toTimestamp(noon),
		},
	}
	require.NoError(t, db.PutBlock(ctx, blkB))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blkB))

	res, err = db.BlockIDAt(context.Background(), noon)
	require.NoError(t, err)
	assert.Equal(t, "00000008b", res)
}

func TestReadTimelineAfter(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	beforeNoon := time.Date(2020, time.February, 02, 11, 0, 0, 0, time.UTC)
	noon := time.Date(2020, time.February, 02, 12, 0, 0, 0, time.UTC)
	afterNoon := time.Date(2020, time.February, 02, 13, 0, 0, 0, time.UTC)

	blkBeforeNoon := &pbeos.Block{
		Id:     "00000007a",
		Number: 7,
		Header: &pbeos.BlockHeader{
			Timestamp: toTimestamp(beforeNoon),
		},
	}
	require.NoError(t, db.PutBlock(ctx, blkBeforeNoon))

	blkNoon := &pbeos.Block{
		Id:     "00000008a",
		Number: 8,
		Header: &pbeos.BlockHeader{
			Timestamp: toTimestamp(noon),
		},
	}
	require.NoError(t, db.PutBlock(ctx, blkNoon))

	blkAfterNoon := &pbeos.Block{
		Id:     "00000009a",
		Number: 9,
		Header: &pbeos.BlockHeader{
			Timestamp: toTimestamp(afterNoon),
		},
	}
	require.NoError(t, db.PutBlock(ctx, blkAfterNoon))

	res, tm, err := db.BlockIDAfter(context.Background(), noon, false)
	require.NoError(t, err)
	assert.Equal(t, "00000009a", res) // not included
	assert.Equal(t, afterNoon.UTC(), tm.UTC())

	res, tm, err = db.BlockIDAfter(context.Background(), noon, true)
	require.NoError(t, err)
	assert.Equal(t, "00000008a", res)
	assert.Equal(t, noon.UTC(), tm.UTC())

	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blkAfterNoon))

	// Test BEFORE
	res, tm, err = db.BlockIDBefore(context.Background(), noon, false)
	require.NoError(t, err)
	assert.Equal(t, "00000007a", res)
	assert.Equal(t, beforeNoon.UTC(), tm.UTC())

	res, tm, err = db.BlockIDBefore(context.Background(), noon, true)
	require.NoError(t, err)
	assert.Equal(t, "00000008a", res)
	assert.Equal(t, noon.UTC(), tm.UTC())

	// FIXME: now 9a takes precedence, because its irreversible
	// but it shouldn't, it should use `8` because that blockNum is the first to appear
	// and *that* block num doesn't have anything irreversible
	// That is very low risk, and we LIMIT 4, so the moment we have 4 blocks that are
	// *reversible*, we'll use the first one
	res, tm, err = db.BlockIDAfter(context.Background(), noon, true)
	require.NoError(t, err)
	assert.Equal(t, "00000009a", res)
	assert.Equal(t, afterNoon.UTC(), tm.UTC())

}

func toTimestamp(t time.Time) *tspb.Timestamp {
	el, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return el
}
