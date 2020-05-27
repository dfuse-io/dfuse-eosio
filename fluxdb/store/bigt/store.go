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

package bigt

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigtable"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

type KVStore struct {
	client      *bigtable.Client
	tablePrefix string

	tblRows  *bigtable.Table
	tblIndex *bigtable.Table
	tblABIs  *bigtable.Table
	tblLast  *bigtable.Table
}

func NewKVStore(ctx context.Context, dsnString string, opts ...option.ClientOption) (*KVStore, error) {
	dsn, err := basebigt.ParseDSN(dsnString)
	if err != nil {
		return nil, err
	}

	optionalTestEnv(dsn.Project, dsn.Instance)

	client, err := bigtable.NewClient(ctx, dsn.Project, dsn.Instance, opts...)
	if err != nil {
		return nil, err
	}

	fdb := &KVStore{
		client:      client,
		tablePrefix: dsn.TablePrefix,
	}

	tblRows := fmt.Sprintf("flux-%s-rows", dsn.TablePrefix)
	fdb.tblRows = client.Open(tblRows)
	tblIndex := fmt.Sprintf("flux-%s-idxs", dsn.TablePrefix)
	fdb.tblIndex = client.Open(tblIndex)
	tblABIs := fmt.Sprintf("flux-%s-abis", dsn.TablePrefix)
	fdb.tblABIs = client.Open(tblABIs)
	tblLast := fmt.Sprintf("flux-%s-last", dsn.TablePrefix)
	fdb.tblLast = client.Open(tblLast)

	if dsn.CreateTables {
		adminClient, err := bigtable.NewAdminClient(ctx, dsn.Project, dsn.Instance, opts...)
		if err != nil {
			zlog.Warn("couldn't do admin tasks", zap.Error(err))
		} else {
			zlog.Info("creating tables", zap.Strings("tables", []string{tblRows, tblIndex, tblLast, tblABIs}))
			createTable(ctx, adminClient, tblRows, rowFamilyName)
			createTable(ctx, adminClient, tblIndex, indexFamilyName)
			createTable(ctx, adminClient, tblLast, lastBlockFamilyName)
			createTable(ctx, adminClient, tblABIs, abiFamilyName)
		}
	}

	return fdb, nil
}

func (s *KVStore) Close() error {
	return s.client.Close()
}

func (s *KVStore) NewBatch(logger *zap.Logger) store.Batch {
	return newbatch(s, logger)
}

func (s *KVStore) FetchABI(ctx context.Context, prefixKey, keyStart, keyEnd string) (rowKey string, rawABI []byte, err error) {
	var err2 error
	err = s.tblABIs.ReadRows(ctx, bigtable.NewRange(keyStart, keyEnd), func(row bigtable.Row) bool {
		item, ok := btRowItem(row, abiFamilyName, abiColumnName)
		if !ok {
			err2 = fmt.Errorf("expected abi family and column give no data: %q", item)
			return false
		}

		if !strings.HasPrefix(item.Row, prefixKey) {
			err2 = store.ErrNotFound
			return false
		}

		rowKey = row.Key()
		rawABI = item.Value

		return false
	}, bigtable.LimitRows(1))

	if err != nil {
		return "", nil, err
	}

	if err2 != nil {
		return "", nil, err2
	}

	if rawABI == nil {
		return "", nil, store.ErrNotFound
	}

	return rowKey, rawABI, nil
}

func (s *KVStore) FetchIndex(ctx context.Context, tableKey, prefixKey, keyStart string) (rowKey string, rawIndex []byte, err error) {
	var err2 error
	err = s.tblIndex.ReadRows(ctx, bigtable.InfiniteRange(keyStart), func(row bigtable.Row) bool {
		item, ok := btRowItem(row, indexFamilyName, indexColumnName)
		if !ok {
			err2 = fmt.Errorf("expected index family and column give no data: %q", item)
			return false
		}

		if !strings.HasPrefix(item.Row, prefixKey) {
			err2 = store.ErrNotFound
			return false
		}

		rowKey = row.Key()
		rawIndex = item.Value

		return false
	}, bigtable.LimitRows(1))

	if err != nil {
		return "", nil, err
	}

	if err2 != nil {
		return "", nil, err2
	}

	if rawIndex == nil {
		return "", nil, store.ErrNotFound
	}

	return rowKey, rawIndex, nil
}

