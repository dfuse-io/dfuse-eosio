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
	"strconv"
	"strings"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (fdb *FluxDB) ReadTable(ctx context.Context, r *ReadTableRequest) (resp *ReadTableResponse, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading state table", zap.Reflect("request", r))

	rowData := make(map[string]*TableRow)
	rowUpdated := func(blockNum uint32, primaryKey string, value []byte) error {
		if len(value) < 8 {
			return errors.New("table data index mappings should contain at least the payer")
		}

		payer := big.Uint64(value)
		tableDataPrimaryKey, err := strconv.ParseUint(primaryKey, 16, 64)
		if err != nil {
			return fmt.Errorf("unable to transform table data primary key to uint64: %w", err)
		}

		rowData[primaryKey] = &TableRow{tableDataPrimaryKey, payer, value[8:], blockNum}

		return nil
	}

	rowDeleted := func(_ uint32, primaryKey string) error {
		delete(rowData, primaryKey)
		return nil
	}

	tableKey := r.tableKey()
	err = fdb.read(ctx, tableKey, r.BlockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read rows for table key %q: %w", tableKey, err)
	}

	// abi, err := fdb.GetABI(ctx, r.BlockNum, r.Account, r.SpeculativeWrites)
	// if err != nil {
	// 	return nil, err
	// }

	// zlog.Debug("handling speculative writes", zap.Int("write_count", len(r.SpeculativeWrites)))
	// for _, blockWrite := range r.SpeculativeWrites {
	// 	for _, row := range blockWrite.FluxRows {
	// 		if r.Account != row.Account || r.Scope != row.Scope || r.Table != row.Table {
	// 			continue
	// 		}

	// 		stringPrimaryKey := fmt.Sprintf("%016x", row.PrimKey)

	// 		if row.Deletion {
	// 			delete(rowData, stringPrimaryKey)
	// 		} else {
	// 			rowData[stringPrimaryKey] = &TableRow{
	// 				Key:      row.PrimKey,
	// 				Payer:    row.Payer,
	// 				Data:     row.Data,
	// 				BlockNum: blockWrite.BlockNum,
	// 			}
	// 		}
	// 	}
	// }

	zlog.Debug("post-processing table rows", zap.Int("row_count", len(rowData)))
	var rows []*TableRow
	for _, row := range rowData {
		rows = append(rows, row)
	}

	zlog.Debug("sorting table rows")
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })

	return &ReadTableResponse{
		// ABI:  abi,
		Rows: rows,
	}, nil
}

func (fdb *FluxDB) HasSeenPublicKeyOnce(ctx context.Context, publicKey string) (exists bool, err error) {
	return fdb.hasRowKeyPrefix(ctx, fmt.Sprintf("ka2:%s", publicKey))
}

func (fdb *FluxDB) HasSeenTableOnce(
	ctx context.Context,
	account eos.AccountName,
	table eos.TableName,
) (exists bool, err error) {
	return fdb.hasRowKeyPrefix(ctx, fmt.Sprintf("ts:%016x:%016x", N(string(account)), N(string(table))))
}

func (fdb *FluxDB) hasRowKeyPrefix(ctx context.Context, keyPrefix string) (exists bool, err error) {
	ctx, span := dtracing.StartSpan(ctx, "has row key prefix", "key_prefix", keyPrefix)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("has row key prefix", zap.String("key_prefix", keyPrefix))

	return fdb.store.HasTabletRow(ctx, keyPrefix)
}

