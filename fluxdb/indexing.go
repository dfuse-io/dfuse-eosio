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

/**
PAUSING indexing effort. Here are a few benchmarks:

$ curl -sS "http://localhost:8080/v1/read?block_num=6999000&account=eosio&scope=eosio&table=voters&json=true&with_named_key=1"
* Takes 4.036s
* Downloads 34MB of data, ~200k records
* Navigate records in 60ms
* Sorts in 25ms
* Takes MOST of the time JSON-encoding the output.
* With `json=false`, it boils down to 500ms
* A few mils shaved off (10?) with `with_named_key=0`.. we have the `owner` in there anyway.
* The largest optimization would be to have a high performance ABI raw-to-json decoder.


*/

func (fdb *FluxDB) IndexTables(ctx context.Context) error {
	ctx, span := dtracing.StartSpan(ctx, "index tables")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("indexing tables")

	batch := fdb.store.NewBatch(zlog)

	for tableKey, blockNum := range fdb.idxCache.scheduleIndexing {
		zlog.Debug("indexing table", zap.String("table_key", tableKey), zap.Uint32("block_num", blockNum))

		if err := batch.FlushIfFull(ctx); err != nil {
			return fmt.Errorf("flush if full: %w", err)
		}

		zlog.Debug("checking if index already exist in cache")
		index := fdb.idxCache.GetIndex(tableKey)
		if index == nil {
			zlog.Debug("index not in cache")

			var err error
			index, err = fdb.getIndex(ctx, tableKey, blockNum)
			if err != nil {
				return fmt.Errorf("get index %s (%d): %w", tableKey, blockNum, err)
			}

			if index == nil {
				zlog.Debug("index does not exist yet, creating empty one")
				index = NewTableIndex()
			}
		}

		firstRowKey := tableKey + ":" + HexBlockNum(index.AtBlockNum+1)
		lastRowKey := tableKey + ":" + HexBlockNum(blockNum+1)

		zlog.Debug("reading table rows", zap.String("first_row_key", firstRowKey), zap.String("last_row_key", lastRowKey))

		count := 0
		err := fdb.store.ScanTabletRows(ctx, firstRowKey, lastRowKey, func(key string, value []byte) error {
			_, blockNum, primKey, err := explodeWritableRowKey(key)
			if err != nil {
				return fmt.Errorf("couldn't parse row key %q: %w", key, err)
			}

			count++

			if len(value) == 0 {
				delete(index.Map, primKey)
			} else {
				index.Map[primKey] = blockNum
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("read rows: %w", err)
		}

		index.AtBlockNum = blockNum
		index.Squelched = uint32(count)

		zlog.Debug("about to marshal index to binary",
			zap.String("table_key", tableKey),
			zap.Uint32("at_block_num", index.AtBlockNum),
			zap.Uint32("squelched_count", index.Squelched),
			zap.Int("row_count", len(index.Map)),
		)

		snapshot, err := index.MarshalBinary(ctx, tableKey)
		if err != nil {
			return fmt.Errorf("unable to marshal table index to binary: %w", err)
		}

		indexKey := tableKey + ":" + HexRevBlockNum(index.AtBlockNum)

		byteCount := len(snapshot)
		if byteCount > 25000000 {
			zlog.Warn("table index pretty heavy", zap.String("index_key", indexKey), zap.Int("byte_count", byteCount))
		}

		batch.SetIndex(indexKey, snapshot)

		zlog.Debug("caching index in index cache", zap.String("index_key", indexKey), zap.String("table_key", tableKey))
		fdb.idxCache.CacheIndex(tableKey, index)
		fdb.idxCache.ResetCounter(tableKey)
		delete(fdb.idxCache.scheduleIndexing, tableKey)
	}

	if err := batch.Flush(ctx); err != nil {
		return fmt.Errorf("final flush: %w", err)
	}

	return nil
}

func (fdb *FluxDB) getIndex(ctx context.Context, tableKey string, blockNum uint32) (*TableIndex, error) {
	ctx, span := dtracing.StartSpan(ctx, "get index")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("fetching table index from database", zap.String("table_key", tableKey), zap.Uint32("block_num", blockNum))

	prefixKey := tableKey + ":"
	startIndexKey := prefixKey + HexRevBlockNum(blockNum)

	zlog.Debug("reading table index row", zap.String("start_index_key", startIndexKey))
	rowKey, rawIndex, err := fdb.store.FetchIndex(ctx, tableKey, prefixKey, startIndexKey)
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

	index, err := NewTableIndexFromBinary(ctx, tableKey, indexBlockNum, rawIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal binary index: %w", err)
	}

	return index, nil
}

func (fdb *FluxDB) getIndex2(ctx context.Context, blockNum uint32, tablet Tablet) (index *TableIndex, err error) {
	indexableTablet, ok := tablet.(IndexableTablet)
	if !ok {
		return nil, fmt.Errorf("received tablet of type %t is not indexable", tablet)
	}

	ctx, span := dtracing.StartSpan(ctx, "get index")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("fetching table index from database", zap.Stringer("tablet", tablet), zap.Uint32("block_num", blockNum))

	tabletKey := string(tablet.Key())
	prefixKey := tabletKey + ":"
	startIndexKey := prefixKey + HexRevBlockNum(blockNum)

	zlog.Debug("reading table index row", zap.String("start_index_key", startIndexKey))

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

	index, err = NewTableIndexFromBinary2(ctx, indexableTablet, indexBlockNum, rawIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal binary index: %w", err)
	}

	return index, nil

	// var err2 error
	// err = fdb.tblIndex.ReadRows(ctx, bigtable.InfiniteRange(startIndexKey), func(row bigtable.Row) bool {
	// 	item, ok := btRowItem(row, indexFamilyName, indexColumnName)
	// 	if !ok {
	// 		err2 = fmt.Errorf("expected index family and column give no data: %q", item)
	// 		return false
	// 	}

	// 	if !strings.HasPrefix(item.Row, prefixKey) {
	// 		// Not found, or perhaps found ANOTHER ROW that isn't ours!
	// 		return false
	// 	}

	// 	blockNum, err := chunkKeyRevBlockNum(item.Row, prefixKey)
	// 	if err != nil {
	// 		err2 = fmt.Errorf("couldn't infer block num in table index's row key: %w", err)
	// 		return false
	// 	}

	// 	index, err = NewTableIndexFromBinary2(ctx, indexableTablet, blockNum, item.Value)
	// 	if err != nil {
	// 		err2 = fmt.Errorf("couldn't unmarshal binary index: %w", err)
	// 		return false
	// 	}

	// 	zlog.Debug("fetched table index", zap.Int("row_count", len(index.Map)))
	// 	return false
	// }, bigtable.LimitRows(1))

	// if err != nil {
	// 	return nil, err
	// }

	// if err2 != nil {
	// 	return nil, err2
	// }

	// The `index` variable can be null at this point, be warned!
	return index, nil
}

type indexCache struct {
	lastIndexes      map[string]*TableIndex
	lastCounters     map[string]int
	scheduleIndexing map[string]uint32
}

func newIndexCache() *indexCache {
	return &indexCache{
		lastIndexes:      make(map[string]*TableIndex),
		lastCounters:     make(map[string]int),
		scheduleIndexing: make(map[string]uint32),
	}
}

func (t *indexCache) GetIndex(table string) *TableIndex {
	return t.lastIndexes[table]
}

func (t *indexCache) CacheIndex(table string, tableIndex *TableIndex) {
	t.lastIndexes[table] = tableIndex
}

func (t *indexCache) GetCount(table string) int {
	return t.lastCounters[table]
}

func (t *indexCache) IncCount(table string) {
	t.lastCounters[table]++
}

func (t *indexCache) ResetCounter(table string) {
	t.lastCounters[table] = 0
}

// This algorithm determines the space between the indexes
func (t *indexCache) shouldTriggerIndexing(table string) bool {
	mutatedRowsCount := t.lastCounters[table]
	if mutatedRowsCount < 1000 {
		return false
	}

	lastIndex := t.lastIndexes[table]
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

func (t *indexCache) ScheduleIndex(table string, blockNum uint32) {
	t.scheduleIndexing[table] = blockNum
}

func (t *indexCache) IndexingSchedule() map[string]uint32 {
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

func NewTableIndexFromBinary2(ctx context.Context, tablet IndexableTablet, atBlockNum uint32, buffer []byte) (*TableIndex, error) {
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

		blockNumPtr := big.Uint32(buffer[pos+primaryKeyByteCount:])
		mapping[primaryKey] = blockNumPtr
	}

	return &TableIndex{
		AtBlockNum: atBlockNum,
		Squelched:  big.Uint32(buffer[:4]),
		Map:        mapping,
	}, nil
}

func NewTableIndexFromBinary(ctx context.Context, tableKey string, atBlockNum uint32, buffer []byte) (*TableIndex, error) {
	ctx, span := dtracing.StartSpan(ctx, "new table index from binary", "table_key", tableKey, "block_num", atBlockNum)
	defer span.End()

	primaryKeyByteCount := indexPrimaryKeyByteCountByTableKey(tableKey)
	if primaryKeyByteCount == 0 {
		return nil, fmt.Errorf("unknown primary key byte count for table key %q", tableKey)
	}

	// Byte count for primary key + 4 bytes for block num value
	entryByteCount := primaryKeyByteCount + 4

	// First 16 bytes are reserved to keep stats in there..
	byteCount := len(buffer)
	if (byteCount-16) < 0 || (byteCount-16)%entryByteCount != 0 {
		return nil, fmt.Errorf("unable to unmarshal table index: %d bytes alignment + 16 bytes metadata is off (has %d bytes)", entryByteCount, byteCount)
	}

	primaryKeyReader := indexPrimaryKeyReaderByTableKey(tableKey)
	if primaryKeyReader == nil {
		return nil, fmt.Errorf("unknown primary key writer for table key %q", tableKey)
	}

	mapping := map[string]uint32{}
	for pos := 16; pos < byteCount; pos += entryByteCount {
		primaryKey, err := primaryKeyReader(buffer[pos:])
		if err != nil {
			return nil, fmt.Errorf("unable to read primary key for table key %q: %w", tableKey, err)
		}

		blockNumPtr := big.Uint32(buffer[pos+primaryKeyByteCount:])
		mapping[primaryKey] = blockNumPtr
	}

	return &TableIndex{
		AtBlockNum: atBlockNum,
		Squelched:  big.Uint32(buffer[:4]),
		Map:        mapping,
	}, nil
}

func (index *TableIndex) MarshalBinary(ctx context.Context, tableKey string) ([]byte, error) {
	ctx, span := dtracing.StartSpan(ctx, "marshal table index to binary", "table_key", tableKey)
	defer span.End()

	primaryKeyByteCount := indexPrimaryKeyByteCountByTableKey(tableKey)
	if primaryKeyByteCount == 0 {
		return nil, fmt.Errorf("unknown primary key byte count for table key %q", tableKey)
	}

	primaryKeyWriter := indexPrimaryKeyWriterByTableKey(tableKey)
	if primaryKeyWriter == nil {
		return nil, fmt.Errorf("unknown primary key writer for table key %q", tableKey)
	}

	entryByteCount := primaryKeyByteCount + 4 // Byte count for primary key + 4 bytes for block num value

	snapshot := make([]byte, entryByteCount*len(index.Map)+16)
	big.PutUint32(snapshot, index.Squelched)

	pos := 16
	for primaryKey, blockNum := range index.Map {
		err := primaryKeyWriter(primaryKey, snapshot[pos:])
		if err != nil {
			return nil, fmt.Errorf("unable to read primary key for table key %q: %w", tableKey, err)
		}

		big.PutUint32(snapshot[pos+primaryKeyByteCount:], blockNum)
		pos += entryByteCount
	}

	return snapshot, nil
}

func (index *TableIndex) MarshalBinary2(ctx context.Context, tablet IndexableTablet) ([]byte, error) {
	ctx, span := dtracing.StartSpan(ctx, "marshal table index to binary", "tablet", tablet)
	defer span.End()

	primaryKeyByteCount := tablet.PrimaryKeyByteCount()
	entryByteCount := primaryKeyByteCount + 4 // Byte count for primary key + 4 bytes for block num value

	snapshot := make([]byte, entryByteCount*len(index.Map)+16)
	big.PutUint32(snapshot, index.Squelched)

	pos := 16
	for primaryKey, blockNum := range index.Map {
		err := tablet.EncodePrimaryKey(snapshot[pos:], primaryKey)
		if err != nil {
			return nil, fmt.Errorf("unable to read primary key for tablet %q: %w", tablet, err)
		}

		big.PutUint32(snapshot[pos+primaryKeyByteCount:], blockNum)
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

func indexPrimaryKeyByteCountByTableKey(tableKey string) int {
	switch {
	case strings.HasPrefix(tableKey, "al:"):
		return 16
	case strings.HasPrefix(tableKey, "arl:"):
		return 1
	// Block resource limit has no fields after prefix, so we must match without the :
	case strings.HasPrefix(tableKey, "brl"):
		return 1
	case strings.HasPrefix(tableKey, "ka2:"):
		return 16
	case strings.HasPrefix(tableKey, "td:"):
		return 8
	case strings.HasPrefix(tableKey, "ts:"):
		return 8
	default:
		return 0
	}
}

func indexPrimaryKeyReaderByTableKey(tableKey string) indexPrimaryKeyReader {
	switch {
	case strings.HasPrefix(tableKey, "al:"):
		return authLinkIndexPrimaryKeyReader
	case strings.HasPrefix(tableKey, "arl:"):
		return accountResourceLimitIndexPrimaryKeyReader
	// Block resource limit has no fields after prefix, so we must match without the :
	case strings.HasPrefix(tableKey, "brl"):
		return blockResourceLimitIndexPrimaryKeyReader
	case strings.HasPrefix(tableKey, "ka2:"):
		return keyAccountIndexPrimaryKeyReader
	case strings.HasPrefix(tableKey, "td:"):
		return tableDataIndexPrimaryKeyReader
	case strings.HasPrefix(tableKey, "ts:"):
		return tableScopeIndexPrimaryKeyReader
	default:
		return nil
	}
}

func indexPrimaryKeyWriterByTableKey(tableKey string) indexPrimaryKeyWriter {
	switch {
	case strings.HasPrefix(tableKey, "al:"):
		return authLinkIndexPrimaryKeyWriter
	case strings.HasPrefix(tableKey, "arl:"):
		return accountResourceLimitIndexPrimaryKeyWriter
	// Block resource limit has no fields after prefix, so we must match without the :
	case strings.HasPrefix(tableKey, "brl"):
		return blockResourceLimitIndexPrimaryKeyWriter
	case strings.HasPrefix(tableKey, "ka2:"):
		return keyAccountIndexPrimaryKeyWriter
	case strings.HasPrefix(tableKey, "td:"):
		return tableDataIndexPrimaryKeyWriter
	case strings.HasPrefix(tableKey, "ts:"):
		return tableScopeIndexPrimaryKeyWriter
	default:
		return nil
	}
}

var authLinkIndexPrimaryKeyReader = twoUint64PrimaryKeyReaderFactory("auth link")
var accountResourceLimitIndexPrimaryKeyReader = oneBytePrimaryKeyReaderFactory("account resource limit")
var blockResourceLimitIndexPrimaryKeyReader = oneBytePrimaryKeyReaderFactory("block resource limit")
var keyAccountIndexPrimaryKeyReader = twoUint64PrimaryKeyReaderFactory("key account")
var tableDataIndexPrimaryKeyReader = oneUint64PrimaryKeyReaderFactory("table data")
var tableScopeIndexPrimaryKeyReader = oneUint64PrimaryKeyReaderFactory("table scope")

func oneBytePrimaryKeyReaderFactory(tag string) indexPrimaryKeyReader {
	return func(buffer []byte) (string, error) {
		if len(buffer) < 1 {
			return "", fmt.Errorf("%s primary key reader: not enough bytes to read, %d bytes left, wants %d", tag, len(buffer), 1)
		}

		return fmt.Sprintf("%02x", buffer[0]), nil
	}
}

func oneUint64PrimaryKeyReaderFactory(tag string) indexPrimaryKeyReader {
	return func(buffer []byte) (string, error) {
		primaryKey, err := readOneUint64(buffer)
		if err != nil {
			return "", fmt.Errorf("%s primary key reader: %w", tag, err)
		}

		return primaryKey, nil
	}
}

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

	return fmt.Sprintf("%016x", big.Uint64(buffer)), nil
}

var authLinkIndexPrimaryKeyWriter = twoUint64PrimaryKeyWriterFactory("auth link")
var accountResourceLimitIndexPrimaryKeyWriter = oneBytePrimaryKeyWriterFactory("account resource limit")
var blockResourceLimitIndexPrimaryKeyWriter = oneBytePrimaryKeyWriterFactory("block resource limit")
var keyAccountIndexPrimaryKeyWriter = twoUint64PrimaryKeyWriterFactory("key account")
var tableDataIndexPrimaryKeyWriter = oneUint64PrimaryKeyWriterFactory("table data")
var tableScopeIndexPrimaryKeyWriter = oneUint64PrimaryKeyWriterFactory("table scope")

func oneBytePrimaryKeyWriterFactory(tag string) indexPrimaryKeyWriter {
	return func(primaryKey string, buffer []byte) error {
		value, err := strconv.ParseUint(primaryKey, 16, 8)
		if err != nil {
			return fmt.Errorf("%s primary key writer: unable to transform primary key to byte: %w", tag, err)
		}

		buffer[0] = byte(value)
		return nil
	}
}

func oneUint64PrimaryKeyWriterFactory(tag string) indexPrimaryKeyWriter {
	return func(primaryKey string, buffer []byte) error {
		err := writeOneUint64(primaryKey, buffer)
		if err != nil {
			return fmt.Errorf("%s primary key writer: %w", tag, err)
		}

		return nil
	}
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

	big.PutUint64(buffer, value)
	return nil
}
