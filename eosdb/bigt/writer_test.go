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

package bigt

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/bigtable/bttest"
	"github.com/andreyvit/diff"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/jsonpb"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

var ctx = context.Background()

func newWriter(t *testing.T) (*EOSDatabase, func()) {
	srv, err := bttest.NewServer("localhost:0")
	require.NoError(t, err)
	conn, err := dgrpc.NewInternalClient(srv.Addr)
	require.NoError(t, err)

	db, err := NewDriver("test", "dev", "dev", true, time.Second, 10, option.WithGRPCConn(conn))
	require.NoError(t, err)

	// db.SetWriterChainID(hexChainID)

	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		srv.Close()
	}

	return db, cleanup
}

func TestBigtableWriter(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	ctx := context.Background()
	blockID := "00000002a"
	previousRef := bstream.BlockRefFromID("00000001a")
	block := testBlock(t, "00000002a")
	block.Header.Previous = previousRef.ID()

	blk, err := codec.BlockFromProto(block)
	require.NoError(t, err)

	require.NoError(t, db.PutBlock(ctx, block))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, block))
	require.NoError(t, db.Flush(ctx))

	// Block written
	resp, err := db.GetBlock(ctx, blockID)
	require.NoError(t, err)
	assert.Equal(t, blockID, resp.Block.Id)
	assert.True(t, resp.Irreversible)

	// Timeline written
	respID, _, err := db.BlockIDBefore(ctx, blk.Time(), true) // direct timestamp
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = db.BlockIDAfter(ctx, time.Time{}, true) // first block since epoch
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = db.BlockIDBefore(ctx, blk.Time().Add(-time.Second), true) // nothing before
	assert.Error(t, err)

	respID, _, err = db.BlockIDAfter(ctx, blk.Time().Add(time.Second), true) // nothing after
	assert.Error(t, err)
}

func TestStoreBlock(t *testing.T) {
	db, cleanup := newWriter(t)
	defer cleanup()

	zlibCompressedPackedTrxString, err := ioutil.ReadFile("testdata/zlib-compressed-packed-trx.hex.txt")
	require.NoError(t, err)

	zlibCompressedPackedTrx := mustHexDecode(string(zlibCompressedPackedTrxString))

	blockWithZlibCompressedTrx := testBlock(t, "00000002a")
	blockWithZlibCompressedTrx.Transactions = append(blockWithZlibCompressedTrx.Transactions,
		&pbcodec.TransactionReceipt{PackedTransaction: &pbcodec.PackedTransaction{
			Compression:       1,
			PackedTransaction: zlibCompressedPackedTrx,
		}},
	)

	tests := []struct {
		name                        string
		blocks                      []*pbcodec.Block
		expectedMutationsGoldenFile string
	}{
		{
			name:                        "empty block",
			blocks:                      []*pbcodec.Block{testBlock(t, "00000002a")},
			expectedMutationsGoldenFile: "empty-block.golden.json",
		},
		{
			name: "deferred trx failed",
			blocks: []*pbcodec.Block{
				testBlock(t, "00000002a",
					`{"id":"a1","ram_ops":[{"namespace":"NAMESPACE_DEFERRED_TRX","action":"ACTION_REMOVE"}]}`,
					`{"id":"a2","ram_ops":[{"action_index":2}],"failed_dtrx_trace":{"id":"a1","ram_ops":[{"namespace":"NAMESPACE_DEFERRED_TRX","action":"ACTION_REMOVE"}]}}`,
					`{"id":"a3"}`,
				),
			},
			expectedMutationsGoldenFile: "failed-dtrx-ram-replacement.golden.json",
		},
		{
			name: "deferred trx creation (pushed by user)",
			blocks: []*pbcodec.Block{
				testBlock(t, "00000002a",
					`{"id":"a1","dtrx_ops":[{"transaction_id":"a2","operation":"OPERATION_PUSH_CREATE","transaction":{}}]}`,
				),
			},
			expectedMutationsGoldenFile: "deferred_trx_creation_via_push.golden.json",
		},
		{
			name: "zlib compressed packed transaction",
			blocks: []*pbcodec.Block{
				blockWithZlibCompressedTrx,
			},
			expectedMutationsGoldenFile: "zlib_compressed_packed_transaction.golden.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
			}

			checkMutations(t, db, test.expectedMutationsGoldenFile)
			require.NoError(t, db.FlushAllMutations(context.Background()))
			for _, blk := range test.blocks {
				blk, err := db.GetBlock(ctx, blk.Id)
				require.NoError(t, err)
				_ = blk
			}

		})
	}
}

