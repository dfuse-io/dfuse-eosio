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
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
)

func (fdb *FluxDB) WriteBatch(ctx context.Context, w []*WriteRequest) error {
	ctx, span := dtracing.StartSpan(ctx, "write batch", "write_request_count", len(w))
	defer span.End()

	if err := fdb.isNextBlock(ctx, w[0].BlockNum); err != nil {
		return fmt.Errorf("next block check: %w", err)
	}

	batch := fdb.store.NewBatch(zlog)

	for _, req := range w {
		if err := fdb.writeBlock(ctx, batch, req); err != nil {
			return fmt.Errorf("write block: %w", err)
		}

		if err := batch.FlushIfFull(ctx); err != nil {
			return fmt.Errorf("flushing if full: %w", err)
		}
	}

	if err := batch.Flush(ctx); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	if sched := fdb.idxCache.IndexingSchedule(); len(sched) != 0 {
		err := fdb.IndexTables(ctx)
		if err != nil {
			return fmt.Errorf("index tables: %w", err)
		}
	}

	return nil
}

func (fdb *FluxDB) VerifyAllShardsWritten(ctx context.Context) (string, error) {
	seen := make(map[string]string)
	if err := fdb.store.ScanLastShardsWrittenBlock(ctx, "shard-", func(key string, blockRef bstream.BlockRef) error {
		seen[strings.TrimPrefix(key, "shard-")] = blockRef.ID()
		return nil
	}); err != nil {
		return "", err
	}

	shardToBlockID := make(map[string]string)

	var referenceBlock string
	for i := 0; i < fdb.shardCount; i++ {
		key := fmt.Sprintf("%03d", i)
		shardToBlockID[key] = seen[key]
		if i == 0 {
			referenceBlock = seen[key]
		}
	}

	var faultyShards []string
	var missingShards []string
	for key, seenBlock := range shardToBlockID {
		if seenBlock == "" {
			missingShards = append(missingShards, key)
		}
		if seenBlock != referenceBlock {
			faultyShards = append(faultyShards, key)
		}
	}

	var err error
	if missingShards != nil {
		err = fmt.Errorf("missing shards: %v", missingShards)
	}

	if faultyShards != nil {
		err = fmt.Errorf("shards not matching reference block %s (shards %v): %w", referenceBlock, faultyShards, err)
	}

	return referenceBlock, err

}

func (fdb *FluxDB) UpdateGlobalLastBlockID(ctx context.Context, blockID string) error {
	batch := fdb.store.NewBatch(zlog)
	batch.SetLast(lastBlockRowKey, []byte(blockID))
	if err := batch.Flush(ctx); err != nil {
		return fmt.Errorf("flushing last block marker: %w", err)
	}

	return nil
}

func (fdb *FluxDB) writeBlock(ctx context.Context, batch store.Batch, w *WriteRequest) (err error) {
	for _, entry := range w.SigletEntries {
		var value []byte
		if !isDeletionEntry(entry) {
			value = entry.Value()
		}

		batch.SetRow(string(entry.Key()), value)
	}

	for _, row := range w.TabletRows {
		var value []byte
		if !isDeletionRow(row) {
			value = row.Value()
		}

		batch.SetRow(string(row.Key()), value)

		tabletKey := string(row.Tablet().Key())

		fdb.idxCache.IncCount(tabletKey)
		if fdb.idxCache.shouldTriggerIndexing(tabletKey) {
			fdb.idxCache.ScheduleIndex(tabletKey, w.BlockNum)
		}
	}

	batch.SetLast(fdb.lastBlockKey(), []byte(hex.EncodeToString(w.BlockID)))
	return nil
}
