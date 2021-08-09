package kv

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/streamingfast/kvdb/store"
)

var storeCachePool = make(map[string]store.KVStore)
var storeCachePoolLock sync.Mutex

func newCachedKVDB(dsn string) (out store.KVStore, err error) {
	storeCachePoolLock.Lock()
	defer storeCachePoolLock.Unlock()

	out = storeCachePool[dsn]
	if out == nil {
		zlog.Debug("kv store store is not cached for this DSN, creating a new one",
			zap.String("dsn", dsn),
		)
		out, err = store.New(dsn)
		if err != nil {
			return nil, fmt.Errorf("new kvdb store: %w", err)
		}
		storeCachePool[dsn] = out
	} else {
		zlog.Debug("re-using cached kv store",
			zap.String("dsn", dsn),
		)
	}
	return out, nil
}
