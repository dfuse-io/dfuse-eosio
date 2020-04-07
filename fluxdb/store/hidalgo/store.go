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

package hidalgo

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/coreos/bbolt"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/hidal-go/hidalgo/kv"
	kvbbolt "github.com/hidal-go/hidalgo/kv/bbolt"
	"go.uber.org/zap"
)

type KVStore struct {
	db kv.KV

	tblRows  string
	tblIndex string
	tblABIs  string
	tblLast  string
}

type KVStoreDsn struct {
	path        string
	createTable bool
}

func NewKVStore(ctx context.Context, dsnString string) (*KVStore, error) {
	// Hidalgo is really poor (inexistant) on examples, so it's hard to see what's the
	// correct way of creating the `kv.KV` instance. We will assume that's the right way for now.
	registration := kv.ByName("bbolt")
	if registration == nil {
		return nil, fmt.Errorf("no bbolt registration found, does the register package '_ github.com/hidal-go/hidalgo/kv/bbolt' still present")
	}

	kvdns, err := parseDNS(dsnString)
	if err != nil {
		return nil, err
	}

	// It's also possible to instantiate from a `bbolt` instance directly, if using `bbolt.New` (`hidalgo/kv/bbolt` package)
	zlog.Info("opening hidalgo database file", zap.String("dsn", dsnString), zap.String("path", kvdns.path), zap.Bool("create_tables", kvdns.createTable))
	db, err := registration.OpenPath(kvdns.path)
	if err != nil {
		return nil, fmt.Errorf("unable to open bbolt db %q: %w", kvdns.path, err)
	}

	store := &KVStore{
		db:       db,
		tblRows:  "tablet",
		tblIndex: "index",
		tblABIs:  "abi",
		tblLast:  "block",
	}

	if kvdns.createTable {
		tables := []string{store.tblRows, store.tblIndex, store.tblABIs, store.tblLast}
		zlog.Info("creating buckets", zap.Strings("tables", tables))
		for _, table := range tables {
			err := createBucket(ctx, db, table)
			if err != nil {
				return nil, err
			}
		}
	}

	return store, nil
}

func (s *KVStore) Close() error {
	return s.db.Close()
}

func (s *KVStore) NewBatch(logger *zap.Logger) store.Batch {
	return newBatch(s, logger)
}

func (s *KVStore) FetchABI(ctx context.Context, prefixKey, keyStart, keyEnd string) (rowKey string, rawABI []byte, err error) {
	err = s.scanRange(ctx, s.tblABIs, keyStart, keyEnd, func(key string, value []byte) error {
		if !strings.HasPrefix(key, prefixKey) {
			return store.BreakScan
		}

		rowKey = key
		rawABI = value

		// We only ever check a single row
		return store.BreakScan
	})

	if err != nil && err != store.BreakScan {
		return "", nil, fmt.Errorf("unable to fetch ABI for key prefix %q: %w", prefixKey, err)
	}

	if rawABI == nil {
		return "", nil, store.ErrNotFound
	}

	return rowKey, rawABI, nil
}

func (s *KVStore) FetchIndex(ctx context.Context, tableKey, prefixKey, keyStart string) (rowKey string, rawIndex []byte, err error) {
	err = s.scanInfiniteRange(ctx, s.tblIndex, keyStart, func(key string, value []byte) error {
		if !strings.HasPrefix(key, prefixKey) {
			return store.BreakScan
		}

		rowKey = key
		rawIndex = value

		// We always only check a single row
		return store.BreakScan
	})

	if err != nil && err != store.BreakScan {
		return "", nil, fmt.Errorf("unable to fetch index for key prefix %q: %w", prefixKey, err)
	}

	if rawIndex == nil {
		return "", nil, store.ErrNotFound
	}

	return rowKey, rawIndex, nil
}

func (s *KVStore) HasTabletRow(ctx context.Context, keyPrefix string) (exists bool, err error) {
	err = s.scanPrefix(ctx, s.tblRows, keyPrefix, func(key string, _ []byte) error {
		exists = true
		return store.BreakScan
	})

	if err != nil && err != store.BreakScan {
		return false, fmt.Errorf("unable to determine if table %q has key prefix %q: %w", s.tblRows, keyPrefix, err)
	}

	return exists, nil
}

func (s *KVStore) FetchTabletRow(ctx context.Context, key string, onTabletRow store.OnTabletRow) error {
	value, err := s.fetchKey(ctx, s.tblRows, key)
	if err != nil {
		return err
	}

	err = onTabletRow(key, value)
	if err != nil && err != store.BreakScan {
		return fmt.Errorf("on tablet row for key %q failed: %w", key, err)
	}

	return nil
}

