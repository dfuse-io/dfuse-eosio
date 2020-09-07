package accounthist

import (
	"context"
	"testing"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/require"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"

	"github.com/eoscanada/eos-go"
)

func Test_shardNewestSequenceData(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()
	account := eos.MustStringToName("a")

	processSequenceDataKeyValue := func(item store.KV) (SequenceData, error) {
		s := SequenceData{}
		_, _, nextOrdinal := decodeActionKeySeqNum(item.Key)
		s.CurrentOrdinal = nextOrdinal
		return s, nil
	}

	serviceZero := newTestService(kvStore, 0, 10)
	serviceZero.writeAction(ctx, account, SequenceData{
		CurrentOrdinal: 1,
		LastGlobalSeq:  3,
	}, &pbcodec.ActionTrace{}, []byte{})
	serviceZero.forceFlush(ctx)

	serviceTwo := newTestService(kvStore, 2, 10)
	serviceTwo.writeAction(ctx, account, SequenceData{
		CurrentOrdinal: 7,
		LastGlobalSeq:  1,
	}, &pbcodec.ActionTrace{}, []byte{})
	serviceTwo.forceFlush(ctx)

	seq, err := serviceZero.shardNewestSequenceData(ctx, account, 0, processSequenceDataKeyValue)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), seq.CurrentOrdinal)

	_, err = serviceZero.shardNewestSequenceData(ctx, account, 1, processSequenceDataKeyValue)
	require.Error(t, store.ErrNotFound)

	seq, err = serviceZero.shardNewestSequenceData(ctx, account, 2, processSequenceDataKeyValue)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), seq.CurrentOrdinal)

}