func checkMutations(t *testing.T, db *EOSDatabase, goldenFile string) {
	actual := marshalMutations(t, db)
	goldenFile = filepath.Join("testdata", goldenFile)
	if os.Getenv("GOLDEN_UPDATE") != "" {
		ioutil.WriteFile(goldenFile, actual, 0644)
	}

	expected, err := ioutil.ReadFile(goldenFile)
	if os.IsNotExist(err) {
		assert.Fail(t, "failed", "golden file %s does not exist, generate it with 'GOLDEN_UPDATE=true go test ./'", goldenFile)
		expected = []byte("{}")
	} else {
		require.NoError(t, err)
	}

	expectedString := string(expected)
	actualString := string(actual)

	assert.JSONEq(t, expectedString, actualString, diff.LineDiff(expectedString, actualString))
}

func marshalMutations(t *testing.T, db *EOSDatabase) []byte {
	t.Helper()

	type goldenMutations struct {
		Accounts     []goldenTestSetEntry
		Blocks       []goldenTestSetEntry
		Timeline     []goldenTestSetEntry
		Transactions []goldenTestSetEntry
	}

	mutations := &goldenMutations{
		Accounts:     generateMutationsTestSetEntries(db.Accounts.PendingSets()),
		Blocks:       generateMutationsTestSetEntries(db.Blocks.PendingSets()),
		Timeline:     generateMutationsTestSetEntries(db.Timeline.PendingSets()),
		Transactions: generateMutationsTestSetEntries(db.Transactions.PendingSets()),
	}

	content, err := json.MarshalIndent(mutations, "", "  ")
	require.NoError(t, err)

	return content
}

func generateMutationsTestSetEntries(pendingMutations []*basebigt.SetEntry) (out []goldenTestSetEntry) {
	for _, mutation := range pendingMutations {
		familyColumn := mutation.Family + ":" + mutation.Column

		mutationValue := maybeProtoBytesToMutationValue(familyColumn, mutation.Value)
		if mutationValue == nil {
			mutationValue = string(mutation.Value)
		}

		out = append(out, goldenTestSetEntry{
			Key: mutation.Key, FamilyColumn: familyColumn, Value: mutationValue,
		})
	}

	sort.Sort(goldenTestSetEntries(out))

	return out
}

func maybeProtoBytesToMutationValue(familyColumn string, bytes []byte) interface{} {
	message := familyColumnToProtoMessage(familyColumn)
	if message == nil {
		return nil
	}

	err := proto.Unmarshal(bytes, message)
	if err != nil {
		return nil
	}

	marshaller := &jsonpb.Marshaler{}
	normalizedContent, err := marshaller.MarshalToString(message)
	if err != nil {
		return nil
	}

	data := map[string]interface{}{}
	err = json.Unmarshal([]byte(normalizedContent), &data)
	if err != nil {
		return nil
	}

	return data
}

func familyColumnToProtoMessage(familyColumn string) proto.Message {
	switch familyColumn {
	case "block:proto":
		return &pbcodec.Block{}
	case "trace:proto":
		return &pbcodec.TransactionTrace{}
	case "trx:proto":
		return &pbcodec.SignedTransaction{}
	case "meta:blockheader":
		return &pbcodec.BlockHeader{}
	case "trxs:trxRefsProto", "trxs:traceRefsProto":
		return &pbcodec.TransactionRefs{}
	case "dtrx:created-by", "dtrx:canceled-by":
		return &pbcodec.ExtDTrxOp{}
	}

	return nil
}

type goldenTestSetEntry struct {
	Key, FamilyColumn string
	Value             interface{}
}

type goldenTestSetEntries []goldenTestSetEntry

func (a goldenTestSetEntries) Len() int      { return len(a) }
func (a goldenTestSetEntries) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a goldenTestSetEntries) Less(i, j int) bool {
	left := a[i]
	right := a[j]

	return (left.Key + left.FamilyColumn) < (right.Key + right.FamilyColumn)
}

func testBlock(t *testing.T, id string, trxTraceJSONs ...string) *pbcodec.Block {
	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbcodec.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbcodec.Block{
		Id:                id,
		Number:            eos.BlockNum(id),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbcodec.BlockHeader{
		Previous:  fmt.Sprintf("%08d%s", pbblock.Number-1, id[8:]),
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
