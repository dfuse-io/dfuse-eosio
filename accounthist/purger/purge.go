package purger

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
)

type Purger struct {
	kvStore store.KVStore
}

var PurgerKeyEncoder accounthist.KeyEncoderFunc
var PurgerActionKeyPrefix byte
var PurgerRowKeyDecoder accounthist.RowKeyDecoderFunc

func (p *Purger) PurgeAccounts(ctx context.Context, maxEntriesPerAccount uint64) error {
	return accounthist.ScanAccounts(ctx, p.kvStore, PurgerActionKeyPrefix, PurgerRowKeyDecoder, func(account uint64, shardNum byte, ordinalNum uint64) error {
		actionKey := PurgerKeyEncoder(&bstream.Block{}, &pbcodec.ActionTrace{}, account)
		if ordinalNum >= maxEntriesPerAccount {
			zlog.Info("account action count exceed max entries, no need to proceed further",
				zap.Stringer("account", EOSName(account)),
				zap.Uint64("max_entries", maxEntriesPerAccount),
				zap.Int("ordinal_num", int(ordinalNum)),
				zap.Int("shard_num", int(shardNum)),
			)
			p.purgeAccountAboveShard(ctx, account, shardNum)
			return nil
		}

		//shard (value or nil)
		//-> s.purgeAccountAboveShard(ctx, account, shardNum)
		//-> do not purge the account actions hasn't maxed

		nextShardNum := shardNum + 1
		seenActions := ordinalNum
		for {
			seqData, err := accounthist.ShardNewestSequenceData(ctx, p.kvStore, actionKey, nextShardNum, PurgerRowKeyDecoder, false)
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
				p.purgeAccountAboveShard(ctx, account, nextShardNum)
				return nil
			}
			nextShardNum++
		}
	})
}

var BatchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
	kvStore.BatchDelete(ctx, keys)
}

func (p *Purger) purgeAccountAboveShard(ctx context.Context, account uint64, shardNum byte) error {
	actionKey := PurgerKeyEncoder(&bstream.Block{}, &pbcodec.ActionTrace{}, account)
	startKey := actionKey.Row(shardNum, 0)
	endKey := store.Key(keyer.EncodeAccountWithPrefixKey(PurgerActionKeyPrefix, account)).PrefixNext()
	it := p.kvStore.Scan(ctx, startKey, endKey, 0)

	zlog.Info("purging account actions above a certain shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("shard_num", int(shardNum)),
		zap.Stringer("start_key", startKey),
		zap.Stringer("end_key", endKey),
	)
	count := uint64(0)
	for it.Next() {
		count++
		BatchDeleteKeys(ctx, p.kvStore, [][]byte{it.Item().Key})
	}
	if it.Err() != nil {
		return it.Err()
	}

	zlog.Info("account purged above shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("shard_num", int(shardNum)),
		zap.Uint64("deleted_keys_count", count),
	)

	p.kvStore.FlushPuts(ctx)
	return nil
}
