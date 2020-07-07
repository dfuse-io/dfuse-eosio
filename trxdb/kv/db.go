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

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type DB struct {
	store               store.KVStore
	indexableCategories map[pbtrxdb.IndexableCategory]bool

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

func New(dsnString string, logger *zap.Logger) (trxdb.Driver, error) {
	storeCachePoolLock.Lock()
	defer storeCachePoolLock.Unlock()

	cachedKVStore := storeCachePool[dsnString]
	if cachedKVStore == nil {
		logger.Debug("kv store store is not cached for this DSN, creating a new one")
		kvStore, err := store.New(dsnString)
		if err != nil {
			return nil, fmt.Errorf("new kvdb store: %w", err)
		}

		storeCachePool[dsnString] = kvStore
		cachedKVStore = kvStore
	} else {
		logger.Debug("re-using cached kv store")
	}

	logger.Debug("creating new kv trxdb instance")
	db := &DB{
		store:               cachedKVStore,
		enc:                 trxdb.NewProtoEncoder(),
		dec:                 trxdb.NewProtoDecoder(),
		logger:              logger,
		indexableCategories: trxdb.FullIndexing.ToMap(),
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.store.Close()
}
