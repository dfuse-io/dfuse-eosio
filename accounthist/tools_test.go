package accounthist

import (
	"context"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	"github.com/stretchr/testify/assert"
)

func Test_shardSummary(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	maxEntries := uint64(10)
	runShard(t, 0, maxEntries, kvStore,
		ct.Block(t, "00000002bb",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(3))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(4))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing3", ct.GlobalSequence(5))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing4", ct.GlobalSequence(6))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing5", ct.GlobalSequence(7))),
		),
	)

	runShard(t, 1, maxEntries, kvStore,
		ct.Block(t, "00000001aa",
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing1", ct.GlobalSequence(1))),
			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing2", ct.GlobalSequence(2))),
		),
	)

	summary, err := shardSummary(context.Background(), kvStore, eos.MustStringToName("a"), 5)
	require.NoError(t, err)
	assert.Equal(t, summary, []*sequenceData{
		{nextOrdinal: 6, lastGlobalSeq: 7},
		{nextOrdinal: 3, lastGlobalSeq: 2},
	})

}
