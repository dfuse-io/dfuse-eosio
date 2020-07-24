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
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func (fdb *FluxDB) ReadTabletAt(
	ctx context.Context,
	blockNum uint32,
	tablet Tablet,
	speculativeWrites []*WriteRequest,
) ([]TabletRow, error) {
	ctx, span := dtracing.StartSpan(ctx, "read tablet", "tablet", tablet, "block_num", blockNum)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading tablet", zap.Stringer("tablet", tablet), zap.Uint32("block_num", blockNum))

	startKey := tablet.KeyAt(0)
	endKey := tablet.KeyAt(blockNum + 1)
	rowByPrimaryKey := map[string]TabletRow{}

	idx, err := fdb.getIndex(ctx, blockNum, tablet)
	if err != nil {
		return nil, fmt.Errorf("fetch tablet index: %w", err)
	}

	if idx != nil {
		zlog.Debug("tablet index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		startKey = tablet.KeyAt(idx.AtBlockNum + 1)

		// Let's pre-allocated `rowByPrimaryKey` and `keys`, `rows` is likely to need at least as much rows as in the index itself
		rowByPrimaryKey = make(map[string]TabletRow, len(idx.Map))
		keys := make([]string, len(idx.Map))

		i := 0
		for primaryKey, blockNum := range idx.Map {
			keys[i] = string(tablet.KeyForRowAt(blockNum, primaryKey))
			i++
		}

		// Fetch all rows in the index.. could be millions
		// We need to batch so that the RowList, when serialized, doesn't blow up 1MB
		// We should batch in 10,000 key reads, we can parallelize those...
		chunkSize := 5000
		chunks := int(math.Ceil(float64(len(keys)) / float64(chunkSize)))

		zlog.Debug("reading index rows chunks", zap.Int("chunk_count", chunks))
		for i := 0; i < chunks; i++ {
			chunkStart := i * chunkSize
			chunkEnd := (i + 1) * chunkSize
			max := len(keys)
			if max < chunkEnd {
				chunkEnd = max
			}

			keysChunk := keys[chunkStart:chunkEnd]
			zlog.Debug("reading tablet index rows chunk", zap.Int("chunk_index", i), zap.Int("key_count", len(keysChunk)))

			keyRead := false
			err := fdb.store.FetchTabletRows(ctx, keysChunk, func(key string, value []byte) error {
				if len(value) == 0 {
					return fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in a tablet index, row %s", key)
				}

				row, err := tablet.NewRowFromKV(key, value)
				if err != nil {
					return fmt.Errorf("tablet index new row %s: %w", key, err)
				}

				rowByPrimaryKey[string(row.PrimaryKey())] = row

				keyRead = true
				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("reading tablet index rows chunk %d: %w", i, err)
			}

			if !keyRead {
				return nil, fmt.Errorf("reading a tablet index yielded no row: %s", keysChunk)
			}
		}

		zlog.Debug("finished reconciling index")
	}

	zlog.Debug("reading tablet rows from database",
		zap.Int("row_count", len(rowByPrimaryKey)),
		zap.Bool("index_found", idx != nil),
		zap.Int("index_row_count", idx.RowCount()),
		zap.String("start_key", startKey),
		zap.String("end_key", endKey),
	)

	deletedCount := 0
	updatedCount := 0

	err = fdb.store.ScanTabletRows(ctx, startKey, endKey, func(key string, value []byte) error {
		row, err := tablet.NewRowFromKV(key, value)
		if err != nil {
			return fmt.Errorf("tablet new row %s: %w", key, err)
		}

		if isDeletionRow(row) {
			deletedCount++
			delete(rowByPrimaryKey, string(row.PrimaryKey()))

			return nil
		}

		updatedCount++
		rowByPrimaryKey[string(row.PrimaryKey())] = row

		return nil
	})

	if err != nil {
		return nil, err
	}

	zlog.Debug("reading tablet rows from speculative writes",
		zap.Int("row_count", len(rowByPrimaryKey)),
		zap.Int("speculative_write_count", len(speculativeWrites)),
	)

	for _, speculativeWrite := range speculativeWrites {
		for _, speculativeRow := range speculativeWrite.TabletRows {
			if speculativeRow.Tablet() != tablet {
				continue
			}

			if isDeletionRow(speculativeRow) {
				delete(rowByPrimaryKey, string(speculativeRow.PrimaryKey()))
			} else {
				rowByPrimaryKey[string(speculativeRow.PrimaryKey())] = speculativeRow
			}
		}
	}

	zlog.Debug("post-processing tablet rows", zap.Int("row_count", len(rowByPrimaryKey)))

	i := 0
	rows := make([]TabletRow, len(rowByPrimaryKey))
	for _, row := range rowByPrimaryKey {
		rows[i] = row
		i++
	}

	sort.Slice(rows, func(i, j int) bool { return string(rows[i].PrimaryKey()) < string(rows[j].PrimaryKey()) })

	zlog.Info("finished reading tablet rows", zap.Int("deleted_count", deletedCount), zap.Int("updated_count", updatedCount))
	return rows, nil
}

func (fdb *FluxDB) ReadTabletRowAt(
	ctx context.Context,
	blockNum uint32,
	tablet Tablet,
	primaryKey string,
	speculativeWrites []*WriteRequest,
) (TabletRow, error) {
	ctx, span := dtracing.StartSpan(ctx, "read tablet row", "tablet", tablet, "block_num", blockNum)
	defer span.End()

	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("reading tablet row",
		zap.Stringer("tablet", tablet),
		zap.Uint32("block_num", blockNum),
		zap.String("primary_key", primaryKey),
	)

	idx, err := fdb.getIndex(ctx, blockNum, tablet)
	if err != nil {
		return nil, fmt.Errorf("fetch tablet index: %w", err)
	}

	startKey := tablet.KeyAt(0)
	endKey := tablet.KeyAt(blockNum + 1)
	var row TabletRow
	if idx != nil {
		zlogger.Debug("tablet index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		startKey = tablet.KeyAt(idx.AtBlockNum + 1)

		if blockNum, ok := idx.Map[primaryKey]; ok {
			rowKey := string(tablet.KeyForRowAt(blockNum, primaryKey))
			zlogger.Debug("reading index row", zap.String("row_key", rowKey))

			value, err := fdb.store.FetchTabletRow(ctx, rowKey)
			if err == store.ErrNotFound {
				return nil, fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in an index, row %s", rowKey)
			}
			if err != nil {
				return nil, fmt.Errorf("reading tablet index row %q: %w", rowKey, err)
			}
			if len(value) <= 0 {
				row, err = tablet.NewRowFromKV(rowKey, value)
				if err != nil {
					return nil, fmt.Errorf("could not create table from key value with row key %q: %w", rowKey, err)
				}
			}

		}
		zlogger.Debug("finished reconciling index", zap.Bool("row_exist", row != nil))
	}

	zlogger.Debug("reading tablet row from database",
		zap.Bool("row_exist", row != nil),
		zap.Bool("index_found", idx != nil),
		zap.String("start_key", startKey),
		zap.String("end_key", endKey),
	)

	deletedCount := 0
	updatedCount := 0

	err = fdb.store.ScanTabletRows(ctx, startKey, endKey, func(key string, value []byte) error {
		candidateRow, err := tablet.NewRowFromKV(key, value)
		if err != nil {
			return fmt.Errorf("tablet new row %s: %w", key, err)
		}

		if candidateRow.PrimaryKey() != primaryKey {
			return nil
		}

		if isDeletionRow(candidateRow) {
			row = nil
			deletedCount++

			return nil
		}

		updatedCount++
		row = candidateRow
		return nil
	})
	if err != nil {
		return nil, err
	}

	zlogger.Debug("reading tablet row from speculative writes",
		zap.Int("deleted_count", deletedCount),
		zap.Int("updated_count", updatedCount),
		zap.Int("speculative_write_count", len(speculativeWrites)),
	)

	for _, speculativeWrite := range speculativeWrites {
		for _, speculativeRow := range speculativeWrite.TabletRows {
			if speculativeRow.Tablet() != tablet {
				continue
			}

			if speculativeRow.PrimaryKey() != primaryKey {
				continue
			}

			if isDeletionRow(speculativeRow) {
				deletedCount++
				row = nil
			} else {
				updatedCount++
				row = speculativeRow
			}
		}
	}

	zlogger.Info("finished reading tablet row",
		zap.Int("deleted_count", deletedCount),
		zap.Int("updated_count", updatedCount),
		zap.String("primary_key", primaryKey),
	)
	return row, nil
}

func (fdb *FluxDB) ReadSingletEntryAt(
	ctx context.Context,
	singlet Singlet,
	blockNum uint32,
	speculativeWrites []*WriteRequest,
) (SingletEntry, error) {
	ctx, span := dtracing.StartSpan(ctx, "read singlet entry", "singlet", singlet, "block_num", blockNum)
	defer span.End()

	// We are using inverted block num, so we are scanning from highest block num (request block num) to lowest block (0)
	startKey := singlet.KeyAt(blockNum)
	endKey := singlet.KeyAt(0)

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading singlet entry from database", zap.Stringer("singlet", singlet), zap.Uint32("block_num", blockNum), zap.String("start_key", startKey), zap.String("end_key", endKey))

	var entry SingletEntry
	key, value, err := fdb.store.FetchSingletEntry(ctx, startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("db fetch single entry: %w", err)
	}

	// If there is a key set (record found) and the value is non-nil (it's a deleted entry), then populated entry
	if key != "" && len(value) > 0 {
		entry, err = singlet.NewEntryFromKV(key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to create single tablet row %s: %w", key, err)
		}
	}

	zlog.Debug("reading singlet entry from speculative writes", zap.Bool("db_exist", entry != nil), zap.Int("speculative_write_count", len(speculativeWrites)))
	for _, writeRequest := range speculativeWrites {
		for _, speculativeEntry := range writeRequest.SingletEntries {
			if entry.Singlet() != singlet {
				continue
			}

			if isDeletionEntry(speculativeEntry) {
				entry = nil
			} else {
				entry = speculativeEntry
			}
		}
	}

	zlog.Debug("finished reading singlet entry", zap.Bool("entry_exist", entry != nil))
	return entry, nil
}

func (fdb *FluxDB) HasSeenAnyRowForTablet(ctx context.Context, tablet Tablet) (exists bool, err error) {
	ctx, span := dtracing.StartSpan(ctx, "has seen tablet row", "tablet", tablet.String())
	defer span.End()

	return fdb.store.HasTabletRow(ctx, tablet.Key())
}

func (fdb *FluxDB) FetchLastWrittenBlock(ctx context.Context) (out bstream.BlockRef, err error) {
	zlogger := logging.Logger(ctx, zlog)

	lastBlockKey := fdb.lastBlockKey()
	out, err = fdb.store.FetchLastWrittenBlock(ctx, lastBlockKey)
	if err == store.ErrNotFound {
		zlogger.Info("last written block empty, returning block ID 0")
		return bstream.BlockRefEmpty, nil
	}

	if err != nil {
		return out, fmt.Errorf("kv store: %w", err)
	}

	zlogger.Debug("last written block", zap.Stringer("block", out))
	return
}

func (fdb *FluxDB) CheckCleanDBForSharding() error {
	_, err := fdb.store.FetchLastWrittenBlock(context.Background(), lastBlockRowKey)
	if err == store.ErrNotFound {
		// When there is nothing, it's what we expect, so there is no error
		return nil
	}

	if err != nil {
		return err
	}

	// At this point, the fetch return something viable, this is not correct for sharding reprocessing
	return errors.New("live injector's marker of last written block present, expected no element to exist")
}

func (fdb *FluxDB) lastBlockKey() string {
	if fdb.IsSharding() {
		return fmt.Sprintf("shard-%03d", fdb.shardIndex)
	}

	return lastBlockRowKey
}
