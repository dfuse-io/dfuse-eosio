package accounthist

import (
	"context"
	"fmt"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/golang/protobuf/proto"

	"github.com/dfuse-io/kvdb/store"
)

type shardSummary struct {
	ShardNum byte
	SeqData  SequenceData
}

func (s *Service) ShardSummary(ctx context.Context, account uint64) ([]*shardSummary, error) {
	out := []*shardSummary{}
	for i := 0; i < 5; i++ {
		seqData, err := s.shardNewestSequenceData(ctx, account, byte(i), s.processSequenceDataKeyValue)

		if err == store.ErrNotFound {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error while fetching sequence data for account: %w", err)
		}

		out = append(out, &shardSummary{ShardNum: byte(i), SeqData: seqData})
	}
	return out, nil
}

type shard struct {
	ShardNum   byte
	Checkpoint *pbaccounthist.ShardCheckpoint
}

func (s *Service) ShardAnalysis(ctx context.Context) (out []*shard, err error) {
	startKey := []byte{prefixLastBlock}
	endKey := store.Key(startKey).PrefixNext()

	it := s.kvStore.Scan(ctx, startKey, endKey, 0)

	for it.Next() {

		shardByte := decodeLastProcessedBlockKey(it.Item().Key)
		checkpoint := &pbaccounthist.ShardCheckpoint{}
		if err := proto.Unmarshal(it.Item().Value, checkpoint); err != nil {
			return nil, err
		}
		out = append(out, &shard{
			ShardNum:   shardByte,
			Checkpoint: checkpoint,
		})
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("unable to scan shard: %w", it.Err())
	}
	return out, nil
}
