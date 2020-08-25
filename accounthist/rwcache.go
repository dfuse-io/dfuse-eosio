package accounthist

import (
	"context"

	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type RWCache struct {
	store.KVStore

	puts    map[string][]byte
	deletes map[string]struct{}

	isLastRow func(key []byte) bool
}

func NewRWCache(backingStore store.KVStore) *RWCache {
	return &RWCache{
		puts:      map[string][]byte{},
		deletes:   map[string]struct{}{},
		KVStore:   backingStore,
		isLastRow: func(key []byte) bool { return key[0] == prefixLastBlock },
	}
}

func (c *RWCache) Put(ctx context.Context, key, value []byte) error {
	skey := string(key)
	c.puts[skey] = value
	delete(c.deletes, skey)
	return nil
}

func (c *RWCache) BatchDelete(ctx context.Context, keys [][]byte) error {
	for _, key := range keys {
		skey := string(key)
		if _, found := c.puts[skey]; found {
			delete(c.puts, skey)
		} else {
			c.deletes[skey] = struct{}{}
		}
	}
	return nil
}

func (c *RWCache) FlushPuts(ctx context.Context) error {
	var countFirstKeys, countLastKeys int
	lastKeys := map[string][]byte{}
	for k, v := range c.puts {
		bkey := []byte(k)
		if c.isLastRow != nil && c.isLastRow(bkey) {
			lastKeys[k] = v
			continue
		}
		countFirstKeys++
		if err := c.KVStore.Put(ctx, bkey, v); err != nil {
			return err
		}
	}
	if err := c.KVStore.FlushPuts(ctx); err != nil {
		return err
	}

	// Put some rows last
	for k, v := range lastKeys {
		countLastKeys++
		if err := c.KVStore.Put(ctx, []byte(k), v); err != nil {
			return err
		}
	}
	if err := c.KVStore.FlushPuts(ctx); err != nil {
		return err
	}

	keys := make([][]byte, 0, len(c.deletes))
	for k := range c.deletes {
		keys = append(keys, []byte(k))
	}

	if err := c.KVStore.BatchDelete(ctx, keys); err != nil {
		return err
	}

	zlog.Info("flushed keys to storage", zap.Int("put_first_keys", countFirstKeys), zap.Int("put_last_keys", countLastKeys), zap.Int("deleted_keys", len(keys)))

	c.puts = map[string][]byte{}
	c.deletes = map[string]struct{}{}

	return nil
}