func (s *KVStore) HasTabletRow(ctx context.Context, keyPrefix string) (exists bool, err error) {
	filters := bigtable.ChainFilters(bigtable.CellsPerRowLimitFilter(1), bigtable.LatestNFilter(1), bigtable.StripValueFilter())
	err = s.tblRows.ReadRows(ctx, bigtable.PrefixRange(keyPrefix), func(row bigtable.Row) bool {
		if strings.HasPrefix(row.Key(), keyPrefix) {
			exists = true
		}

		return false
	}, bigtable.LimitRows(1), bigtable.RowFilter(filters))

	if err != nil {
		return false, derr.Wrapf(err, "unable to read rows for row prefix key %s", keyPrefix)
	}

	return exists, nil
}

func (s *KVStore) FetchTabletRow(ctx context.Context, key string, onTabletRow store.OnTabletRow) error {
	row, err := s.tblRows.ReadRow(ctx, key)
	if err != nil {
		return derr.Wrapf(err, "read tablet row %q", key)
	}

	item, ok := btRowItem(row, rowFamilyName, rowColumnName)
	if !ok {
		return store.ErrNotFound
	}

	err = onTabletRow(row.Key(), item.Value)
	if err != nil && err != store.BreakScan {
		return fmt.Errorf("on tablet row for key %q failed: %w", row.Key(), err)
	}

	return nil
}

func (s *KVStore) FetchTabletRows(ctx context.Context, keys []string, onTabletRow store.OnTabletRow) error {
	var err2 error
	err := s.tblRows.ReadRows(ctx, bigtable.RowList(keys), func(row bigtable.Row) bool {
		item, ok := btRowItem(row, rowFamilyName, rowColumnName)
		if !ok {
			err2 = fmt.Errorf("expected tablet row family and column give no data: %q", item)
			return false
		}

		err2 = onTabletRow(row.Key(), item.Value)
		if err2 == store.BreakScan {
			return false
		}

		if err2 != nil {
			err2 = fmt.Errorf("on tablet row for key %q failed: %w", row.Key(), err2)
			return false
		}

		return true
	})

	if err2 != nil {
		return err2
	}

	return err
}

func (s *KVStore) ScanTabletRows(ctx context.Context, keyStart, keyEnd string, onTabletRow store.OnTabletRow) error {
	var err2 error
	err := s.tblRows.ReadRows(ctx, bigtable.NewRange(keyStart, keyEnd), func(row bigtable.Row) bool {
		item, ok := btRowItem(row, rowFamilyName, rowColumnName)
		if !ok {
			err2 = fmt.Errorf("expected tablet row family and column give no data: %q", item)
			return false
		}

		err2 = onTabletRow(row.Key(), item.Value)
		if err2 == store.BreakScan {
			return false
		}

		if err2 != nil {
			err2 = fmt.Errorf("on tablet row for key %q failed: %w", row.Key(), err2)
			return false
		}

		return true
	})

	if err2 != nil {
		return err2
	}

	return err
}

func (s *KVStore) FetchLastWrittenBlock(ctx context.Context, key string) (bstream.BlockRef, error) {
	row, err := s.tblLast.ReadRow(ctx, key, latestCellFilter)
	if err != nil {
		return nil, err
	}

	if row[lastBlockFamilyName] == nil || len(row[lastBlockFamilyName]) == 0 {
		return nil, store.ErrNotFound
	}

	return bstream.BlockRefFromID(string(row[lastBlockFamilyName][0].Value)), nil
}

func (s *KVStore) ScanLastShardsWrittenBlock(ctx context.Context, keyPrefix string, onBlockRef store.OnBlockRef) error {
	var err2 error
	err := s.tblLast.ReadRows(ctx, bigtable.PrefixRange(keyPrefix), func(row bigtable.Row) bool {
		if row[lastBlockFamilyName] == nil || len(row[lastBlockFamilyName]) == 0 {
			err2 = fmt.Errorf("couldn't find %q:%q for key %q", lastBlockFamilyName, lastBlockColumnName, row.Key())
			return false
		}

		err2 = onBlockRef(row.Key(), bstream.BlockRefFromID(row[lastBlockFamilyName][0].Value))
		if err2 == store.BreakScan {
			return false
		}

		if err2 != nil {
			err2 = fmt.Errorf("on block ref for key %q failed: %w", row.Key(), err2)
			return false
		}

		return true
	}, latestCellFilter)

	if err2 != nil {
		return err2
	}

	return err
}

type batch struct {
	store          *KVStore
	size           int
	tableMutations map[string]map[string]*bigtable.Mutation

	zlog *zap.Logger
}

func newbatch(store *KVStore, logger *zap.Logger) *batch {
	batchSet := &batch{store: store, zlog: logger}
	batchSet.Reset()

	return batchSet
}

