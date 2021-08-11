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

package trxdb_loader

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	_ "github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	_ "github.com/streamingfast/kvdb/store/badger"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type chainIDOption = string

func newLoader(t *testing.T, options ...interface{}) (*TrxDBLoader, trxdb.DB, func()) {

	db, err := trxdb.New("badger:///tmp?cache=shared&mode=memory&createTables=true", trxdb.WithLogger(zlog))
	require.NoError(t, err)

	l := NewTrxDBLoader("", nil, 1, db, 1, nil, 0, nil)

	var chainID string
	for _, option := range options {
		switch v := option.(type) {
		case chainIDOption:
			chainID = v
		}
	}

	if chainID != "" {
		hexChainID, err := hex.DecodeString(chainID)
		require.NoError(t, err)
		db.SetWriterChainID(hexChainID)
	}

	require.NoError(t, err)

	cleanup := func() {
	}

	return l, db, cleanup
}

func TestBigtableLoader(t *testing.T) {
	loader, trxdbDriver, cleanup := newLoader(t)
	defer cleanup()

	ctx := context.Background()
	blockID := "00000002aa"
	previousRef := bstream.NewBlockRefFromID("00000001aa")
	block := testBlock(t, "00000002aa")
	block.Header.Previous = previousRef.ID()

	blk, err := codec.BlockFromProto(block)
	require.NoError(t, err)

	fkable := forkable.New(loader, forkable.WithLogger(zlog), forkable.WithExclusiveLIB(previousRef))
	require.NoError(t, fkable.ProcessBlock(blk, nil))
	loader.UpdateIrreversibleData([]*bstream.PreprocessedBlock{{Block: blk}})
	require.NoError(t, loader.db.Flush(ctx))

	resp, err := trxdbDriver.GetBlock(ctx, blockID)
	require.NoError(t, err)
	assert.Equal(t, blockID, resp.Block.Id)
	assert.True(t, resp.Irreversible)
}

func TestBigtableLoader_Timeline(t *testing.T) {
	t.Skip() // not yet ready without sqlite
	loader, trxdbDriver, cleanup := newLoader(t)
	defer cleanup()

	ctx := context.Background()
	blockID := "00000002aa"
	previousRef := bstream.NewBlockRefFromID("00000001aa")
	block := testBlock(t, "00000002aa")
	block.Header.Previous = previousRef.ID()

	blk, err := codec.BlockFromProto(block)
	require.NoError(t, err)

	fkable := forkable.New(loader, forkable.WithLogger(zlog), forkable.WithExclusiveLIB(previousRef))
	require.NoError(t, fkable.ProcessBlock(blk, nil))
	loader.UpdateIrreversibleData([]*bstream.PreprocessedBlock{{Block: blk}})
	require.NoError(t, trxdbDriver.Flush(ctx))

	respID, _, err := trxdbDriver.BlockIDBefore(ctx, blk.Time(), true) // direct timestamp
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = trxdbDriver.BlockIDBefore(ctx, blk.Time(), true) // direct timestamp
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = trxdbDriver.BlockIDAfter(ctx, time.Time{}, true) // first block since epoch
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = trxdbDriver.BlockIDBefore(ctx, blk.Time().Add(-time.Second), true) // nothing before
	assert.Error(t, err)

	respID, _, err = trxdbDriver.BlockIDAfter(ctx, blk.Time().Add(time.Second), true) // nothing after
	assert.Error(t, err)
}

func testBlock(t *testing.T, id string, trxTraceJSONs ...string) *pbcodec.Block {
	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbcodec.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbcodec.Block{
		Id:                          id,
		Number:                      eos.BlockNum(id),
		UnfilteredTransactionTraces: trxTraces,
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
