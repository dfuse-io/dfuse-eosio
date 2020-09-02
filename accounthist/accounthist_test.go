package accounthist

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/bstream/forkable"
	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveShard(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := &Service{
		shardNum:             0,
		maxEntriesPerAccount: 2,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:thing")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing")),
		),
	)

	results := listActions(t, s, "some1", nil)
	require.Len(t, results, 1)

	assert.Equal(t, "some1:00:1", results[0].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some1:some:thing", ct.GlobalSequence(1)), results[0].actionTrace)
}

func TestLiveShard_DeleteWindow(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := &Service{
		shardNum:             0,
		maxEntriesPerAccount: 2,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]sequenceData{},
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

	results := listActions(t, s, "some1", nil)
	require.Len(t, results, 2)

	assert.Equal(t, "some1:00:3", results[0].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some1:some:bing2", ct.GlobalSequence(7)), results[0].actionTrace)

	assert.Equal(t, "some1:00:2", results[1].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some1:some:bing1", ct.GlobalSequence(6)), results[1].actionTrace)

	results = listActions(t, s, "some2", nil)
	require.Len(t, results, 2)

	assert.Equal(t, "some2:00:4", results[0].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some2:some:thing5", ct.GlobalSequence(5)), results[0].actionTrace)

	assert.Equal(t, "some2:00:3", results[1].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some2:some:thing4", ct.GlobalSequence(4)), results[1].actionTrace)
}

func streamBlocks(t *testing.T, s *Service, blocks ...*pbcodec.Block) {
	preprocessor := preprocessingFunc(s.blockFilter)

	for _, block := range blocks {
		blk := ct.ToBstreamBlock(t, block)
		obj, err := preprocessor(blk)
		require.NoError(t, err)

		s.ProcessBlock(blk, &forkable.ForkableObject{Obj: obj})
	}
}

type actionResult struct {
	cursor      *pbaccounthist.Cursor
	actionTrace *pbcodec.ActionTrace
}

func (r *actionResult) StringCursor() string {
	return fmt.Sprintf("%s:%02x:%d", eos.NameToString(r.cursor.Account), byte(r.cursor.ShardNum), r.cursor.SequenceNumber)
}

func listActions(t *testing.T, s *Service, account string, cursor *pbaccounthist.Cursor) (out []*actionResult) {
	ctx := context.Background()

	err := s.StreamActions(ctx, eos.MustStringToName(account), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		out = append(out, &actionResult{cursor, actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}
