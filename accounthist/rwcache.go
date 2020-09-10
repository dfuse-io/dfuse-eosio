package accounthist

import (
	"context"
	"time"

	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type RWCache struct {
	store.KVStore

	orderedPuts []string
	puts        map[string][]byte
	deletes     map[string]struct{}

	isLastRow func(key []byte) bool
}

func NewRWCache(backingStore store.KVStore) *RWCache {
	return &RWCache{
		puts:        map[string][]byte{},
		orderedPuts: []string{},
		deletes:     map[string]struct{}{},
		KVStore:     backingStore,
		isLastRow:   func(key []byte) bool { return key[0] == prefixLastBlock },
	}
}

func (c *RWCache) Put(ctx context.Context, key, value []byte) error {
	skey := string(key)
	if _, found := c.puts[skey]; !found {
		c.orderedPuts = append(c.orderedPuts, skey)
	}
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
	t0 := time.Now()

	var countFirstKeys, countLastKeys int
	lastKeys := map[string][]byte{}

	c.OrderedPuts(func(sKey string, value []byte) error {
		bkey := []byte(sKey)
		if c.isLastRow != nil && c.isLastRow(bkey) {
			lastKeys[sKey] = value
			return nil
		}
		countFirstKeys++
		if err := c.KVStore.Put(ctx, bkey, value); err != nil {
			return err
		}
		return nil
	})

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

	for k, v := range lastKeys {
		countLastKeys++
		if err := c.KVStore.Put(ctx, []byte(k), v); err != nil {
			return err
		}
	}

	if err := c.KVStore.FlushPuts(ctx); err != nil {
		return err
	}

	d0 := time.Since(t0)
	zlog.Info("flushed keys to storage",
		zap.Int("put_first_keys", countFirstKeys),
		zap.Int("put_last_keys", countLastKeys),
		zap.Int("deleted_keys", len(keys)),
		zap.Duration("time_delta", d0),
	)

	c.puts = map[string][]byte{}
	c.deletes = map[string]struct{}{}
	c.orderedPuts = []string{}
	return nil
}

func (c *RWCache) OrderedPuts(f func(sKey string, value []byte) error) error {
	for _, sKey := range c.orderedPuts {
		if v, found := c.puts[sKey]; found {
			if err := f(sKey, v); err != nil {
				return err
			}
		}
	}
	return nil
}
