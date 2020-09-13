package injector

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
)

type ShardDetail struct {
	ShardNum   byte
	Checkpoint *pbaccounthist.ShardCheckpoint
}

func (i *Injector) ShardAnalysis(ctx context.Context, checkpointPrefix byte) (out []*ShardDetail, err error) {

	startKey := []byte{checkpointPrefix}
	endKey := store.Key(startKey).PrefixNext()

	it := i.KvStore.Scan(ctx, startKey, endKey, 0)

	for it.Next() {

		shardByte := keyer.DecodeCheckpointKey(it.Item().Key)
		checkpoint := &pbaccounthist.ShardCheckpoint{}
		if err := proto.Unmarshal(it.Item().Value, checkpoint); err != nil {
			return nil, err
		}
		out = append(out, &ShardDetail{
			ShardNum:   shardByte,
			Checkpoint: checkpoint,
		})
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("unable to scan shard: %w", it.Err())
	}
	return out, nil
}

type KeyShardSummary struct {
	ShardNum byte
	SeqData  accounthist.SequenceData
}

func (i *Injector) KeySummary(ctx context.Context, key accounthist.ActionKey) ([]*KeyShardSummary, error) {
	out := []*KeyShardSummary{}
	for j := 0; j < 5; j++ {
		// TODO: fix contract
		seqData, err := accounthist.ShardNewestSequenceData(ctx, i.KvStore, key, byte(j), InjectorRowKeyDecoder, true)

		if err == store.ErrNotFound {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error while fetching sequence data for account: %w", err)
		}

		out = append(out, &KeyShardSummary{ShardNum: byte(j), SeqData: seqData})
	}
	return out, nil
}
