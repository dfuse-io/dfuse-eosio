package accounthist

import (
	"context"
	"fmt"
	"math"

	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const sequenceDataValueLength = 24

type SequenceData struct {
	CurrentOrdinal     uint64 // while in memory, this value is the last written shard ordinal number that was assisgned
	LastGlobalSeq      uint64 // taken from the top-most action stored in this shard, defines by the chain
	LastDeletedOrdinal uint64 // taken from the top-most action stored in this shard
	MaxEntries         uint64 // initialized with the process' max entries per account, but can be reduced if some more recent shards covered this account
}

func (sqd *SequenceData) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("current_ordinal", sqd.CurrentOrdinal)
	encoder.AddUint64("last_global_seq", sqd.LastGlobalSeq)
	return nil
}

func (ws *Service) shardNewestSequenceData(ctx context.Context, account uint64, shardNum byte, process func(item store.KV) (SequenceData, error)) (SequenceData, error) {
	startKey := encodeActionKey(account, shardNum, math.MaxUint64)
	endKey := store.Key(encodeActionKey(account, shardNum, 0)).PrefixNext()

	zlog.Info("reading last sequence data for current shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("current_shard_num", int(shardNum)),
		zap.Stringer("start_key", Key(startKey)),
		zap.Stringer("end_key", Key(endKey)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := ws.kvStore.Scan(ctx, startKey, endKey, 1)
	hasNext := it.Next()
	if !hasNext && it.Err() != nil {
		return SequenceData{}, fmt.Errorf("scan last action: %w", it.Err())
	}

	if !hasNext {
		return SequenceData{}, store.ErrNotFound
	}

	return process(it.Item())
}
