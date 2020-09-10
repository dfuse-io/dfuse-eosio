package accounthist

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	"github.com/stretchr/testify/assert"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb/store"
)

func TestSharding(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	maxEntries := uint64(5)
	shardZero := runShard(t, 0, maxEntries, kvStore,
		ct.Block(t, "00000004dd",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(12))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(13))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(14))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(15))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:dthing5", ct.GlobalSequence(16))),
		),
		ct.Block(t, "00000005ee",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:ething1", ct.GlobalSequence(17))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:ething2", ct.GlobalSequence(18))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:ething3", ct.GlobalSequence(19))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:ething4", ct.GlobalSequence(20))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:ething5", ct.GlobalSequence(21))),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "a:00:4", actionTrace: ct.ActionTrace(t, "a:some:ething2", ct.GlobalSequence(18))},
		{cursor: "a:00:3", actionTrace: ct.ActionTrace(t, "a:some:ething1", ct.GlobalSequence(17))},
		{cursor: "a:00:2", actionTrace: ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(13))},
		{cursor: "a:00:1", actionTrace: ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(12))},
	}, listActions(t, shardZero, "a", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "b:00:5", actionTrace: ct.ActionTrace(t, "b:some:ething4", ct.GlobalSequence(20))},
		{cursor: "b:00:4", actionTrace: ct.ActionTrace(t, "b:some:ething3", ct.GlobalSequence(19))},
		{cursor: "b:00:3", actionTrace: ct.ActionTrace(t, "b:some:dthing5", ct.GlobalSequence(16))},
		{cursor: "b:00:2", actionTrace: ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(15))},
		{cursor: "b:00:1", actionTrace: ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(14))},
	}, listActions(t, shardZero, "b", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "c:00:1", actionTrace: ct.ActionTrace(t, "c:some:ething5", ct.GlobalSequence(21))},
	}, listActions(t, shardZero, "c", nil))

	shardOne := runShard(t, 1, maxEntries, kvStore,
		ct.Block(t, "00000002bb",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:bthing1", ct.GlobalSequence(2))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:bthing2", ct.GlobalSequence(3))),
			ct.TrxTrace(t, ct.ActionTrace(t, "b:some:bthing3", ct.GlobalSequence(4))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing4", ct.GlobalSequence(5))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:bthing5", ct.GlobalSequence(6))),
		),
		ct.Block(t, "00000003cc",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(7))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:cthing2", ct.GlobalSequence(8))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:cthing3", ct.GlobalSequence(9))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:cthing4", ct.GlobalSequence(10))),
			ct.TrxTrace(t, ct.ActionTrace(t, "c:some:cthing5", ct.GlobalSequence(11))),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "a:00:4", actionTrace: ct.ActionTrace(t, "a:some:ething2", ct.GlobalSequence(18))},
		{cursor: "a:00:3", actionTrace: ct.ActionTrace(t, "a:some:ething1", ct.GlobalSequence(17))},
		{cursor: "a:00:2", actionTrace: ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(13))},
		{cursor: "a:00:1", actionTrace: ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(12))},
		{cursor: "a:01:3", actionTrace: ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(7))},
	}, listActions(t, shardOne, "a", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "b:00:5", actionTrace: ct.ActionTrace(t, "b:some:ething4", ct.GlobalSequence(20))},
		{cursor: "b:00:4", actionTrace: ct.ActionTrace(t, "b:some:ething3", ct.GlobalSequence(19))},
		{cursor: "b:00:3", actionTrace: ct.ActionTrace(t, "b:some:dthing5", ct.GlobalSequence(16))},
		{cursor: "b:00:2", actionTrace: ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(15))},
		{cursor: "b:00:1", actionTrace: ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(14))},
	}, listActions(t, shardOne, "b", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "c:00:1", actionTrace: ct.ActionTrace(t, "c:some:ething5", ct.GlobalSequence(21))},
		{cursor: "c:01:6", actionTrace: ct.ActionTrace(t, "c:some:cthing5", ct.GlobalSequence(11))},
		{cursor: "c:01:5", actionTrace: ct.ActionTrace(t, "c:some:cthing4", ct.GlobalSequence(10))},
		{cursor: "c:01:4", actionTrace: ct.ActionTrace(t, "c:some:cthing3", ct.GlobalSequence(9))},
		{cursor: "c:01:3", actionTrace: ct.ActionTrace(t, "c:some:cthing2", ct.GlobalSequence(8))},
	}, listActions(t, shardOne, "c", nil))

	shardTwo := runShard(t, 2, maxEntries, kvStore,
		ct.Block(t, "00000001aa",
			ct.TrxTrace(t, ct.ActionTrace(t, "d:some:athing1", ct.GlobalSequence(1))),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "d:02:1", actionTrace: ct.ActionTrace(t, "d:some:athing1", ct.GlobalSequence(1))},
	}, listActions(t, shardTwo, "d", nil))

}

//func TestShardingMaxEntries(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	maxEntries := uint64(5)
//	shardZero := runShard(t, 0, maxEntries, kvStore,
//		ct.Block(t, "00000003cc",
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(3))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(4))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing3", ct.GlobalSequence(5))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing4", ct.GlobalSequence(6))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing5", ct.GlobalSequence(7))),
//		),
//	)
//
//	assert.Equal(t, []*actionResult{
//		{cursor: "a:00:5", actionTrace: ct.ActionTrace(t, "a:some:cthing5", ct.GlobalSequence(7))},
//		{cursor: "a:00:4", actionTrace: ct.ActionTrace(t, "a:some:cthing4", ct.GlobalSequence(6))},
//		{cursor: "a:00:3", actionTrace: ct.ActionTrace(t, "a:some:cthing3", ct.GlobalSequence(5))},
//		{cursor: "a:00:2", actionTrace: ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(4))},
//		{cursor: "a:00:1", actionTrace: ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(3))},
//	}, listActions(t, shardZero, "a", nil))
//
//	service := runShard(t, 1, maxEntries, kvStore,
//		ct.Block(t, "00000001aa",
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing1", ct.GlobalSequence(1))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing2", ct.GlobalSequence(2))),
//		),
//	)
//
//	_, err := service.shardNewestSequenceData(context.Background(), eos.MustStringToName("a"), 1, service.processSequenceDataKeyValue)
//	assert.Error(t, err, store.ErrNotFound)
//}

func runShard(t *testing.T, shardNum byte, maxEntriesPerAccount uint64, kvStore store.KVStore, blocks ...*pbcodec.Block) *Service {
	s := &Service{
		shardNum:             shardNum,
		maxEntriesPerAccount: maxEntriesPerAccount,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]SequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	streamBlocks(t, s, blocks...)
	return s
}
