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
	accountKey := AccountKey(accountUint)

	kvStore.Put(ctx, accountKey.Row(0, 1), []byte{0x02})
	kvStore.Put(ctx, accountKey.Row(2, 7), []byte{0x03})
	kvStore.FlushPuts(ctx)

	seq, err := ShardNewestSequenceData(ctx, kvStore, accountKey, 0, AccountKeyRowDecoder, false)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), seq.CurrentOrdinal)

	_, err = ShardNewestSequenceData(ctx, kvStore, accountKey, 1, AccountKeyRowDecoder, false)
	require.Error(t, store.ErrNotFound)

	seq, err = ShardNewestSequenceData(ctx, kvStore, accountKey, 2, AccountKeyRowDecoder, false)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), seq.CurrentOrdinal)

}
