package injector

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

func (i *Injector) getSequenceData(ctx context.Context, key accounthist.Facet) (out accounthist.SequenceData, err error) {
	out, found := i.cacheSeqData[key.String()]
	if found {
		i.currentBatchMetrics.accountCacheHit++
		return
	}
	i.currentBatchMetrics.accountCacheMiss++

	if !i.startedFromCheckpoint {
		zlog.Debug("skipping read data sequence accounthist service did not start from a checkpoint there is nothing to read",
			zap.String("key", key.String()),
			zap.Int("shard_num", int(i.ShardNum)),
		)
		out.MaxEntries = i.MaxEntries
		i.UpdateSeqData(key, out)
		return
	}
	out, err = accounthist.ShardSeqDataPerFacet(ctx, i.KvStore, key, i.ShardNum, i.facetFactory.DecodeRow, true)
	if err == store.ErrNotFound {
		zlog.Debug("account never seen before, initializing a new sequence data",
			zap.Stringer("key", key),
		)
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error while fetching token sequence data: %w", err)
		return
	}

	out.MaxEntries = i.MaxEntries
	zlog.Debug("token sequence data setup",
		zap.Int("shard_num", int(i.ShardNum)),
		zap.Stringer("key", key),
		zap.Uint64("seq_data_current_ordinal", out.CurrentOrdinal),
		zap.Uint64("seq_data_max_entries", out.MaxEntries),
		zap.Uint64("seq_data_last_global_sequence", out.LastGlobalSeq),
		zap.Uint64("seq_data_last_deleted_ordinal", out.LastDeletedOrdinal),
	)
	i.UpdateSeqData(key, out)
	return
}
