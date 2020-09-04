package accounthist

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"

	"github.com/eoscanada/eos-go"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

func Test_purgeAccounts(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()

	processSequenceDataKeyValue := func(item store.KV) (SequenceData, error) {
		s := SequenceData{}
		_, _, nextOrdinal := decodeActionKeySeqNum(item.Key)
		s.CurrentOrdinal = nextOrdinal
		return s, nil
	}

	maxEntries := uint64(10)
	accountA := eos.MustStringToName("a") // only and maxed out on shard 0
	accountB := eos.MustStringToName("b") // on shard 0 and maxed out on shard 1
	accountC := eos.MustStringToName("c") // not maxed out
	accountD := eos.MustStringToName("d") // no actions
	accountE := eos.MustStringToName("e") // only on shard 1 and maxed out on shard 1

	shard0Service := newTestService(kvStore, 0, 10)
	insertKeys(ctx, shard0Service, accountA, 12, 41)
	insertKeys(ctx, shard0Service, accountB, 5, 53)
	insertKeys(ctx, shard0Service, accountC, 3, 58)

	shard1Service := newTestService(kvStore, 1, 10)
	insertKeys(ctx, shard1Service, accountA, 8, 8)
	insertKeys(ctx, shard1Service, accountB, 7, 16)
	insertKeys(ctx, shard1Service, accountC, 2, 23)
	insertKeys(ctx, shard1Service, accountE, 16, 25)

	shard2Service := newTestService(kvStore, 2, 10)
	insertKeys(ctx, shard2Service, accountA, 3, 1)
	insertKeys(ctx, shard2Service, accountB, 4, 3)
	insertKeys(ctx, shard2Service, accountC, 1, 7)

	err := shard0Service.purgeAccounts(context.Background(), maxEntries)
	require.NoError(t, err)

	fmt.Println("verifying accountA: only and maxed out on shard 0")
	seq, err := shard0Service.shardNewestSequenceData(context.Background(), accountA, 0, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(12), seq.CurrentOrdinal)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountA, 1, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountA, 2, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountB: on shard 0 and maxed out on shard 1")

	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountB, 0, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), seq.CurrentOrdinal)

	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountB, 1, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), seq.CurrentOrdinal)

	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountB, 2, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountC: not maxed out")
	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountC, 0, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), seq.CurrentOrdinal)
	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountC, 1, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), seq.CurrentOrdinal)
	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountC, 2, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), seq.CurrentOrdinal)

	fmt.Println("verifying accountD: no actions")
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountD, 0, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountD, 1, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountD, 2, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)

	fmt.Println("verifying accountE: only on shard 1 and maxed out on shard 1")
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountE, 0, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
	seq, err = shard0Service.shardNewestSequenceData(context.Background(), accountE, 1, processSequenceDataKeyValue)
	assert.NoError(t, err)
	assert.Equal(t, uint64(16), seq.CurrentOrdinal)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountE, 2, processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
}

func Test_purgeAccountAboveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()

	accountA := eos.MustStringToName("a")

	shard0Service := newTestService(kvStore, 0, 10)
	insertKeys(ctx, shard0Service, accountA, 12, 26)

	shard1Service := newTestService(kvStore, 1, 10)
	shard1InsertedKeys := insertKeys(ctx, shard1Service, accountA, 5, 21)

	shard2Service := newTestService(kvStore, 2, 10)
	shard2InsertedKeys := insertKeys(ctx, shard2Service, accountA, 21, 1)

	deletedKeys := [][]byte{}
	expectDeleteKeys := shard1InsertedKeys
	expectDeleteKeys = append(expectDeleteKeys, shard2InsertedKeys...)

	batchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
		deletedKeys = append(deletedKeys, keys...)
		kvStore.BatchDelete(ctx, keys)
	}

	shard0Service.purgeAccountAboveShard(context.Background(), accountA, byte(0))
	assert.Equal(t, expectDeleteKeys, deletedKeys)
	_, err := shard0Service.shardNewestSequenceData(context.Background(), accountA, 1, shard0Service.processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
	_, err = shard0Service.shardNewestSequenceData(context.Background(), accountA, 2, shard0Service.processSequenceDataKeyValue)
	assert.Error(t, err, store.ErrNotFound)
}

func insertKeys(ctx context.Context, s *Service, account uint64, keyCount int, sequenceNumber uint64) [][]byte {
	revOrderInsertKeys := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		acctSeqData := SequenceData{CurrentOrdinal: uint64(i + 1), LastGlobalSeq: (sequenceNumber + 1)}
		revOrderInsertKeys[keyCount-1-i] = encodeActionKey(account, s.shardNum, acctSeqData.CurrentOrdinal)
		s.writeAction(ctx, account, acctSeqData, &pbcodec.ActionTrace{}, []byte{})
	}
	s.forceFlush(ctx)
	return revOrderInsertKeys
}
