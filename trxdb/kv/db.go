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
	blkWriteStore store.KVStore
	trxWriteStore store.KVStore

	blksReadStore store.KVStore
	trxReadStore  store.KVStore

	irrBlockStore store.KVStore

	// Required only when writing
	writerChainID []byte

	enc *trxdb.ProtoEncoder
	dec *trxdb.ProtoDecoder

	purgeInterval uint64
	logger        *zap.Logger
}

func init() {
	trxdb.Register("badger", New)
	trxdb.Register("tikv", New)
	trxdb.Register("bigkv", New)
	trxdb.Register("cznickv", New)
}

func New(dsns []string) (trxdb.DB, error) {

	db := &DB{
		enc:    trxdb.NewProtoEncoder(),
		dec:    trxdb.NewProtoDecoder(),
		logger: zap.NewNop(),
	}

	hasSeenWriter := false
	for _, dsn := range dsns {
		cleanDsn, reads, writes, err := parseAndCleanDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse and clean kv driver dsn: %w", err)
		}

		isWriter := isWriter(writes)
		if isWriter && hasSeenWriter {
			return nil, fmt.Errorf("unable to have 2 writer DSNs")
		}
		hasSeenWriter = isWriter

		driver, err := newCachedKVDB(cleanDsn)
		if err != nil {
			return nil, fmt.Errorf("unable retrieve kvdb driver: %w", err)
		}

		///

		db.setupReadWriteOpts(driver, reads, writes)
	}
	return db, nil

}

type storeIterFunc func(s store.KVStore) error

func (db *DB) itrWritableStores(f storeIterFunc) error {
	return db.itrStores(f, []store.KVStore{db.blkWriteStore, db.trxWriteStore, db.irrBlockStore})
}

func (db *DB) itrAllStores(f storeIterFunc) error {
	return db.itrStores(f, []store.KVStore{db.blkWriteStore, db.trxWriteStore, db.blksReadStore, db.trxReadStore, db.irrBlockStore})
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

func (db *DB) setupReadWriteOpts(driver store.KVStore, read, write []string) {
	if inSlice("blk", read) {
		db.blksReadStore = driver
	}

	if inSlice("trx", read) {
		db.trxReadStore = driver
		// trx WINS and overrides
	}

	if inSlice("all", read) {
		db.trxReadStore = driver
		db.blksReadStore = driver
	}
	// read == "none" sets nothing

	if inSlice("blk", write) {
		db.blkWriteStore = driver
		db.irrBlockStore = driver
	}

	if inSlice("trx", write) {
		db.trxWriteStore = driver
		// trx WINS and overrides
		db.irrBlockStore = driver
	}
	if inSlice("all", write) {
		db.trxWriteStore = driver
		db.blkWriteStore = driver
		db.irrBlockStore = driver
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
	if db.blkWriteStore != nil {
		fields = append(fields, zap.Bool("blk_write_store_enabled", true))
	} else {
		fields = append(fields, zap.Bool("blk_write_store_enabled", false))
	}
	if db.trxWriteStore != nil {
		fields = append(fields, zap.Bool("trx_write_store_enabled", true))
	} else {
		fields = append(fields, zap.Bool("trx_write_store_enabled", false))
	}
	if db.blksReadStore != nil {
		fields = append(fields, zap.Bool("blk_read_store", true))
	} else {
		fields = append(fields, zap.Bool("blk_read_store", false))
	}
	if db.trxReadStore != nil {
		fields = append(fields, zap.Bool("trx_read_store", true))
	} else {
		fields = append(fields, zap.Bool("trx_read_store", false))
	}
	if db.irrBlockStore != nil {
		fields = append(fields, zap.Bool("irr_block_store", true))
	} else {
		fields = append(fields, zap.Bool("irr_block_store", false))
	}
	db.logger.Info("trxdb driver dump", fields...)
}
