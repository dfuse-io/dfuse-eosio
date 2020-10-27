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

func (i *Injector) ShardCheckpointAnalysis(ctx context.Context, checkpointPrefix byte) (out []*ShardDetail, err error) {

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

type FacetShardDetail struct {
	ShardNum      byte
	LatestSeqData accounthist.SequenceData
	RowKeyCount   uint64
}

type FacetShardSummary struct {
	ShardNum byte
	SeqData  accounthist.SequenceData
}

func (i *Injector) FacetShardsSummary(ctx context.Context, key accounthist.Facet) ([]*FacetShardSummary, error) {
	out := []*FacetShardSummary{}
	currentShardNum := byte(0)
	for {
		// TODO: fix contract
		seqData, shardNum, err := accounthist.LatestShardSeqDataPerFacet(ctx, i.KvStore, key, currentShardNum, i.facetFactory.DecodeRow, true)
		if err == store.ErrNotFound {
			return out, nil
		} else if err != nil {
			return nil, fmt.Errorf("error while fetching sequence data for account: %w", err)
		}
		out = append(out, &FacetShardSummary{ShardNum: shardNum, SeqData: seqData})
		currentShardNum = (shardNum + 1)
	}
	return out, nil
}

func (i *Injector) FacetShardSummary(ctx context.Context, facet accounthist.Facet, shardNum byte) (*FacetShardDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()
	out := &FacetShardDetail{ShardNum: shardNum}

	startKey, endKey := accounthist.FacetShardRange(facet, shardNum)
	it := i.KvStore.Scan(ctx, startKey, endKey, 0)
	hasSeenAction := false
	for it.Next() {
		if !hasSeenAction {
			_, _, currentOrdinal := i.facetFactory.DecodeRow(it.Item().Key)
			out.LatestSeqData = accounthist.SequenceData{
				CurrentOrdinal: currentOrdinal,
			}

			newact := &pbaccounthist.ActionRow{}
			if err := proto.Unmarshal(it.Item().Value, newact); err != nil {
				return nil, fmt.Errorf("unable to decode row: %w", err)
			}
			out.LatestSeqData.LastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
			out.LatestSeqData.LastDeletedOrdinal = newact.LastDeletedSeq
			hasSeenAction = true
		}
		out.RowKeyCount++
	}
	if it.Err() != nil {
		return nil, fmt.Errorf("scan action: %w", it.Err())
	}
	return out, nil
}
