package accounthist

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/bstream/forkable"
	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharding(t *testing.T) {
	// ct.Block(t, "00000001aa", ct.FilteredBlock{
	// 	UnfilteredStats: ct.Counts{2, 2, 2},
	// 	FilteredStats:   ct.Counts{1, 1, 1},
	// },
	// 	ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
	// ),

	fObj := &forkable.ForkableObject{Obj: map[uint64][]byte{}, StepIndex: 0, StepCount: 1}

	kv := &testKV{}
	s := &Service{
		shardNum:             1,
		maxEntriesPerAccount: 2,
		flushBlocksInterval:  1,
		kvStore:              kv,
		historySeqMap:        map[uint64]sequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}
	scall1 := kv.TestScan([]store.KV{})
	pcall1 := kv.TestPrefix([]store.KV{})

	scall2 := kv.TestScan([]store.KV{})
	pcall2 := kv.TestPrefix([]store.KV{})

	require.NoError(t, s.ProcessBlock(ct.ToBstreamBlock(t, ct.Block(t, "00000001a",
		ct.TrxTrace(t, ct.ActionTrace(t, "............1:some:thing", ct.GlobalSequence(1))),
		ct.TrxTrace(t, ct.ActionTrace(t, "............2:some:thing", ct.GlobalSequence(1))),
	)), fObj))

	fmt.Println("MAMA", s.historySeqMap)
	assert.Equal(t, uint64(1), s.historySeqMap[1].historySeqNum)
	assert.Equal(t, []byte{
		0x2,
		0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0xff,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, scall1.startKey)
	assert.Equal(t, []byte{
		0x2,
		0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, pcall1.prefix)

	assert.Equal(t, []byte{
		0x2,
		0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0xff,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, scall2.startKey)
	assert.Equal(t, []byte{
		0x2,
		0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}, pcall2.prefix)

	// require.NoError(t, s.ProcessBlock(ct.ToBstreamBlock(t, ct.Block(t, "00000002a",
	// 	ct.TrxTrace(t, ct.ActionTrace(t, "1:some:thing")),
	// 	ct.TrxTrace(t, ct.ActionTrace(t, "2:some:thing")),
	// )), fObj))

	// process a shard 0
	// two blocks
	// two action traces on two different accounts per block
	// maxEntries = 3
	// process shard 1
}

type testKV struct {
	store.KVStore
	puts    []kv
	deletes [][]byte

	scans    []*scanCall
	prefixes []*prefixCall
	flushes  int
}

type kv struct {
	key, value []byte
}

type scanCall struct {
	startKey []byte
	endKey   []byte
	result   []store.KV
}

type prefixCall struct {
	prefix []byte
	result []store.KV
}

func (m *testKV) Put(ctx context.Context, key, value []byte) error {
	m.puts = append(m.puts, kv{key, value})
	return nil
}

func (m *testKV) TestScan(result []store.KV) *scanCall {
	call := &scanCall{
		result: result,
	}
	m.scans = append(m.scans, call)
	return call
}

func (m *testKV) Scan(ctx context.Context, start, end []byte, limit int) *store.Iterator {
	fmt.Println("CALLING SCAN")
	call := m.scans[0]
	m.scans = m.scans[1:]

	call.startKey = start
	call.endKey = end

	it := store.NewIterator(ctx)
	go func() {
		for _, res := range call.result {
			it.PushItem(res)
		}
		it.PushFinished()
	}()
	return it
}

func (m *testKV) TestPrefix(result []store.KV) *prefixCall {
	call := &prefixCall{
		result: result,
	}
	m.prefixes = append(m.prefixes, call)
	return call
}

func (m *testKV) Prefix(ctx context.Context, prefix []byte, limit int) *store.Iterator {
	fmt.Println("CALL PREFIX")
	call := m.prefixes[0]
	m.prefixes = m.prefixes[1:]

	call.prefix = prefix

	it := store.NewIterator(ctx)
	go func() {
		for _, res := range call.result {
			it.PushItem(res)
		}
		it.PushFinished()
	}()
	return it
}

func (m *testKV) DeleteBatch(ctx context.Context, keys [][]byte) error {
	for _, key := range keys {
		m.deletes = append(m.deletes, key)
	}
	return nil
}

func (m *testKV) FlushPuts(ctx context.Context) error {
	m.flushes++
	return nil
}
