package accounthist

import (
	"encoding/hex"
	"math"
	"time"

	"github.com/dfuse-io/kvdb/store"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

const (
	DatabaseTimeout = 10 * time.Minute
)

type AccounthistMode string

const (
	AccounthistModeAccount         AccounthistMode = "account"
	AccounthistModeAccountContract AccounthistMode = "account-contract"
)

type RowKeyDecoderFunc func(key []byte) (Facet, byte, uint64)

type KeyEncoderFunc func(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) Facet

type FacetFactory interface {
	Collection() byte
	NewFacet(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) Facet
	NewCheckpointKey(shardNum byte) []byte
	DecodeRow(key []byte) (Facet, byte, uint64)
	ActionFilter(act *pbcodec.ActionTrace) bool
}

// facet is the key prefix for virtual tables (i.e. 02:account or 03:account:contract)
type Facet interface {
	String() string
	Bytes() []byte
	Account() uint64
	// TODO: should replace RowKey with store.Key
	Row(shard byte, seqData uint64) RowKey
}

type RowKey []byte

func (k RowKey) String() string {
	return hex.EncodeToString(k)
}

func FacetShardRange(facet Facet, shard byte) (RowKey, RowKey) {
	startKey := facet.Row(shard, math.MaxUint64)
	endKey := store.Key(facet.Row(shard, 0)).PrefixNext()
	return startKey, RowKey(endKey)
}

func FacetRangeLowerBound(facet Facet, lowShardNum byte) (RowKey, RowKey) {
	startKey := facet.Row(lowShardNum, math.MaxUint64)
	endKey := store.Key(facet.Row(0xff, 0)).PrefixNext()
	return startKey, RowKey(endKey)
}