func (fdb *FluxDB) read(
	ctx context.Context,
	tableKey string,
	blockNum uint32,
	rowUpdated func(blockNum uint32, primaryKey string, value []byte) error,
	rowDeleted func(blockNum uint32, primaryKey string) error,
) error {
	ctx, span := dtracing.StartSpan(ctx, "read table", "table_key", tableKey, "block_num", blockNum)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading rows from database", zap.String("table_key", tableKey), zap.Uint32("block_num", blockNum))

	idx, err := fdb.getIndex(ctx, tableKey, blockNum)
	if err != nil {
		return err
	}

	firstRowKey := tableKey + ":00000000"
	lastRowKey := tableKey + ":" + HexBlockNum(blockNum+1)

	if idx != nil {
		zlog.Debug("index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		firstRowKey = tableKey + "/" + HexBlockNum(idx.AtBlockNum+1)

		var keys []string
		for primaryKey, blockNum := range idx.Map {
			keys = append(keys, fmt.Sprintf("%s:%08x:%s", tableKey, blockNum, primaryKey))
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

			zlog.Debug("reading index rows chunk", zap.Int("key_count", len(keysChunk)))
			keyRead := false
			err := fdb.store.FetchTabletRows(ctx, keysChunk, func(rowKey string, value []byte) error {
				if len(value) == 0 {
					return fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in an index, row %s", rowKey)
				}

				_, rowBlockNum, primaryKey, err := explodeWritableRowKey(rowKey)
				if err != nil {
					return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
				}

				err = rowUpdated(rowBlockNum, primaryKey, value)
				if err != nil {
					return fmt.Errorf("rowUpdated callback failed for row %q (indexed rows): %w", rowKey, err)
				}

				keyRead = true
				return nil
			})

			if err != nil {
				return fmt.Errorf("reading keys chunks: %w", err)
			}

			if !keyRead {
				return fmt.Errorf("reading a indexed key yielded no row: %s", keysChunk)
			}
		}

		zlog.Debug("finished reconciling index")
	}

	// check for latest index based on r.BlockNum
	// go through keys from last index's `AtBlockNum`, through to `BlockNum`
	// fetch all the keys within the index
	// parse all rows following the index, and keep the latest, so simply override with incoming rows..

	zlog.Debug("reading rows range from database", zap.String("first_row_key", firstRowKey), zap.String("last_row_key", lastRowKey))

	deletedCount := 0
	updatedCount := 0

	err = fdb.store.ScanTabletRows(ctx, firstRowKey, lastRowKey, func(rowKey string, value []byte) error {
		_, rowBlockNum, primaryKey, err := explodeWritableRowKey(rowKey)
		if err != nil {
			return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
		}

		if len(value) == 0 {
			err := rowDeleted(rowBlockNum, primaryKey)
			if err != nil {
				return fmt.Errorf("rowDeleted callback failed for row %q (live rows): %w", rowKey, err)
			}

			deletedCount++
			return nil
		}

		err = rowUpdated(rowBlockNum, primaryKey, value)
		if err != nil {
			return fmt.Errorf("rowUpdated callback failed for row %q (live rows): %w", rowKey, err)
		}

		updatedCount++
		return nil
	})

	if err != nil {
		return err
	}

	zlog.Debug("finished reading rows from database", zap.Int("deleted_count", deletedCount), zap.Int("updated_count", updatedCount))
	return nil
}

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

	idx, err := fdb.getIndex2(ctx, blockNum, tablet)
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

	idx, err := fdb.getIndex2(ctx, blockNum, tablet)
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

func (fdb *FluxDB) ReadSigletEntryAt(
	ctx context.Context,
	siglet Siglet,
	blockNum uint32,
	speculativeWrites []*WriteRequest,
) (SigletEntry, error) {
	ctx, span := dtracing.StartSpan(ctx, "read singlet entry", "siglet", siglet, "block_num", blockNum)
	defer span.End()

	// We are using inverted block num, so we are scanning from highest block num (request block num) to lowest block (0)
	startKey := siglet.KeyAt(blockNum + 1)
	endKey := siglet.KeyAt(0)

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading siglet entry from database", zap.Stringer("siglet", siglet), zap.Uint32("block_num", blockNum), zap.String("start_key", startKey), zap.String("end_key", endKey))

	var entry SigletEntry
	key, value, err := fdb.store.FetchSigletEntry(ctx, startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("db fetch single entry: %w", err)
	}

	// If there is a key set (record found) and the value is non-nil (it's a deleted entry), then populated entry
	if key != "" && len(value) > 0 {
		entry, err = siglet.NewEntryFromKV(key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to create single tablet row %s: %w", key, err)
		}
	}

	zlog.Debug("reading siglet entry from speculative writes", zap.Bool("db_exist", entry != nil), zap.Int("speculative_write_count", len(speculativeWrites)))
	for _, writeRequest := range speculativeWrites {
		for _, speculativeEntry := range writeRequest.SigletEntries {
			if entry.Siglet() != siglet {
				continue
			}

			if isDeletionEntry(speculativeEntry) {
				entry = nil
			} else {
				entry = speculativeEntry
			}
		}
	}

	zlog.Debug("finished reading siglet entry", zap.Bool("entry_exist", entry != nil))
	return entry, nil
}

func (fdb *FluxDB) FetchLastWrittenBlock(ctx context.Context) (out bstream.BlockRef, err error) {
	zlogger := logging.Logger(ctx, zlog)

	lastBlockKey := fdb.lastBlockKey()
	out, err = fdb.store.FetchLastWrittenBlock(ctx, lastBlockKey)
	if err == store.ErrNotFound {
		zlogger.Info("last written block empty, returning block ID 0")
		return bstream.BlockRefFromID(strings.Repeat("00", 32)), nil
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

func (fdb *FluxDB) isNextBlock(ctx context.Context, writeBlockNum uint32) error {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("checking if is next block", zap.Uint32("block_num", writeBlockNum))

	lastBlock, err := fdb.FetchLastWrittenBlock(ctx)
	if err != nil {
		return err
	}

	lastBlockNum := uint32(lastBlock.Num())
	if lastBlockNum != writeBlockNum-1 && lastBlockNum != 0 && lastBlockNum != 1 {
		return fmt.Errorf("block %d does not follow last block %d in db", writeBlockNum, lastBlockNum)
	}

	return nil
}
