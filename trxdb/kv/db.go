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

package kv

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

// "read"'s default is "*"
// "write"'s default is "*"
// Dsn examples
// dgraphql in eos mainnet core: store:///        				 by default: read=*&write=*
// dgraphql in curv: store://mainnet/?read=blk,trx&write=none store://curv/?read=trx&write=none
// dgraphql in curv: store://mainnet/?read=blk&write=none store://curv/?read=trx&write=none
// trxdb-loader for mainnet core: store:///?write=blk 		/* read="*" */
// trxdb-loader for curv: store:///?write=trx          		 /* only purpose: NOT WRITE blk */
// single laptop-style deployment:           store:///             by default: read=*&write=*
// single laptop-style deployment, secure:   store:///?read=blk,trx&write=none
type DB struct {
	blkReadStore store.KVStore
	trxReadStore store.KVStore
	irrReadStore store.KVStore

	lastWrittenBlockStore store.KVStore
	enableBlkWrite        bool
	enableTrxWrite        bool
	writeStore            store.KVStore

	// Required only when writing
	writerChainID []byte

	enc *trxdb.ProtoEncoder
	dec *trxdb.ProtoDecoder

	purgeInterval uint64
	logger        *zap.Logger
}

func init() {
	testFactory := func(dsns []string) (trxdb.DB, error) {
		return New(dsns)
	}

	trxdb.Register("badger", testFactory)
	trxdb.Register("tikv", testFactory)
	trxdb.Register("bigkv", testFactory)
	trxdb.Register("netkv", testFactory)
	trxdb.Register("cznickv", testFactory)
}

type dsnOptions struct {
	reads  []string
	writes []string
}

func New(dsns []string) (*DB, error) {
	zlog.Debug("creating kv db", zap.Strings("dsns", dsns))
	db := &DB{
		enc:    trxdb.NewProtoEncoder(),
		dec:    trxdb.NewProtoDecoder(),
		logger: zap.NewNop(),
	}

	hasSeenWriter := false
	for _, dsn := range dsns {
		cleanDsn, dsnOptions, err := parseAndCleanDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse and clean kv driver dsn: %w", err)
		}

		// Currently we only support 1 writer DSN. This assumption is baked into the code
		// so change this carefully.
		isWriter := isWriter(dsnOptions.writes)
		if isWriter && hasSeenWriter {
			return nil, fmt.Errorf("unable to have 2 writer DSNs")
		}
		hasSeenWriter = isWriter

		driver, err := newCachedKVDB(cleanDsn)
		if err != nil {
			return nil, fmt.Errorf("unable retrieve kvdb driver: %w", err)
		}

		db.setupReadWriteOpts(driver, dsnOptions.reads, dsnOptions.writes)
	}
	return db, nil

}

type storeIterFunc func(s store.KVStore) error

func (db *DB) itrAllStores(f storeIterFunc) error {
	return db.itrStores(f, []store.KVStore{db.writeStore, db.blkReadStore, db.trxReadStore})
}

func (db *DB) itrStores(f storeIterFunc, stores []store.KVStore) error {
	for _, s := range stores {
		if s != nil {
			err := f(s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *DB) Close() error {
	return db.itrAllStores(func(s store.KVStore) error {
		return s.Close()
	})
}

// setupReadWriteOpts will set the driver for the database, the trx reader store
// takes precedent over the blk reader store
func (db *DB) setupReadWriteOpts(driver store.KVStore, read, write []string) {
	if inSlice("blk", read) {
		db.blkReadStore = driver
		// the trx reader DSN has precedence
		if db.irrReadStore == nil {
			db.irrReadStore = driver
		}
	}

	if inSlice("trx", read) {
		db.trxReadStore = driver
		// trx WINS and overrides
		db.irrReadStore = driver
	}

	if inSlice("all", read) {
		db.trxReadStore = driver
		db.blkReadStore = driver

		// the Trx reader DSN has precedence
		if db.irrReadStore == nil {
			db.irrReadStore = driver
		}
	}

	if inSlice("last_written_blk", read) {
		db.lastWrittenBlockStore = driver
	}

	if isWriter(write) {
		db.writeStore = driver
	}

	if inSlice("blk", write) {
		db.enableBlkWrite = true
	}

	if inSlice("trx", write) {
		db.enableTrxWrite = true
	}

	if inSlice("all", write) {
		db.enableTrxWrite = true
		db.enableBlkWrite = true
	}
	// write == "none" sets nothing
}

func inSlice(value string, slice []string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

func isWriter(writes []string) bool {
	for _, s := range writes {
		if s != "none" {
			return true
		}
	}
	return false
}

//* using for debugging *//

func (db *DB) Dump() {

	fields := []zap.Field{}
	if s, ok := db.writeStore.(fmt.Stringer); ok {
		fields = append(fields, zap.String("write_store", s.String()))
	}

	if s, ok := db.blkReadStore.(fmt.Stringer); ok {
		fields = append(fields, zap.String("block_read_store", s.String()))
	}

	if s, ok := db.trxReadStore.(fmt.Stringer); ok {
		fields = append(fields, zap.String("trx_read_store", s.String()))
	}

	fields = append(fields,
		zap.Bool("blk_write_enabled", db.enableBlkWrite),
		zap.Bool("trx_write_enabled", db.enableTrxWrite),
		zap.Bool("blk_read_store_enabled", db.blkReadStore != nil),
		zap.Bool("trx_read_store_enabled", db.trxReadStore != nil),
	)

	db.logger.Info("trxdb driver dump", fields...)
}