func (b *batch) Reset() {
	b.size = 0
	b.tableMutations = map[string]map[string]*bigtable.Mutation{
		"abi":   make(map[string]*bigtable.Mutation),
		"row":   make(map[string]*bigtable.Mutation),
		"index": make(map[string]*bigtable.Mutation),
		"last":  make(map[string]*bigtable.Mutation),
	}
}

// Limit from Google Bigtable is 256MB, the emulator (USE A RECENT
// ONE) has 256MB.  This is needed to run through the emulator. Otherwise, run:
//     go get github.com/googleapis/google-cloud-go/bigtable/cmd/emulator
// and run that one.
var maxBatchSize = 75000000

func (b *batch) FlushIfFull(ctx context.Context) error {
	if b.size <= maxBatchSize {
		// We are not there yet, size is below 75 MB
		return nil
	}

	// Size greater than 75 MB (actual real limit near 100 MB)
	b.zlog.Info("flushing a full batch set", zap.Int("size", b.size))
	if err := b.Flush(ctx); err != nil {
		return derr.Wrap(err, "flushing batch set")
	}

	return nil
}

func (b *batch) Flush(ctx context.Context) error {
	ctx, span := dtracing.StartSpan(ctx, "flush batch set")
	defer span.End()

	b.zlog.Debug("flushing batch set")

	tableNames := []string{
		"abi",
		"row",
		"index",

		// The table name `last` must always be the last table in this list!
		"last",
	}

	// TODO: We could eventually parallelize this, but remember, last would need to be processed last, after all others!
	for _, tblName := range tableNames {
		muts := b.tableMutations[tblName]

		var tbl *bigtable.Table
		switch tblName {
		case "abi":
			tbl = b.store.tblABIs
		case "row":
			tbl = b.store.tblRows
		case "index":
			tbl = b.store.tblIndex
		case "last":
			tbl = b.store.tblLast
		default:
			panic("unknown table: " + tblName)
		}

		if len(muts) <= 0 {
			continue
		}

		var keys []string
		var vals []*bigtable.Mutation
		for k, v := range muts {
			keys = append(keys, k)
			vals = append(vals, v)
		}

		b.zlog.Debug("applying bulk update", zap.String("table_name", tblName), zap.Int("mutation_count", len(muts)), zap.Int("size", b.size))
		ctx, span := dtracing.StartSpan(ctx, "apply bulk updates", "table", tblName, "mutation_count", len(muts))

		errors, err := tbl.ApplyBulk(ctx, keys, vals)
		if err != nil {
			span.End()
			return derr.Wrap(err, "apply bulk")
		}

		if len(errors) != 0 {
			span.End()
			return fmt.Errorf("some errors writing to bigtable: %s", errors)
		}

		span.End()
	}

	b.Reset()

	return nil
}

func (b *batch) setTable(table string, key, family, column string, value []byte) {
	mut := bigtable.NewMutation()
	mut.Set(family, column, bigtable.Now(), value)
	b.tableMutations[table][key] = mut
	b.size += len(value) + 100 /* 100 = overhead */
}

func (b *batch) SetABI(key string, value []byte) {
	b.setTable("abi", key, abiFamilyName, abiColumnName, value)
}

func (b *batch) SetRow(key string, value []byte) {
	b.setTable("row", key, rowFamilyName, rowColumnName, value)
}

func (b *batch) SetLast(key string, value []byte) {
	b.setTable("last", key, lastBlockFamilyName, lastBlockColumnName, value)
}

func (b *batch) SetIndex(key string, tableSnapshot []byte) {
	b.setTable("index", key, indexFamilyName, indexColumnName, tableSnapshot)
}

func createTable(ctx context.Context, admin *bigtable.AdminClient, tableName, familyName string) {
	if err := admin.CreateTable(ctx, tableName); err != nil {
		zlog.Warn("failed creating table", zap.String("table_name", tableName), zap.Error(err))
	}

	if err := admin.CreateColumnFamily(ctx, tableName, familyName); err != nil {
		zlog.Warn("failed creating table family",
			zap.String("table_name", tableName),
			zap.String("family_name", familyName),
			zap.Error(err),
		)
	}

	if err := admin.SetGCPolicy(ctx, tableName, familyName, bigtable.MaxVersionsPolicy(1)); err != nil {
		zlog.Warn("failed applying gc policy to table",
			zap.String("table_name", tableName),
			zap.String("family_name", familyName),
			zap.Error(err),
		)
	}
}