func (s *KVStore) FetchTabletRows(ctx context.Context, keys []string, onTabletRow store.OnTabletRow) error {
	values, err := s.fetchKeys(ctx, s.tblRows, keys)
	if err != nil {
		return err
	}

	for i, value := range values {
		// We must be prudent here, a `nil` value indicate a key not found, a `[]byte{}` indicates a found key without a value!
		if value == nil {
			continue
		}

		err = onTabletRow(keys[i], value)
		if err == store.BreakScan {
			return nil
		}

		if err != nil {
			return fmt.Errorf("on tablet row for key %q failed: %w", keys[i], err)
		}
	}

	return nil
}

func (s *KVStore) ScanTabletRows(ctx context.Context, keyStart, keyEnd string, onTabletRow store.OnTabletRow) error {
	err := s.scanRange(ctx, s.tblRows, keyStart, keyEnd, func(key string, value []byte) error {
		err := onTabletRow(key, value)
		if err == store.BreakScan {
			return store.BreakScan
		}

		if err != nil {
			return fmt.Errorf("on tablet row for key %q failed: %w", key, err)
		}

		return nil
	})

	if err != nil && err != store.BreakScan {
		return fmt.Errorf("unable to scan tablet rows [%q, %q[: %w", keyStart, keyEnd, err)
	}

	return nil
}

func (s *KVStore) FetchLastWrittenBlock(ctx context.Context, key string) (out bstream.BlockRef, err error) {
	value, err := s.fetchKey(ctx, s.tblLast, key)
	if err != nil {
		return nil, err
	}

	return bstream.BlockRefFromID(string(value)), nil
}

func (s *KVStore) ScanLastShardsWrittenBlock(ctx context.Context, keyPrefix string, onBlockRef store.OnBlockRef) error {
	err := s.scanPrefix(ctx, s.tblLast, keyPrefix, func(key string, value []byte) error {
		err := onBlockRef(key, bstream.BlockRefFromID(value))
		if err == store.BreakScan {
			return store.BreakScan
		}

		if err != nil {
			return fmt.Errorf("on block ref for table %q key %q failed: %w", s.tblRows, key, err)
		}

		return nil
	})

	if err != nil && err != store.BreakScan {
		return fmt.Errorf("unable to determine if table %q has key prefix %q: %w", s.tblLast, keyPrefix, err)
	}

	return nil
}

func (s *KVStore) fetchKey(ctx context.Context, table, key string) (out []byte, err error) {
	err = kv.View(s.db, func(tx kv.Tx) error {
		out, err = tx.Get(ctx, kv.SKey(table, key))
		return err
	})

	if err == kv.ErrNotFound {
		return nil, store.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("unable to fetch table %q key %q: %w", table, key, err)
	}

	return out, nil
}

