package accounthist

import (
	"context"
	"fmt"

	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/kvdb/store"
	"go.uber.org/zap"
)

type FacetHandlerFunc func(facet Facet, shard byte, ordinalNum uint64) error

func ScanFacets(
	ctx context.Context,
	kvStore store.KVStore,
	facetCollectionPrefix byte,
	decoder RowKeyDecoderFunc,
	facetFunc FacetHandlerFunc,
) error {
	currentKey := []byte{facetCollectionPrefix}
	endKey := store.Key(currentKey).PrefixNext()
	hasMoreFacet := true
	for hasMoreFacet {
		it := kvStore.Scan(ctx, currentKey, endKey, 1)
		hasNext := it.Next()
		if !hasNext && it.Err() != nil {
			return fmt.Errorf("scanning accounts last action: %w", it.Err())
		}

		if !hasNext {
			hasMoreFacet = false
			continue
		}
		facetKey, shardNum, ordinalNum := decoder(it.Item().Key)
		zlog.Info("found facet",
			zap.Stringer("facet_key", facetKey),
			zap.Int("shard_num", int(shardNum)),
			zap.Uint64("ordinal_num", ordinalNum),
		)

		err := facetFunc(facetKey, shardNum, ordinalNum)
		if err != nil {
			return fmt.Errorf("handle facet failed for account %s at shard %d with ordinal number: %d: %w", eos.NameToString(facetKey.Account()), int(shardNum), ordinalNum, err)
		}

		currentKey = store.Key(facetKey.Bytes()).PrefixNext()
	}

	return nil
}
