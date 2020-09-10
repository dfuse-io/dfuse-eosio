package accounthist

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/stretchr/testify/assert"
)

func TestLiveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := &Service{
		shardNum:             0,
		maxEntriesPerAccount: 2,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]SequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:thing")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing")),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "some1:00:1", actionTrace: ct.ActionTrace(t, "some1:some:thing", ct.GlobalSequence(1))},
	}, listActions(t, s, "some1", nil))
}

func TestLiveShard_DeleteWindow(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := &Service{
		shardNum:             0,
		maxEntriesPerAccount: 2,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]SequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

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
		{cursor: "some1:00:3", actionTrace: ct.ActionTrace(t, "some1:some:bing2", ct.GlobalSequence(7))},
		{cursor: "some1:00:2", actionTrace: ct.ActionTrace(t, "some1:some:bing1", ct.GlobalSequence(6))},
	}, listActions(t, s, "some1", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "some2:00:4", actionTrace: ct.ActionTrace(t, "some2:some:thing5", ct.GlobalSequence(5))},
		{cursor: "some2:00:3", actionTrace: ct.ActionTrace(t, "some2:some:thing4", ct.GlobalSequence(4))},
	}, listActions(t, s, "some2", nil))
}
