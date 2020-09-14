package accounthist

import (
	"fmt"
	"math"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

	"github.com/dfuse-io/kvdb/store"
)

type AccountKey uint64

func AccountKeyActionGate(act *pbcodec.ActionTrace) bool {
	// allow all actions to pass
	return true
}

func NewAccountKey(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) ActionKey {
	return AccountKey(account)
}

func AccountKeyRowDecoder(key []byte) (ActionKey, byte, uint64) {
	account, shard, seqNum := keyer.DecodeAccountKeySeqNum(key)
	return AccountKey(account), shard, seqNum
}

func (a AccountKey) Range(shard byte) (RowKey, RowKey) {
	startKey := keyer.EncodeAccountKey(uint64(a), shard, math.MaxUint64)
	endKey := store.Key(keyer.EncodeAccountKey(uint64(a), shard, 0)).PrefixNext()
	return startKey, RowKey(endKey)
}

func (a AccountKey) String() string {
	return fmt.Sprintf("%d", a)
}

func (a AccountKey) Account() uint64 {
	return uint64(a)
}

func (a AccountKey) Row(shard byte, seqData uint64) RowKey {
	return keyer.EncodeAccountKey(uint64(a), shard, seqData)
}
