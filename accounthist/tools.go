package accounthist

import (
	"context"
	"fmt"

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
