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
		kvStore:              kvStore,
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001a", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:some:thing")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:some:thing")),
		),
	)

	results := listActions(t, s, "some1", nil)
	require.Len(t, results, 1)

	assert.Equal(t, "some1:00:0", results[0].StringCursor())
	assert.Equal(t, ct.ActionTrace(t, "some1:some:thing", ct.GlobalSequence(1)), results[0].actionTrace)
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

// type testKV struct {
// 	store.KVStore
// 	puts    []kv
// 	deletes [][]byte

// 	scans    []*scanCall
// 	prefixes []*prefixCall
// 	flushes  int
// }

// type kv struct {
// 	key, value []byte
// }

// type scanCall struct {
// 	startKey []byte
// 	endKey   []byte
// 	result   []store.KV
// }

// type prefixCall struct {
// 	prefix []byte
// 	result []store.KV
// }

// func (m *testKV) Put(ctx context.Context, key, value []byte) error {
// 	m.puts = append(m.puts, kv{key, value})
// 	return nil
// }

// func (m *testKV) TestScan(result []store.KV) *scanCall {
// 	call := &scanCall{
// 		result: result,
// 	}
// 	m.scans = append(m.scans, call)
// 	return call
// }

// func (m *testKV) Scan(ctx context.Context, start, end []byte, limit int) *store.Iterator {
// 	fmt.Println("CALLING SCAN")
// 	call := m.scans[0]
// 	m.scans = m.scans[1:]

// 	call.startKey = start
// 	call.endKey = end

// 	it := store.NewIterator(ctx)
// 	go func() {
// 		for _, res := range call.result {
// 			it.PushItem(res)
// 		}
// 		it.PushFinished()
// 	}()
// 	return it
// }

// func (m *testKV) TestPrefix(result []store.KV) *prefixCall {
// 	call := &prefixCall{
// 		result: result,
// 	}
// 	m.prefixes = append(m.prefixes, call)
// 	return call
// }

// func (m *testKV) Prefix(ctx context.Context, prefix []byte, limit int) *store.Iterator {
// 	fmt.Println("CALL PREFIX")
// 	call := m.prefixes[0]
// 	m.prefixes = m.prefixes[1:]

// 	call.prefix = prefix

// 	it := store.NewIterator(ctx)
// 	go func() {
// 		for _, res := range call.result {
// 			it.PushItem(res)
// 		}
// 		it.PushFinished()
// 	}()
// 	return it
// }

// func (m *testKV) DeleteBatch(ctx context.Context, keys [][]byte) error {
// 	for _, key := range keys {
// 		m.deletes = append(m.deletes, key)
// 	}
// 	return nil
// }

// func (m *testKV) FlushPuts(ctx context.Context) error {
// 	m.flushes++
// 	return nil
// }
