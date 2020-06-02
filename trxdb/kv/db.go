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
)

type DB struct {
	store store.KVStore

	// Required only when writing
	writerChainID []byte

	enc *trxdb.ProtoEncoder
	dec *trxdb.ProtoDecoder
}

var dbCachePool = make(map[string]trxdb.Driver)
var dbCachePoolLock sync.Mutex

func init() {
	trxdb.Register("badger", New)
	trxdb.Register("tikv", New)
	trxdb.Register("bigkv", New)
	trxdb.Register("cznickv", New)
}

func New(dsnString string, opts ...trxdb.Option) (trxdb.Driver, error) {
	dbCachePoolLock.Lock()
	defer dbCachePoolLock.Unlock()

	db := dbCachePool[dsnString]
	if db == nil {

		kvStore, err := store.New(dsnString)
		if err != nil {
			return nil, fmt.Errorf("badger new: open badger db: %w", err)
		}

		db = trxdb.Driver(&DB{
			store: kvStore,
			enc:   trxdb.NewProtoEncoder(),
			dec:   trxdb.NewProtoDecoder(),
		})
		dbCachePool[dsnString] = db
	}

	return db, nil
}
