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
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func (fdb *FluxDB) IndexTables(ctx context.Context) error {
	ctx, span := dtracing.StartSpan(ctx, "index tables")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("indexing tables")

	batch := fdb.store.NewBatch(zlog)

	for tablet, blockNum := range fdb.idxCache.scheduleIndexing {
		zlog.Debug("indexing table", zap.Stringer("tablet", tablet), zap.Uint32("block_num", blockNum))

		if err := batch.FlushIfFull(ctx); err != nil {
			return fmt.Errorf("flush if full: %w", err)
		}

		zlog.Debug("checking if index already exist in cache")
		index := fdb.idxCache.GetIndex(tablet)
		if index == nil {
			zlog.Debug("index not in cache")

			var err error
			index, err = fdb.getIndex(ctx, blockNum, tablet)
			if err != nil {
				return fmt.Errorf("get index %s (%d): %w", tablet, blockNum, err)
			}

			if index == nil {
				zlog.Debug("index does not exist yet, creating empty one")
				index = NewTableIndex()
			}
		}

		startKey := tablet.KeyAt(index.AtBlockNum + 1)
		endKey := tablet.KeyAt(blockNum + 1)

		zlog.Debug("reading table rows for indexation", zap.String("first_row_key", startKey), zap.String("last_row_key", endKey))

		count := 0
		err := fdb.store.ScanTabletRows(ctx, startKey, endKey, func(key string, value []byte) error {
			row, err := tablet.NewRowFromKV(key, value)
			if err != nil {
				return fmt.Errorf("couldn't parse row key %q: %w", key, err)
			}

			count++

			if len(value) == 0 {
				delete(index.Map, row.PrimaryKey())
			} else {
				index.Map[row.PrimaryKey()] = blockNum
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("read rows: %w", err)
		}

		index.AtBlockNum = blockNum
		index.Squelched = uint32(count)

		zlog.Debug("about to marshal index to binary",
			zap.Stringer("tablet", tablet),
			zap.Uint32("at_block_num", index.AtBlockNum),
			zap.Uint32("squelched_count", index.Squelched),
			zap.Int("row_count", len(index.Map)),
		)

		snapshot, err := index.MarshalBinary(ctx, tablet)
		if err != nil {
			return fmt.Errorf("unable to marshal table index to binary: %w", err)
		}

		indexKey := tablet.Key() + "/" + HexRevBlockNum(index.AtBlockNum)

		byteCount := len(snapshot)
		if byteCount > 25000000 {
			zlog.Warn("table index pretty heavy", zap.String("index_key", indexKey), zap.Int("byte_count", byteCount))
		}

		batch.SetIndex(indexKey, snapshot)

		zlog.Debug("caching index in index cache", zap.String("index_key", indexKey), zap.Stringer("tablet", tablet))
		fdb.idxCache.CacheIndex(tablet, index)
		fdb.idxCache.ResetCounter(tablet)
		delete(fdb.idxCache.scheduleIndexing, tablet)
	}

	if err := batch.Flush(ctx); err != nil {
		return fmt.Errorf("final flush: %w", err)
	}

	return nil
}

func (fdb *FluxDB) getIndex(ctx context.Context, blockNum uint32, tablet Tablet) (index *TableIndex, err error) {
	ctx, span := dtracing.StartSpan(ctx, "get index")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("fetching table index from database", zap.Stringer("tablet", tablet), zap.Uint32("block_num", blockNum))

	tabletKey := string(tablet.Key())
	prefixKey := tabletKey + "/"
	startIndexKey := prefixKey + HexRevBlockNum(blockNum)

	zlog.Debug("reading table index row", zap.String("start_index_key", startIndexKey))
	rowKey, rawIndex, err := fdb.store.FetchIndex(ctx, tabletKey, prefixKey, startIndexKey)
	if err == store.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	indexBlockNum, err := chunkKeyRevBlockNum(rowKey, prefixKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't infer block num in table index's row key: %w", err)
	}

	index, err = NewTableIndexFromBinary(ctx, tablet, indexBlockNum, rawIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal binary index: %w", err)
	}

	return index, nil
}

type indexCache struct {
	lastIndexes      map[Tablet]*TableIndex
	lastCounters     map[Tablet]int
	scheduleIndexing map[Tablet]uint32
}

func newIndexCache() *indexCache {
	return &indexCache{
		lastIndexes:      make(map[Tablet]*TableIndex),
		lastCounters:     make(map[Tablet]int),
		scheduleIndexing: make(map[Tablet]uint32),
	}
}

func (t *indexCache) GetIndex(tablet Tablet) *TableIndex {
	return t.lastIndexes[tablet]
}

func (t *indexCache) CacheIndex(tablet Tablet, tableIndex *TableIndex) {
	t.lastIndexes[tablet] = tableIndex
}

func (t *indexCache) GetCount(tablet Tablet) int {
	return t.lastCounters[tablet]
}

func (t *indexCache) IncCount(tablet Tablet) {
	t.lastCounters[tablet]++
}

func (t *indexCache) ResetCounter(tablet Tablet) {
	t.lastCounters[tablet] = 0
}

// This algorithm determines the space between the indexes
func (t *indexCache) shouldTriggerIndexing(tablet Tablet) bool {
	mutatedRowsCount := t.lastCounters[tablet]
	if mutatedRowsCount < 1000 {
		return false
	}

	lastIndex := t.lastIndexes[tablet]
	if lastIndex == nil {
		return true
	}

	tableRowsCount := len(lastIndex.Map)

	if tableRowsCount > 50000 && mutatedRowsCount < 5000 {
		return false
	}

	if tableRowsCount > 100000 && mutatedRowsCount < 10000 {
		return false
	}

	return true
}

func (t *indexCache) ScheduleIndex(tablet Tablet, blockNum uint32) {
	t.scheduleIndexing[tablet] = blockNum
}

func (t *indexCache) IndexingSchedule() map[Tablet]uint32 {
	return t.scheduleIndexing
}

type TableIndex struct {
	AtBlockNum uint32
	Squelched  uint32
	Map        map[string]uint32 // Map[primaryKey] => blockNum
}

func NewTableIndex() *TableIndex {
	return &TableIndex{Map: make(map[string]uint32)}
}

func (index *TableIndex) RowCount() int {
	if index == nil {
		return 0
	}

	return len(index.Map)
}

func NewTableIndexFromBinary(ctx context.Context, tablet Tablet, atBlockNum uint32, buffer []byte) (*TableIndex, error) {
	ctx, span := dtracing.StartSpan(ctx, "new table index from binary", "tablet", tablet, "block_num", atBlockNum)
	defer span.End()

	// Byte count for primary key + 4 bytes for block num value
	primaryKeyByteCount := tablet.PrimaryKeyByteCount()
	entryByteCount := primaryKeyByteCount + 4

	// First 16 bytes are reserved to keep stats in there..
	byteCount := len(buffer)
	if (byteCount-16) < 0 || (byteCount-16)%entryByteCount != 0 {
		return nil, fmt.Errorf("unable to unmarshal table index: %d bytes alignment + 16 bytes metadata is off (has %d bytes)", entryByteCount, byteCount)
	}

	mapping := map[string]uint32{}
	for pos := 16; pos < byteCount; pos += entryByteCount {
		primaryKey, err := tablet.DecodePrimaryKey(buffer[pos:])
		if err != nil {
			return nil, fmt.Errorf("unable to read primary key for tablet %q: %w", tablet, err)
		}

		blockNumPtr := bigEndian.Uint32(buffer[pos+primaryKeyByteCount:])
		mapping[primaryKey] = blockNumPtr
	}

	return &TableIndex{
		AtBlockNum: atBlockNum,
		Squelched:  bigEndian.Uint32(buffer[:4]),
		Map:        mapping,
	}, nil
}

func (index *TableIndex) MarshalBinary(ctx context.Context, tablet Tablet) ([]byte, error) {
	ctx, span := dtracing.StartSpan(ctx, "marshal table index to binary", "tablet", tablet)
	defer span.End()

	primaryKeyByteCount := tablet.PrimaryKeyByteCount()
	entryByteCount := primaryKeyByteCount + 4 // Byte count for primary key + 4 bytes for block num value

	snapshot := make([]byte, entryByteCount*len(index.Map)+16)
	bigEndian.PutUint32(snapshot, index.Squelched)

	pos := 16
	for primaryKey, blockNum := range index.Map {
		err := tablet.EncodePrimaryKey(snapshot[pos:], primaryKey)
		if err != nil {
			return nil, fmt.Errorf("unable to read primary key for tablet %q: %w", tablet, err)
		}

		bigEndian.PutUint32(snapshot[pos+primaryKeyByteCount:], blockNum)
		pos += entryByteCount
	}

	return snapshot, nil
}

func (index *TableIndex) String() string {
	builder := &strings.Builder{}
	fmt.Fprintln(builder, "INDEX:")

	fmt.Fprintln(builder, "  * At block num:", index.AtBlockNum)
	fmt.Fprintln(builder, "  * Squelches:", index.Squelched)
	var keys []string
	for primKey := range index.Map {
		keys = append(keys, primKey)
	}

	sort.Strings(keys)

	fmt.Fprintln(builder, "Snapshot (primkey -> blocknum)")
	for _, k := range keys {
		fmt.Fprintf(builder, "  %s -> %d\n", k, index.Map[k])
	}

	return builder.String()
}

type indexPrimaryKeyReader = func(buffer []byte) (string, error)
type indexPrimaryKeyWriter = func(primaryKey string, buffer []byte) error

func twoUint64PrimaryKeyReaderFactory(tag string) indexPrimaryKeyReader {
	return func(buffer []byte) (string, error) {
		if len(buffer) < 16 {
			return "", fmt.Errorf("%s primary key reader: not enough bytes to read, %d bytes left, wants %d", tag, len(buffer), 16)
		}

		chunk1, err := readOneUint64(buffer)
		if err != nil {
			return "", fmt.Errorf("%s primary key reader, chunk #1: %w", tag, err)
		}

		chunk2, err := readOneUint64(buffer[8:])
		if err != nil {
			return "", fmt.Errorf("%s primary key reader, chunk #2: %w", tag, err)
		}

		return strings.Join([]string{chunk1, chunk2}, ":"), nil
	}
}

func readOneUint64(buffer []byte) (string, error) {
	if len(buffer) < 8 {
		return "", fmt.Errorf("not enough bytes to read uint64, %d bytes left, wants %d", len(buffer), 8)
	}

	return fmt.Sprintf("%016x", bigEndian.Uint64(buffer)), nil
}

func twoUint64PrimaryKeyWriterFactory(tag string) indexPrimaryKeyWriter {
	return func(primaryKey string, buffer []byte) error {

		chunks := strings.Split(primaryKey, ":")
		if len(chunks) != 2 {
			return fmt.Errorf("%s primary key should have 2 chunks, got %d", tag, len(chunks))
		}

		err := writeOneUint64(chunks[0], buffer)
		if err != nil {
			return fmt.Errorf("%s primary key writer, chunk #1: %w", tag, err)
		}

		err = writeOneUint64(chunks[1], buffer[8:])
		if err != nil {
			return fmt.Errorf("%s primary key writer, chunk #2: %w", tag, err)
		}

		return nil
	}
}

func writeOneUint64(primaryKey string, buffer []byte) error {
	value, err := strconv.ParseUint(primaryKey, 16, 64)
	if err != nil {
		return fmt.Errorf("unable to transform primary key to uint64: %w", err)
	}

	bigEndian.PutUint64(buffer, value)
	return nil
}
