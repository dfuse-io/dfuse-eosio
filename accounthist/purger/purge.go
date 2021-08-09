package purger

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"go.uber.org/zap"

	"github.com/streamingfast/kvdb/store"
)

type LogFunc func(facet accounthist.Facet, belowShardNum int, currentCount uint64)
type Purger struct {
	kvStore      store.KVStore
	facetFactory accounthist.FacetFactory
	enableDryRun bool
}

func NewPurger(kvStore store.KVStore, facetFactory accounthist.FacetFactory, enableDryRun bool) *Purger {
	return &Purger{
		kvStore:      kvStore,
		facetFactory: facetFactory,
		enableDryRun: enableDryRun,
	}
}

func (p *Purger) PurgeAccounts(ctx context.Context, maxEntriesPerAccount uint64, logFunc LogFunc) error {

	zlog.Info("purging accounts", zap.Uint64("max_entries", maxEntriesPerAccount), zap.Bool("dry_run", p.enableDryRun))

	return accounthist.ScanFacets(ctx, p.kvStore, p.facetFactory.Collection(), p.facetFactory.DecodeRow, func(facet accounthist.Facet, baseShardNum byte, ordinalNum uint64) error {
		zlog.Debug("purging facet",
			zap.Stringer("facet", facet),
			zap.Int("base_shard_num", int(baseShardNum)),
			zap.Uint64("ordinal_num", ordinalNum),
		)
		currentShardNum := baseShardNum
		seenActions := ordinalNum
		for {
			if seenActions >= maxEntriesPerAccount {
				zlog.Info("account action count exceed max entries",
					zap.Stringer("facet", facet),
					zap.Uint64("max_entries", maxEntriesPerAccount),
					zap.Uint64("seen_actions", seenActions),
					zap.Int("current_shard", int(currentShardNum)),
				)

				logFunc(facet, int(currentShardNum), seenActions)
				p.purgeAccountAboveShard(ctx, facet, currentShardNum)
				return nil
			}

			seqData, latestShardNum, err := accounthist.LatestShardSeqDataPerFacet(ctx, p.kvStore, facet, currentShardNum+1, p.facetFactory.DecodeRow, false)
			if err == store.ErrNotFound {
				zlog.Info("account has not been maxed out",
					zap.Stringer("facet", facet),
					zap.Uint64("action_count", seenActions),
					zap.Int("last_shard_num", int(latestShardNum)),
				)
				return nil
			} else if err != nil {
				zlog.Info("error while fetching sequence data for account",
					zap.String("error", err.Error()),
					zap.Stringer("facet", facet),
					zap.Uint64("action_count", seenActions),
					zap.Int("shard_num", int(latestShardNum)),
				)
				return fmt.Errorf("error while fetching sequence data for account: %w", err)
			}

			seenActions += seqData.CurrentOrdinal
			currentShardNum = latestShardNum
		}
	})
}

func (p *Purger) purgeAccountAboveShard(ctx context.Context, facet accounthist.Facet, shardNum byte) error {
	startKey, endKey := accounthist.FacetRangeLowerBound(facet, shardNum+1)
	it := p.kvStore.Scan(ctx, startKey, endKey, 0)

	zlog.Info("purging account actions above a certain shard",
		zap.Stringer("facet_key", facet),
		zap.Int("shard_num", int(shardNum)),
		zap.Stringer("start_key", startKey),
		zap.Stringer("end_key", endKey),
	)
	count := uint64(0)
	for it.Next() {
		count++

		if p.enableDryRun {
			continue
		}

		err := p.kvStore.BatchDelete(ctx, [][]byte{it.Item().Key})
		if err != nil {
			return fmt.Errorf("error deleteing batch keys: %w", err)
		}
	}
	if it.Err() != nil {
		return it.Err()
	}

	zlog.Info("account purged above shard",
		zap.Stringer("facet_key", facet),
		zap.Int("shard_num", int(shardNum)),
		zap.Uint64("deleted_keys_count", count),
	)

	p.kvStore.FlushPuts(ctx)
	return nil
}
