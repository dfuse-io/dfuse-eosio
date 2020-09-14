package accounthist

import (
	"context"
	"encoding/hex"
	"fmt"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/golang/protobuf/proto"

	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

func ShardNewestSequenceData(ctx context.Context, kvStore store.KVStore, key ActionKey, shardNum byte, decoder RowKeyDecoderFunc, unmarshalAction bool) (SequenceData, error) {
	startKey, endKey := key.Range(shardNum)
	zlog.Debug("reading last sequence data for current shard",
		zap.Stringer("key", key),
		zap.Int("current_shard_num", int(shardNum)),
		zap.String("start_key", hex.EncodeToString(startKey)),
		zap.String("end_key", hex.EncodeToString(endKey)),
	)

	ctx, cancel := context.WithTimeout(ctx, DatabaseTimeout)
	defer cancel()

	//t0 := time.Now()
	//defer func() {
	//	i.currentBatchMetrics.totalReadSeqDuration += time.Since(t0)
	//	i.currentBatchMetrics.readSeqCallCount++
	//}()

	it := kvStore.Scan(ctx, startKey, endKey, 1)
	hasNext := it.Next()
	if !hasNext && it.Err() != nil {
		return SequenceData{}, fmt.Errorf("scan last action: %w", it.Err())
	}

	if !hasNext {
		return SequenceData{}, store.ErrNotFound
	}

	seqData := SequenceData{}
	_, _, currentOrdinal := decoder(it.Item().Key)
	seqData.CurrentOrdinal = currentOrdinal

	if unmarshalAction {
		newact := &pbaccounthist.ActionRow{}
		if err := proto.Unmarshal(it.Item().Value, newact); err != nil {
			return SequenceData{}, err
		}
		seqData.LastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
		seqData.LastDeletedOrdinal = newact.LastDeletedSeq
	}
	return seqData, nil
}
