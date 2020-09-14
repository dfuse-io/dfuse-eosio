package accounthist

import (
	"context"
	"fmt"
	"math"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func ScanAccounts(
	ctx context.Context,
	kvStore store.KVStore,
	actionKeyPrefix byte,
	decoder RowKeyDecoderFunc,
	handleAccount func(account uint64, shard byte, ordinalNum uint64) error) error {

	startKey := keyer.EncodeAccountWithPrefixKey(actionKeyPrefix, 0)
	hasMoreAccounts := true
	for hasMoreAccounts {
		endKey := keyer.EncodeAccountWithPrefixKey(actionKeyPrefix, math.MaxUint64)
		it := kvStore.Scan(ctx, startKey, endKey, 1)
		hasNext := it.Next()
		if !hasNext && it.Err() != nil {
			return fmt.Errorf("scanning accounts last action: %w", it.Err())
		}

		if !hasNext {
			hasMoreAccounts = false
			continue
		}
		actionKey, shardNum, ordinalNum := decoder(it.Item().Key)
		zlog.Info("found account",
			zap.Stringer("action_key", actionKey),
			zap.Int("shard_num", int(shardNum)),
			zap.Uint64("ordinal_num", ordinalNum),
		)

		err := handleAccount(actionKey.Account(), shardNum, ordinalNum)
		if err != nil {
			return fmt.Errorf("handle account failed for account %s at shard %d with ordinal number: %d: %w", eos.NameToString(actionKey.Account()), int(shardNum), ordinalNum, err)
		}

		startKey = store.Key(keyer.EncodeAccountWithPrefixKey(actionKeyPrefix, actionKey.Account())).PrefixNext()
	}

	return nil
}
