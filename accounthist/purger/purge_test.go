package purger

import (
	_ "github.com/dfuse-io/kvdb/store/badger"
)

//func Test_purgeAccounts(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	kvStore = injector.NewRWCache(kvStore)
//	ctx := context.Background()
//
//	maxEntries := uint64(10)
//	accountA := eos.MustStringToName("a") // only and maxed out on shard 0
//	accountB := eos.MustStringToName("b") // on shard 0 and maxed out on shard 1
//	accountC := eos.MustStringToName("c") // not maxed out
//	accountD := eos.MustStringToName("d") // no actions
//	accountE := eos.MustStringToName("e") // only on shard 1 and maxed out on shard 1
//
//	shard0Service := setupAccountInjector(kvStore, 0, 10)
//	insertKeys(ctx, shard0Service, accountA, 12, 41)
//	insertKeys(ctx, shard0Service, accountB, 5, 53)
//	insertKeys(ctx, shard0Service, accountC, 3, 58)
//
//	shard1Service := setupAccountInjector(kvStore, 1, 10)
//	insertKeys(ctx, shard1Service, accountA, 8, 8)
//	insertKeys(ctx, shard1Service, accountB, 7, 16)
//	insertKeys(ctx, shard1Service, accountC, 2, 23)
//	insertKeys(ctx, shard1Service, accountE, 16, 25)
//
//	shard2Service := setupAccountInjector(kvStore, 2, 10)
//	insertKeys(ctx, shard2Service, accountA, 3, 1)
//	insertKeys(ctx, shard2Service, accountB, 4, 3)
//	insertKeys(ctx, shard2Service, accountC, 1, 7)
//
//	purger := Purger{
//		kvStore: kvStore,
//	}
//
//	err := purger.PurgeAccounts(context.Background(), maxEntries)
//	require.NoError(t, err)
//
//	fmt.Println("verifying accountA: only and maxed out on shard 0")
//	seq, err := accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountA), 0, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(12), seq.CurrentOrdinal)
//
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountA), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountA), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//
//	fmt.Println("verifying accountB: on shard 0 and maxed out on shard 1")
//
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountB), 0, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(5), seq.CurrentOrdinal)
//
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountB), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(7), seq.CurrentOrdinal)
//
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountB), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//
//	fmt.Println("verifying accountC: not maxed out")
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountC), 0, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(3), seq.CurrentOrdinal)
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountC), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(2), seq.CurrentOrdinal)
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountC), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(1), seq.CurrentOrdinal)
//
//	fmt.Println("verifying accountD: no actions")
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountD), 0, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountD), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountD), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//
//	fmt.Println("verifying accountE: only on shard 1 and maxed out on shard 1")
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountE), 0, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//	seq, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountE), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.NoError(t, err)
//	assert.Equal(t, uint64(16), seq.CurrentOrdinal)
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountE), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//}
//
//func Test_purgeAccountAboveShard(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	kvStore = injector.NewRWCache(kvStore)
//	ctx := context.Background()
//
//	accountA := eos.MustStringToName("a")
//
//	shard0Service := setupAccountInjector(kvStore, 0, 10)
//	insertKeys(ctx, shard0Service, accountA, 12, 26)
//
//	shard1Service := setupAccountInjector(kvStore, 1, 10)
//	shard1InsertedKeys := insertKeys(ctx, shard1Service, accountA, 5, 21)
//
//	shard2Service := setupAccountInjector(kvStore, 2, 10)
//	shard2InsertedKeys := insertKeys(ctx, shard2Service, accountA, 21, 1)
//
//	deletedKeys := [][]byte{}
//	expectDeleteKeys := shard1InsertedKeys
//	expectDeleteKeys = append(expectDeleteKeys, shard2InsertedKeys...)
//
//	BatchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
//		deletedKeys = append(deletedKeys, keys...)
//		kvStore.BatchDelete(ctx, keys)
//	}
//
//	purger := Purger{kvStore: kvStore}
//	purger.purgeAccountAboveShard(context.Background(), accountA, byte(0))
//	assert.Equal(t, expectDeleteKeys, deletedKeys)
//	_, err := accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountA), 1, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//	_, err = accounthist.ShardNewestSequenceData(context.Background(), kvStore, accounthist.AccountKey(accountA), 2, accounthist.AccountKeyRowDecoder, false)
//	assert.Error(t, err, store.ErrNotFound)
//}
