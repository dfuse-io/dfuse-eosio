package accounthist

import (
	"context"
	"fmt"
	"math"

	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
)

//SCAN 02:0000000000000 02:fffffffffffffff LIMIT 1
//-> 02:000000b1:ff:fffffff0000 -> 65535 actions
//over 1000 (maxEntries), so:
//SCAN 02:0000000b1:00:00000000000000  02:0000000b1:fe:fffffffffffff
//* Get all those keys and delete them
//SCAN: 02:000000000b2 02:FFFFFFFFFFFFFFF LIMIT 1  (that's the next account, b2)
//-> 02:00000eosio:ff:fffffffffff8
//got only 10, let's continue on other shards (shardNum = 0 here), to get
//to a SUM of 1000, THEN start deleting older keys.

func purgeAccounts(ctx context.Context, kvStore store.KVStore, maxEntries uint64) error {
	startKey := encodeActionPrefixKey(0)
	endKey := encodeActionPrefixKey(math.MaxUint64)
	it := kvStore.Scan(ctx, startKey, endKey, 0)
	for it.Next() {
		account, shardNum, ordinalNum := decodeActionKeySeqNum(it.Item().Key)
		zlog.Info("found account",
			zap.String("account", EOSName(account).String()),
			zap.Int("shard_num", int(shardNum)),
			zap.Int("ordinal_num", int(ordinalNum)),
		)
		if ordinalNum > maxEntries {
			zlog.Info("account action count exceed max entries",
				zap.String("account", EOSName(account).String()),
				zap.Uint64("max_entries", maxEntries),
				zap.Int("ordinal_num", int(ordinalNum)),
			)
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("fetching accounts: %w", err)
	}
	return nil
}

var batchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
	kvStore.BatchDelete(ctx, keys)
}

func (s *Service) purgeAccountAboveShard(ctx context.Context, account uint64, shardNum byte) error {
	startKey := encodeActionKey(account, shardNum, 0)
	endKey := store.Key(encodeActionPrefixKey(account)).PrefixNext()
	it := s.kvStore.Scan(ctx, startKey, endKey, 0)
	zlog.Info("purging account actions above a certain shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("shard_num", int(shardNum)),
		zap.Stringer("start_key", Key(startKey)),
		zap.Stringer("end_key", Key(endKey)),
	)
	count := uint64(0)
	for it.Next() {
		count++
		batchDeleteKeys(ctx, s.kvStore, [][]byte{it.Item().Key})
	}
	if it.Err() != nil {
		return it.Err()
	}

	zlog.Info("account purged above shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("shard_num", int(shardNum)),
		zap.Uint64("deleted_keys_count", count),
	)

	s.kvStore.FlushPuts(ctx)
	return nil
}
