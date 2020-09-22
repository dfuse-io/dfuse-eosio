package purger

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/injector"
	"github.com/dfuse-io/kvdb/store"
	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_purgeAccounts(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	kvStore = injector.NewRWCache(kvStore)
	ctx := context.Background()

	maxEntries := uint64(10)
	accountA := eos.MustStringToName("a") // only and maxed out on shard 0
	accountB := eos.MustStringToName("b") // on shard 0 and maxed out on shard 1
	accountC := eos.MustStringToName("c") // not maxed out
	accountD := eos.MustStringToName("d") // no actions
	accountE := eos.MustStringToName("e") // only on shard 1 and maxed out on shard 1

	shard0Service := setupAccountInjector(kvStore, 0, 10)
	insertKeys(ctx, shard0Service, accountA, 12, 41)
	insertKeys(ctx, shard0Service, accountB, 5, 53)
	insertKeys(ctx, shard0Service, accountC, 3, 58)

	shard1Service := setupAccountInjector(kvStore, 1, 10)
	insertKeys(ctx, shard1Service, accountA, 8, 8)
	insertKeys(ctx, shard1Service, accountB, 7, 16)
	insertKeys(ctx, shard1Service, accountC, 2, 23)
	insertKeys(ctx, shard1Service, accountE, 16, 25)

	shard2Service := setupAccountInjector(kvStore, 2, 10)
	insertKeys(ctx, shard2Service, accountA, 3, 1)
	insertKeys(ctx, shard2Service, accountB, 4, 3)
	insertKeys(ctx, shard2Service, accountC, 1, 7)

	facetFactory := &accounthist.AccountFactory{}
	purger := NewPurger(kvStore, facetFactory, false)
	err := purger.PurgeAccounts(context.Background(), maxEntries, func(facet accounthist.Facet, belowShardNum int, currentCount uint64) {
		fmt.Println(fmt.Sprintf("Purging account %s below shard %d current seen action count %d", facet.String(), belowShardNum, currentCount))
	})
	require.NoError(t, err)

	fmt.Println("verifying accountA: only and maxed out on shard 0")
	seq, err := accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountA), 0, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(12), seq.CurrentOrdinal)

	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountA), 1, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountA), 2, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountB: on shard 0 and maxed out on shard 1")

	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountB), 0, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), seq.CurrentOrdinal)

	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountB), 1, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), seq.CurrentOrdinal)

	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountB), 2, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountC: not maxed out")
	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountC), 0, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), seq.CurrentOrdinal)
	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountC), 1, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), seq.CurrentOrdinal)
	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountC), 2, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), seq.CurrentOrdinal)

	fmt.Println("verifying accountD: no actions")
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountD), 0, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountD), 1, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountD), 2, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountE: only on shard 1 and maxed out on shard 1")
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountE), 0, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
	seq, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountE), 1, facetFactory.DecodeRow, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(16), seq.CurrentOrdinal)
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountE), 2, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
}

type catchDeleteStore struct {
	store.KVStore
	deletedKeys [][]byte
}

func (c *catchDeleteStore) BatchDelete(ctx context.Context, keys [][]byte) error {
	for _, key := range keys {
		fmt.Printf("%x\n", key)
	}

	c.deletedKeys = append(c.deletedKeys, keys...)
	return c.KVStore.BatchDelete(ctx, keys)
}

func Test_purgeAccountAboveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	kvStore = injector.NewRWCache(kvStore)
	ctx := context.Background()

	accountA := eos.MustStringToName("a")

	shard0Service := setupAccountInjector(kvStore, 0, 10)
	insertKeys(ctx, shard0Service, accountA, 12, 26)

	shard1Service := setupAccountInjector(kvStore, 1, 10)
	shard1InsertedKeys := insertKeys(ctx, shard1Service, accountA, 5, 21)

	shard3Service := setupAccountInjector(kvStore, 3, 10)
	shard3InsertedKeys := insertKeys(ctx, shard3Service, accountA, 21, 1)

	expectDeleteKeys := shard1InsertedKeys
	expectDeleteKeys = append(expectDeleteKeys, shard3InsertedKeys...)

	deleteStore := &catchDeleteStore{
		KVStore: kvStore,
	}

	facetFactory := &accounthist.AccountFactory{}
	facetKey := accounthist.AccountFacet(accountA)
	p := NewPurger(deleteStore, &accounthist.AccountFactory{}, false)
	p.purgeAccountAboveShard(context.Background(), facetKey, byte(0))

	assert.Equal(t, expectDeleteKeys, deleteStore.deletedKeys)
	_, err := accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountA), 1, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
	_, err = accounthist.ShardSeqDataPerFacet(context.Background(), kvStore, accounthist.AccountFacet(accountA), 3, facetFactory.DecodeRow, false)
	assert.Error(t, err, store.ErrNotFound)
}
