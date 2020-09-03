package accounthist

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"

	"github.com/eoscanada/eos-go"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
)

func Test_scanAccount(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	maxEntries := uint64(10)
	runShard(t, 0, maxEntries, kvStore,
		ct.Block(t, "00000003cc",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(11))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(12))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:cthing3", ct.GlobalSequence(13))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:cthing4", ct.GlobalSequence(14))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:cthing5", ct.GlobalSequence(15))),
		),
		ct.Block(t, "00000004dd",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(16))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(17))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(18))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(19))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:dthing5", ct.GlobalSequence(20))),
		),
	)

	runShard(t, 1, maxEntries, kvStore,
		ct.Block(t, "00000001aa",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing1", ct.GlobalSequence(1))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing2", ct.GlobalSequence(2))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:athing3", ct.GlobalSequence(3))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:athing4", ct.GlobalSequence(4))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:athing5", ct.GlobalSequence(5))),
		),
		ct.Block(t, "00000002bb",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:bthing1", ct.GlobalSequence(6))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing2", ct.GlobalSequence(7))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing3", ct.GlobalSequence(8))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing4", ct.GlobalSequence(9))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing5", ct.GlobalSequence(10))),
		),
	)

	purgeAccounts(context.Background(), kvStore, 100)
}

//func Test_purgeAccountAboveShard(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	ctx := context.Background()
//	shardZero := &Service{
//		shardNum:             0,
//		maxEntriesPerAccount: 10,
//		flushBlocksInterval:  1,
//		kvStore:              NewRWCache(kvStore),
//		historySeqMap:        map[uint64]sequenceData{},
//		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
//	}
//
//	accountA := eos.MustStringToName("a")
//	accountb := eos.MustStringToName("b")
//	accountc := eos.MustStringToName("c")
//	accountd := eos.MustStringToName("d")
//
//	insertKeys(ctx, shardZero, accountA, 12, 64)
//	insertKeys(ctx, shardZero, accountb, 23, 76)
//	insertKeys(ctx, shardZero, accountd, 7, 99)
//
//	shardOne := &Service{
//		shardNum:             1,
//		maxEntriesPerAccount: 10,
//		flushBlocksInterval:  1,
//		kvStore:              NewRWCache(kvStore),
//		historySeqMap:        map[uint64]sequenceData{},
//		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
//	}
//
//	insertKeys(ctx, shardOne, accountA, 5, 49)
//	insertKeys(ctx, shardOne, accountb, 3, 54)
//	insertKeys(ctx, shardOne, accountc, 7, 57)
//
//	shardTwo := &Service{
//		shardNum:             2,
//		maxEntriesPerAccount: 10,
//		flushBlocksInterval:  1,
//		kvStore:              NewRWCache(kvStore),
//		historySeqMap:        map[uint64]sequenceData{},
//		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
//	}
//
//	insertKeys(ctx, shardTwo, accountA, 21, 1)
//	insertKeys(ctx, shardTwo, accountb, 7, 21)
//	insertKeys(ctx, shardTwo, accountc, 21, 28)
//
//	expectedKeys := [][]byte{}
//	batchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
//		expectedKeys = append(expectedKeys, )
//	}
//
//	shardZero.purgeAccountAboveShard(context.Background(), accountA, byte(0))
//	_, err := shardLastSequenceData(context.Background(), kvStore, accountA, 1)
//	assert.Error(t, err, store.ErrNotFound)
//	_, err = shardLastSequenceData(context.Background(), kvStore, accountA, 2)
//	assert.Error(t, err, store.ErrNotFound)
//}

func Test_purgeAccountAboveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()
	shardZero := &Service{
		shardNum:             0,
		maxEntriesPerAccount: 10,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	accountA := eos.MustStringToName("a")

	insertKeys(ctx, shardZero, accountA, 12, 26)

	shardOne := &Service{
		shardNum:             1,
		maxEntriesPerAccount: 10,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	shardOneKeys := insertKeys(ctx, shardOne, accountA, 5, 21)

	shardTwo := &Service{
		shardNum:             2,
		maxEntriesPerAccount: 10,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	shardTwoKeys := insertKeys(ctx, shardTwo, accountA, 21, 1)

	expectedKeys := [][]byte{}

	batchDeleteKeys = func(ctx context.Context, kvStore store.KVStore, keys [][]byte) {
		expectedKeys = append(expectedKeys, keys...)
		kvStore.BatchDelete(ctx, keys)
	}

	shardZero.purgeAccountAboveShard(context.Background(), accountA, byte(0))

	for i := 0; i < len(expectedKeys); i++ {
		zlog.Info("key deleted", zap.Stringer("key", Key(expectedKeys[i])))
	}

	for i := 0; i < len(shardOneKeys); i++ {
		zlog.Info("key deleted", zap.Stringer("key", Key(shardOneKeys[i])))
	}

	for i := 0; i < len(shardTwoKeys); i++ {
		zlog.Info("key deleted", zap.Stringer("key", Key(shardTwoKeys[i])))
	}

	_, err := shardLastSequenceData(context.Background(), kvStore, accountA, 1)
	assert.Error(t, err, store.ErrNotFound)
	_, err = shardLastSequenceData(context.Background(), kvStore, accountA, 2)
	assert.Error(t, err, store.ErrNotFound)
}

func insertKeys(ctx context.Context, s *Service, account uint64, keyCount int, sequenceNumber uint64) [][]byte {
	revOrderInsertKeys := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		acctSeqData := sequenceData{nextOrdinal: uint64(i + 1), lastGlobalSeq: (sequenceNumber + 1)}
		revOrderInsertKeys[keyCount-1-i] = encodeActionKey(account, s.shardNum, acctSeqData.nextOrdinal)
		s.writeAction(ctx, account, acctSeqData, &pbcodec.ActionTrace{}, []byte{})
	}
	s.kvStore.FlushPuts(ctx)
	return revOrderInsertKeys
}
