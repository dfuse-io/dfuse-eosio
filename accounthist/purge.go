package accounthist

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
)

func (s *Service) purgeAccounts(ctx context.Context, maxEntriesPerAccount uint64) error {
	return s.ScanAccounts(ctx, func(account uint64, shardNum byte, ordinalNum uint64) error {
		if ordinalNum >= maxEntriesPerAccount {
			zlog.Info("account action count exceed max entries, no need to proceed further",
				zap.Stringer("account", EOSName(account)),
				zap.Uint64("max_entries", maxEntriesPerAccount),
				zap.Int("ordinal_num", int(ordinalNum)),
				zap.Int("shard_num", int(shardNum)),
			)
			s.purgeAccountAboveShard(ctx, account, shardNum)
			return nil
		}

		//shard (value or nil)
		//-> s.purgeAccountAboveShard(ctx, account, shardNum)
		//-> do not purge the account actions hasn't maxed

		nextShardNum := shardNum + 1
		seenActions := ordinalNum
		for {
			seqData, err := s.shardNewestSequenceData(ctx, account, nextShardNum, func(item store.KV) (SequenceData, error) {
				s := SequenceData{}
				_, _, s.CurrentOrdinal = decodeActionKeySeqNum(item.Key)
				return s, nil
			})

			if err == store.ErrNotFound {
				zlog.Info("account has not been maxed out", zap.String("account", EOSName(account).String()), zap.Uint64("action_count", seenActions), zap.Int("last_shard_num", int(shardNum-1)))
				return nil
			} else if err != nil {
				zlog.Info("error while fetching sequence data for account", zap.String("error", err.Error()), zap.String("account", EOSName(account).String()), zap.Uint64("action_count", seenActions), zap.Int("shard_num", int(shardNum)))
				return fmt.Errorf("error while fetching sequence data for account: %w", err)
			}

			seenActions += seqData.CurrentOrdinal
			if seenActions >= maxEntriesPerAccount {
				zlog.Info("account action count exceed max entries",
					zap.String("account", EOSName(account).String()),
					zap.Uint64("max_entries", maxEntriesPerAccount),
					zap.Int("ordinal_num", int(seqData.CurrentOrdinal)),
					zap.Uint64("seen_actions", seenActions),
					zap.Int("shard_num", int(nextShardNum)),
				)
				s.purgeAccountAboveShard(ctx, account, nextShardNum)
				return nil
			}
			nextShardNum++
		}
	})
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
