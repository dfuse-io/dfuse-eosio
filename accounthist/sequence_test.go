package accounthist

import (
	"context"
	"testing"

	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardNewestSequenceData(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()
	accountUint := eos.MustStringToName("a")
	accountKey := AccountFacet(accountUint)

	kvStore.Put(ctx, accountKey.Row(0, 1), []byte{0x02})
	kvStore.Put(ctx, accountKey.Row(2, 7), []byte{0x03})
	kvStore.FlushPuts(ctx)

	facetFactory := &AccountFactory{}

	seq, err := ShardSeqDataPerFacet(ctx, kvStore, accountKey, 0, facetFactory.DecodeRow, false)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), seq.CurrentOrdinal)

	_, err = ShardSeqDataPerFacet(ctx, kvStore, accountKey, 1, facetFactory.DecodeRow, false)
	require.Error(t, store.ErrNotFound)

	seq, err = ShardSeqDataPerFacet(ctx, kvStore, accountKey, 2, facetFactory.DecodeRow, false)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), seq.CurrentOrdinal)

}
