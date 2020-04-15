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

package eos

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
	"github.com/dfuse-io/dfuse-eosio/codecs/deos"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	_ "github.com/dfuse-io/dfuse-eosio/eosdb/sql"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type chainIDOption = string

func newLoader(t *testing.T, options ...interface{}) (*BigtableLoader, eosdb.Driver, func()) {

	db, err := eosdb.New("sqlite3:///tmp/mama.db?cache=shared&mode=memory&createTables=true")
	require.NoError(t, err)

	l := NewBigtableLoader("", nil, 1, db, 1)

	var chainID string
	for _, option := range options {
		switch v := option.(type) {
		case chainIDOption:
			chainID = v
		}
	}

	l.InitLIB("0000000000000000000000000000000000000000000000000000000000000000")

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
	loader, eosdbDriver, cleanup := newLoader(t)
	defer cleanup()

	ctx := context.Background()
	blockID := "00000002a"
	previousRef := bstream.BlockRefFromID("00000001a")
	loader.forkDB.InitLIB(previousRef)
	block := testBlock(t, "00000002a")
	block.Header.Previous = previousRef.ID()

	blk, err := deos.BlockFromProto(block)
	require.NoError(t, err)

	fkable := forkable.New(loader, forkable.WithExclusiveLIB(previousRef))
	require.NoError(t, fkable.ProcessBlock(blk, nil))
	loader.UpdateIrreversibleData([]*bstream.PreprocessedBlock{{Block: blk}})
	require.NoError(t, loader.db.Flush(ctx))

	resp, err := eosdbDriver.GetBlock(ctx, blockID)
	require.NoError(t, err)
	assert.Equal(t, blockID, resp.Block.Id)
	assert.True(t, resp.Irreversible)
}

func TestBigtableLoader_Timeline(t *testing.T) {
	loader, eosdbDriver, cleanup := newLoader(t)
	defer cleanup()

	ctx := context.Background()
	blockID := "00000002a"
	previousRef := bstream.BlockRefFromID("00000001a")
	loader.forkDB.InitLIB(previousRef)
	block := testBlock(t, "00000002a")
	block.Header.Previous = previousRef.ID()

	blk, err := deos.BlockFromProto(block)
	require.NoError(t, err)

	fkable := forkable.New(loader, forkable.WithExclusiveLIB(previousRef))
	require.NoError(t, fkable.ProcessBlock(blk, nil))
	loader.UpdateIrreversibleData([]*bstream.PreprocessedBlock{{Block: blk}})
	require.NoError(t, eosdbDriver.Flush(ctx))

	respID, _, err := eosdbDriver.BlockIDBefore(ctx, blk.Time(), true) // direct timestamp
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = eosdbDriver.BlockIDBefore(ctx, blk.Time(), true) // direct timestamp
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = eosdbDriver.BlockIDAfter(ctx, time.Time{}, true) // first block since epoch
	assert.NoError(t, err)
	assert.Equal(t, blockID, respID)

	respID, _, err = eosdbDriver.BlockIDBefore(ctx, blk.Time().Add(-time.Second), true) // nothing before
	assert.Error(t, err)

	respID, _, err = eosdbDriver.BlockIDAfter(ctx, blk.Time().Add(time.Second), true) // nothing after
	assert.Error(t, err)
}

func testBlock(t *testing.T, id string, trxTraceJSONs ...string) *pbeos.Block {
	trxTraces := make([]*pbeos.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbeos.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbeos.Block{
		Id:                id,
		Number:            eos.BlockNum(id),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbeos.BlockHeader{
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
