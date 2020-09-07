package accounthist

import (
	"context"
	"fmt"
	"math"

	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (s *Service) ScanAccounts(ctx context.Context, handleAccount func(account uint64, shard byte, ordinalNum uint64) error) error {
	startKey := encodeActionPrefixKey(0)
	hasMoreAccounts := true

	for hasMoreAccounts {
		endKey := encodeActionPrefixKey(math.MaxUint64)
		it := s.kvStore.Scan(ctx, startKey, endKey, 1)
		hasNext := it.Next()
		if !hasNext && it.Err() != nil {
			return fmt.Errorf("scanning accounts last action: %w", it.Err())
		}

		if !hasNext {
			hasMoreAccounts = false
			continue
		}

		account, shardNum, ordinalNum := decodeActionKeySeqNum(it.Item().Key)
		zlog.Info("found account",
			zap.String("account", EOSName(account).String()),
			zap.Int("shard_num", int(shardNum)),
			zap.Int("ordinal_num", int(ordinalNum)),
		)
		err := handleAccount(account, shardNum, ordinalNum)
		if err != nil {
			return fmt.Errorf("handle account failed for account %s at shard %d with ordinal number: %d: %w", eos.NameToString(account), int(shardNum), ordinalNum, err)
		}

		startKey = store.Key(encodeActionPrefixKey(account)).PrefixNext()
	}

	return nil
}