func (s *KVStore) fetchKeys(ctx context.Context, table string, keys []string) (out [][]byte, err error) {
	kvKeys := make([]kv.Key, len(keys))
	for i, key := range keys {
		kvKeys[i] = kv.SKey(table, key)
	}

	var values []kv.Value
	err = kv.View(s.db, func(tx kv.Tx) error {
		values, err = tx.GetBatch(ctx, kvKeys)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("unable to fetch table %q keys (%d): %w", table, len(keys), err)
	}

	out = make([][]byte, len(values))
	for i, value := range values {
		out[i] = []byte(value)
	}

	return out, nil
}

func (s *KVStore) scanPrefix(ctx context.Context, table, prefixKey string, onRow func(key string, value []byte) error) error {
	err := kv.View(s.db, func(tx kv.Tx) error {
		return kv.Each(ctx, tx, kv.SKey(table, prefixKey), func(kvKey kv.Key, value kv.Value) error {
			_, key := keyToString(kvKey)

			return onRow(key, []byte(value))
		})
	})

	if err != nil && err != store.BreakScan {
		return fmt.Errorf("unable to scan table %q keys with prefix %q: %w", table, prefixKey, err)
	}

	return nil
}

func (s *KVStore) scanRange(ctx context.Context, table, keyStart, keyEnd string, onRow func(key string, value []byte) error) error {
	// FIXME: It appears there is no way to perform a scan range using hidalgo abstraction.
	//        For now relying on direct implementation, since we only support bboltdb, see
	//        issue I logged for more details: https://github.com/hidal-go/hidalgo/issues/12
	boltdb := s.db.(*kvbbolt.DB).DB()

	min := []byte(keyStart)
	max := []byte(keyEnd)
	openEnded := keyEnd == ""

	zlog.Info("scanning range", zap.String("start", keyStart), zap.String("end", keyEnd))

	// FIXME: Should we return ErrNotFound if not key were iterated?
	err := boltdb.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte(table)).Cursor()

		zlog.Info("scanning range cursor")
		for k, v := c.Seek(min); k != nil && (openEnded || bytes.Compare(k, max) <= 0); k, v = c.Next() {
			zlog.Info("got key scanned", zap.String("key", string(k)))
			err := onRow(string(k), v)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (s *KVStore) scanInfiniteRange(ctx context.Context, table, keyStart string, onRow func(key string, value []byte) error) error {
	return s.scanRange(ctx, table, keyStart, "", onRow)
}

func keyToString(kvKey kv.Key) (table string, key string) {
	if len(kvKey) <= 0 {
		return
	}

	// All our hidalgo key starts withthe bucket (i.e. table), hence the removal of initial segment
	table = string(kvKey[0])
	if len(kvKey) <= 1 {
		return
	}

	key = joinBytes([][]byte(kvKey[1:]), "")
	return
}

// Copied from strings.Join
func joinBytes(a [][]byte, sep string) string {
	switch len(a) {
	case 0:
		return ""
	case 1:
		return string(a[0])
	}
	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(string(a[i]))
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(string(a[0]))
	for _, s := range a[1:] {
		b.WriteString(sep)
		b.WriteString(string(s))
	}
	return b.String()
}

// There is most probably lots of repetition between this batch and the bigtable version.
// We should most probably improve the sharing by having a `baseBatch` struct or something
// like that.
type batch struct {
	store          *KVStore
	count          int
	tableMutations map[string]map[string][]byte

	zlog *zap.Logger
}

func newBatch(store *KVStore, logger *zap.Logger) *batch {
	batchSet := &batch{store: store, zlog: logger}
	batchSet.Reset()

	return batchSet
}

func (b *batch) Reset() {
	b.count = 0
	b.tableMutations = map[string]map[string][]byte{
		b.store.tblABIs:  make(map[string][]byte),
		b.store.tblRows:  make(map[string][]byte),
		b.store.tblIndex: make(map[string][]byte),
		b.store.tblLast:  make(map[string][]byte),
	}
}

// For now, if flush each time we have 100 pending mutations in total, would need to be
// adjusted and to check if we would be able to improve throughput by using "batch" mode
// of bbolt (hopefully, exposed correctly in Hidalgo).
var maxMutationCount = 100

func (b *batch) FlushIfFull(ctx context.Context) error {
	if b.count <= maxMutationCount {
		// We are not there yet
		return nil
	}

	b.zlog.Info("flushing a full batch set", zap.Int("count", b.count))
	if err := b.Flush(ctx); err != nil {
		return derr.Wrap(err, "flushing batch set")
	}

	return nil
}

func (b *batch) Flush(ctx context.Context) error {
	ctx, span := dtracing.StartSpan(ctx, "flush batch set")
	defer span.End()

	b.zlog.Info("flushing batch set")

	tableNames := []string{
		b.store.tblABIs,
		b.store.tblRows,
		b.store.tblIndex,

		// The table name `last` must always be the last table in this list!
		b.store.tblLast,
	}

	// TODO: We could eventually parallelize this, but remember, last would need to be processed last, after all others!
	for _, tblName := range tableNames {
		muts := b.tableMutations[tblName]

		if len(muts) <= 0 {
			continue
		}

		b.zlog.Info("applying bulk update", zap.String("table_name", tblName), zap.Int("mutation_count", len(muts)))
		ctx, span := dtracing.StartSpan(ctx, "apply bulk updates", "table", tblName, "mutation_count", len(muts))

		err := kv.Update(ctx, b.store.db, func(tx kv.Tx) error {
			for key, value := range muts {
				err := tx.Put(kv.SKey(tblName, key), value)
				if err != nil {
					return fmt.Errorf("unable to add table %q key %q to tx: %w", tblName, key, err)
				}
			}

			return nil
		})
		span.End()

		if err != nil {
			return derr.Wrap(err, "apply bulk")
		}
	}

	b.Reset()

	return nil
}

func (b *batch) setTable(table, key string, value []byte) {
	b.tableMutations[table][key] = value
	b.count++
}

func (b *batch) SetABI(key string, value []byte) {
	b.setTable(b.store.tblABIs, key, value)
}

func (b *batch) SetRow(key string, value []byte) {
	b.setTable(b.store.tblRows, key, value)
}

func (b *batch) SetLast(key string, value []byte) {
	b.setTable(b.store.tblLast, key, value)
}

func (b *batch) SetIndex(key string, tableSnapshot []byte) {
	b.setTable(b.store.tblIndex, key, tableSnapshot)
}

func createBucket(ctx context.Context, db kv.KV, table string) error {
	err := kv.Update(ctx, db, func(tx kv.Tx) error {
		return kv.CreateBucket(ctx, tx, kv.SKey(table))
	})

	if err != nil {
		return fmt.Errorf("unable to create bucket %q: %w", table, err)
	}

	return nil
}

func parseDNS(dns string) (*KVStoreDsn, error) {
	u, err := url.Parse(dns)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	if u.Hostname() != "" {
		paths = append(paths, u.Hostname())
	}

	if u.Path != "" {
		paths = append(paths, u.Path)
	}

	path := strings.Join(paths, "/")
	return &KVStoreDsn{
		path:        path,
		createTable: u.Query().Get("createTables") == "true",
	}, nil
}
