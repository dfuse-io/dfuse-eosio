package injector

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	"github.com/stretchr/testify/assert"
)

func Test_AccountLiveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := setupAccountInjector(NewRWCache(kvStore), 0, 2)

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:thing1")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing2")),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "02c524a0800000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "some1:some:thing1", ct.GlobalSequence(1))},
	}, listAccountActions(t, s, "some1", nil))
}

func Test_AccountLiveShard_DeleteWindow(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := setupAccountInjector(NewRWCache(kvStore), 0, 2)

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:thing1")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing2")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing3")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing4")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing5")),
		),

		ct.Block(t, "00000002aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:bing1")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:bing2")),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "02c524a0800000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "some1:some:bing2", ct.GlobalSequence(7))},
		{cursor: "02c524a0800000000000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "some1:some:bing1", ct.GlobalSequence(6))},
	}, listAccountActions(t, s, "some1", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "02c524a1000000000000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "some2:some:thing5", ct.GlobalSequence(5))},
		{cursor: "02c524a1000000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "some2:some:thing4", ct.GlobalSequence(4))},
	}, listAccountActions(t, s, "some2", nil))
}

func Test_Account_Sharding(t *testing.T) {
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
		{cursor: "02300000000000000000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "a:some:ething2", ct.GlobalSequence(18))},
		{cursor: "02300000000000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "a:some:ething1", ct.GlobalSequence(17))},
		{cursor: "02300000000000000000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(13))},
		{cursor: "02300000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(12))},
	}, listAccountActions(t, shardZero, "a", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "02380000000000000000fffffffffffffffa:00:5", actionTrace: ct.ActionTrace(t, "b:some:ething4", ct.GlobalSequence(20))},
		{cursor: "02380000000000000000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "b:some:ething3", ct.GlobalSequence(19))},
		{cursor: "02380000000000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "b:some:dthing5", ct.GlobalSequence(16))},
		{cursor: "02380000000000000000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(15))},
		{cursor: "02380000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(14))},
	}, listAccountActions(t, shardZero, "b", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "02400000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "c:some:ething5", ct.GlobalSequence(21))},
	}, listAccountActions(t, shardZero, "c", nil))

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
		{cursor: "02300000000000000000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "a:some:ething2", ct.GlobalSequence(18))},
		{cursor: "02300000000000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "a:some:ething1", ct.GlobalSequence(17))},
		{cursor: "02300000000000000000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "a:some:dthing2", ct.GlobalSequence(13))},
		{cursor: "02300000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "a:some:dthing1", ct.GlobalSequence(12))},
		{cursor: "02300000000000000001fffffffffffffffc:01:3", actionTrace: ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(7))},
	}, listAccountActions(t, shardOne, "a", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "02380000000000000000fffffffffffffffa:00:5", actionTrace: ct.ActionTrace(t, "b:some:ething4", ct.GlobalSequence(20))},
		{cursor: "02380000000000000000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "b:some:ething3", ct.GlobalSequence(19))},
		{cursor: "02380000000000000000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "b:some:dthing5", ct.GlobalSequence(16))},
		{cursor: "02380000000000000000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "b:some:dthing4", ct.GlobalSequence(15))},
		{cursor: "02380000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "b:some:dthing3", ct.GlobalSequence(14))},
	}, listAccountActions(t, shardOne, "b", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "02400000000000000000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "c:some:ething5", ct.GlobalSequence(21))},
		{cursor: "02400000000000000001fffffffffffffff9:01:6", actionTrace: ct.ActionTrace(t, "c:some:cthing5", ct.GlobalSequence(11))},
		{cursor: "02400000000000000001fffffffffffffffa:01:5", actionTrace: ct.ActionTrace(t, "c:some:cthing4", ct.GlobalSequence(10))},
		{cursor: "02400000000000000001fffffffffffffffb:01:4", actionTrace: ct.ActionTrace(t, "c:some:cthing3", ct.GlobalSequence(9))},
		{cursor: "02400000000000000001fffffffffffffffc:01:3", actionTrace: ct.ActionTrace(t, "c:some:cthing2", ct.GlobalSequence(8))},
	}, listAccountActions(t, shardOne, "c", nil))

	shardTwo := runShard(t, 2, maxEntries, kvStore,
		ct.Block(t, "00000001aa",
			ct.TrxTrace(t, ct.ActionTrace(t, "d:some:athing1", ct.GlobalSequence(1))),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "02480000000000000002fffffffffffffffe:02:1", actionTrace: ct.ActionTrace(t, "d:some:athing1", ct.GlobalSequence(1))},
	}, listAccountActions(t, shardTwo, "d", nil))

}

//func TestShardingMaxEntries(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	maxEntries := uint64(5)
//	shardZero := runShard(t, 0, maxEntries, kvStore,
//		ct.Block(t, "00000003cc",
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(3))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(4))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing3", ct.GlobalSequence(5))),accounthist/app/accounthist/app.go
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
//	}, listAccountActions(t, shardZero, "a", nil))
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
