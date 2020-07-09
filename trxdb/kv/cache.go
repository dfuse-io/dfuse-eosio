package kv

import (
	"fmt"
	"sync"

	"github.com/dfuse-io/kvdb/store"
)

var storeCachePool = make(map[string]store.KVStore)
var storeCachePoolLock sync.Mutex

func newCachedKVDB(dsn string) (out store.KVStore, err error) {
	storeCachePoolLock.Lock()
	defer storeCachePoolLock.Unlock()

	out = storeCachePool[dsn]
	if out == nil {
		zlog.Debug("kv store store is not cached for this DSN, creating a new one")
		out, err = store.New(dsn)
		if err != nil {
			return nil, fmt.Errorf("new kvdb store: %w", err)
		}
		storeCachePool[dsn] = out
	} else {
		zlog.Debug("re-using cached kv store")
	}
	return out, nil
}
