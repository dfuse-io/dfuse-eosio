package accounthist

import (
	"context"
	"time"

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
	t0 := time.Now()

	var countFirstKeys, countLastKeys int
	lastKeys := map[string][]byte{}
	// FIXME: this is RISKY, because keys are written out of order. If
	// that's the case we might write, for a given account, some
	// out-of-order actions and their seqNum.  When we reboot, we
	// won't properly get the highest, and we won't know there are
	// holes below, for this shard.
	// SOLUTION: sort the keys first, in which order?
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
	zlog.Info("flushed keys to storage", zap.Int("put_first_keys", countFirstKeys), zap.Int("put_last_keys", countLastKeys), zap.Int("deleted_keys", len(keys)), zap.Duration("time_delta", d0))

	c.puts = map[string][]byte{}
	c.deletes = map[string]struct{}{}

	return nil
}
