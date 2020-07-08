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
	"sync"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type DB struct {
	accountWriteStore  store.KVStore
	timelineWriteStore store.KVStore
	blksWriteStore     store.KVStore
	trxWriteStore      store.KVStore

	accountReadStore  store.KVStore
	timelineReadStore store.KVStore
	blksReadStore     store.KVStore
	trxReadStore      store.KVStore

	irrBlockStore store.KVStore

	// Required only when writing
	writerChainID []byte

	enc *trxdb.ProtoEncoder
	dec *trxdb.ProtoDecoder

	logger *zap.Logger
}

var storeCachePool = make(map[string]store.KVStore)
var storeCachePoolLock sync.Mutex

func init() {
	trxdb.Register("badger", New)
	trxdb.Register("tikv", New)
	trxdb.Register("bigkv", New)
	trxdb.Register("cznickv", New)
}

func newCachedKVDB(logger, dsn string) (store.KVStore, error) {
	storeCachePoolLock.Lock()
	defer storeCachePoolLock.Unlock()

	cachedKVStore := storeCachePool[dsn]
	if cachedKVStore == nil {
		logger.Debug("kv store store is not cached for this DSN, creating a new one")
		kvStore, err := store.New(dsn)
		if err != nil {
			return nil, fmt.Errorf("new kvdb store: %w", err)
		}

		storeCachePool[dsn] = kvStore
		cachedKVStore = kvStore
	} else {
		logger.Debug("re-using cached kv store")
	}
	return cachedKVStore
}

func New(dsnString string, logger *zap.Logger) (trxdb.Driver, error) {
	// SPLIT with " "

	newCachedKVDB(logger, dsn1)

	// "read"'s default is "*"
	// "write"'s default is "*"

	// dgraphql in eos mainnet core: store:///         by default: read=*&write=*
	// dgraphql in curv: store://mainnet/?read=blk,trx&write=none store://curv/?read=trx&write=none
	// dgraphql in curv: store://mainnet/?read=blk&write=none store://curv/?read=trx&write=none

	// trxdb-loader for mainnet core: store:///?write=blk   /* read="*" */
	// trxdb-loader for curv: store:///?write=trx           /* only purpose: NOT WRITE blk */

	// single laptop-style deployment:           store:///             by default: read=*&write=*
	// single laptop-style deployment, secure:   store:///?read=blk,trx&write=none

	// rwsetting=read_blk_only
	// rwsetting=read_trx_only
	// rwsetting=read_all
	// rwsetting=write_trx_only
	// rwsetting=write_all
	// rwsetting=write_blk_only
	// rwsetting=read_write_all

	// trx defaults to "rw"
	// store:///?trx=write,blk=read
	// store:///?trx=rw,blk=none
	// store:///?blk=none
	// store:///?trx=rw,blk=rw
	// store:///

	logger.Debug("creating new kv trxdb instance")
	db := &DB{
		store:               cachedKVStore,
		enc:                 trxdb.NewProtoEncoder(),
		dec:                 trxdb.NewProtoDecoder(),
		logger:              logger,
		indexableCategories: trxdb.FullIndexing.ToMap(),
	}

	for _, dsn1 := range strings.Split(dsnString, " ", -1) {
		parseDSN(dsn1, db)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.store.Close()
}

func parseDSN(logger, dsn string, db *DB) {
	d, err := url.Parse(dsn)
	//panic(err)

	// purge from `read` and `write`

	driver, err := newCachedKVDB(logger, dsn1)
	//panic(err)

	q := d.ParseQuery()
	read := q.Get("read")
	write := q.Get("write")



	if "blk" in read {
		db.blksReadStore = driver
		db.accountReadStore = driver
		db.timelineReadStore = driver
		db.irrBlockStore = driver
	}
	if "trx" in read {
		db.trxReadStore = driver
		// trx WINS and overrides
		db.irrBlockStore = driver
	}
	if "all" in read || read == "" {
		db.trxReadStore = driver
		db.blksReadStore = driver
		db.accountReadStore = driver
		db.timelineReadStore = driver
		db.irrBlockStore = driver
	}
	// read == "none" sets nothing

	if "blk" in write {
		db.blksWriteStore = driver
		db.accountWriteStore = driver
		db.timelineWriteStore = driver
		db.irrBlockStore = driver
	}
	if "trx" in write {
		db.trxWriteStore = driver
		// trx WINS and overrides
		db.irrBlockStore = driver
	}
	if "all" in write || write == "" {
		db.trxWriteStore = driver
		db.blksWriteStore = driver
		db.accountWriteStore = driver
		db.timelineWriteStore = driver
		db.irrBlockStore = driver
	}
	// write == "none" sets nothing

}
