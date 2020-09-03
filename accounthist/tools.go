package accounthist

import (
	"context"
	"fmt"
	"math"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func shardSummary(ctx context.Context, kvStore store.KVStore, account uint64, maxShardSize int) (out []*sequenceData, err error) {
	nextShardNum := byte(0)
	for i := 0; i < maxShardSize; i++ {
		startKey := encodeActionKey(account, nextShardNum, math.MaxUint64)
		endKey := encodeActionKey(account+1, 0, 0)

		ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
		defer cancel()

		it := kvStore.Scan(ctx, startKey, endKey, 1)
		for it.Next() {

			newact := &pbaccounthist.ActionRow{}
			if err = proto.Unmarshal(it.Item().Value, newact); err != nil {
				return
			}

			_, currentShardNum, historySeqNum := decodeActionKeySeqNum(it.Item().Key)
			out = append(out, &sequenceData{
				nextOrdinal:        historySeqNum + 1,
				lastGlobalSeq:      newact.ActionTrace.Receipt.GlobalSequence,
				lastDeletedOrdinal: newact.LastDeletedSeq,
				maxEntries:         0,
			})
			nextShardNum = currentShardNum + 1
		}
		if it.Err() != nil {
			err = it.Err()
			return
		}
	}

	return
}

func shardLastSequenceData(ctx context.Context, kvstore store.KVStore, account uint64, shardNum byte) (out sequenceData, err error) {
	startKey := encodeActionKey(account, shardNum, math.MaxUint64)
	endKey := store.Key(encodeActionPrefixKey(account)).PrefixNext()

	zlog.Info("reading last sequence data for current shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("current_shard_num", int(shardNum)),
		zap.Stringer("start_key", Key(startKey)),
		zap.Stringer("end_key", Key(endKey)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := kvstore.Scan(ctx, startKey, endKey, 1)
	hasNext := it.Next()
	if !hasNext && it.Err() != nil {
		return out, fmt.Errorf("scan last action: %w", it.Err())
	}

	if !hasNext {
		return out, store.ErrNotFound
	}

	newact := &pbaccounthist.ActionRow{}
	if err = proto.Unmarshal(it.Item().Value, newact); err != nil {
		return
	}

	_, _, out.nextOrdinal = decodeActionKeySeqNum(it.Item().Key)
	out.nextOrdinal++
	out.lastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
	out.lastDeletedOrdinal = newact.LastDeletedSeq

	return
}
