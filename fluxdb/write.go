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

package fluxdb

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"go.uber.org/zap"
)

func (fdb *FluxDB) WriteBatch(ctx context.Context, w []*WriteRequest) error {
	ctx, span := dtracing.StartSpan(ctx, "write batch", "write_request_count", len(w))
	defer span.End()

	if err := fdb.isNextBlock(ctx, w[0].BlockNum); err != nil {
		return derr.Wrap(err, "next block check")
	}

	batch := fdb.store.NewBatch(zlog)

	for _, req := range w {

		if err := fdb.writeBlock(ctx, batch, req); err != nil {
			return derr.Wrap(err, "write block")
		}

		if err := batch.FlushIfFull(ctx); err != nil {
			return derr.Wrap(err, "flushing if full")
		}
	}

	if err := batch.Flush(ctx); err != nil {
		return derr.Wrap(err, "flush")
	}

	if sched := fdb.idxCache.IndexingSchedule(); len(sched) != 0 {
		err := fdb.IndexTables(ctx)
		if err != nil {
			return derr.Wrap(err, "index tables")
		}
	}

	return nil
}

func (fdb *FluxDB) VerifyAllShardsWritten() error {
	ctx := context.Background()

	seen := make(map[string]string)
	err := fdb.store.ScanLastShardsWrittenBlock(ctx, "shard-", func(key string, blockRef bstream.BlockRef) error {
		seen[strings.TrimPrefix(key, "shard-")] = blockRef.ID()
		return nil
	})

	if err != nil {
		return err
	}

	var lastSeenBlock string
	for i := 0; i < fdb.shardCount; i++ {
		key := fmt.Sprintf("%03d", i)
		seenBlock := seen[key]
		if seenBlock == "" {
			zlog.Info("verify all shards written: NO, shard missing", zap.String("missing", key))
			return nil
		}
		if lastSeenBlock == "" {
			lastSeenBlock = seenBlock
			continue
		}
		if seenBlock != lastSeenBlock {
			zlog.Info("verify all shards written: NO, block mismatch", zap.String("first_shard", lastSeenBlock), zap.String("second_shard", seenBlock))
			return nil
		}
	}

	zlog.Info("verify all shards written: YES, marking block for real-time injector", zap.String("block_id", lastSeenBlock))

	batch := fdb.store.NewBatch(zlog)
	batch.SetLast(lastBlockRowKey, []byte(lastSeenBlock))
	if err := batch.Flush(ctx); err != nil {
		return fmt.Errorf("flushing last block marker: %s", err)
	}

	return nil
}

func (fdb *FluxDB) writeBlock(ctx context.Context, batch store.Batch, w *WriteRequest) (err error) {
	for _, row := range w.AllWritableRows() {
		var value []byte
		if !row.isDeletion() {
			value = row.buildData()
		}

		batch.SetRow(row.rowKey(w.BlockNum), value)

		tableKey := row.tableKey()
		fdb.idxCache.IncCount(tableKey)
		if fdb.idxCache.shouldTriggerIndexing(tableKey) {
			fdb.idxCache.ScheduleIndex(tableKey, w.BlockNum)
		}
	}

	for _, abi := range w.ABIs {
		key := fmt.Sprintf("%s:%s", HexName(abi.Account), HexRevBlockNum(w.BlockNum))

		batch.SetABI(key, abi.PackedABI)
	}

	batch.SetLast(fdb.lastBlockKey(), []byte(hex.EncodeToString(w.BlockID)))

	return nil
}
