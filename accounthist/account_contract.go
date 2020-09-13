package accounthist

import (
	"fmt"
	"math"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	"github.com/dfuse-io/kvdb/store"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
)

type AccountContractKey struct {
	account  uint64
	contract uint64
}

func NewAccountContractKey(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) ActionKey {
	contractUint := eos.MustStringToName(act.Action.Account)
	return &AccountContractKey{
		account:  account,
		contract: contractUint,
	}
}

func AccountContractKeyRowDecoder(key []byte) (ActionKey, byte, uint64) {
	account, contract, shard, seqNum := keyer.DecodeAccountContractKeySeqNum(key)
	return &AccountContractKey{account, contract}, shard, seqNum
}

func (a *AccountContractKey) Range(shard byte) (RowKey, RowKey) {
	startKey := keyer.EncodeAccountContractKey(a.account, a.contract, shard, math.MaxUint64)
	endKey := store.Key(keyer.EncodeAccountContractKey(a.account, a.contract, shard, 0)).PrefixNext()
	return startKey, RowKey(endKey)
}

func (a *AccountContractKey) Row(shard byte, seqData uint64) RowKey {
	return keyer.EncodeAccountContractKey(a.account, a.contract, shard, seqData)
}

func (a *AccountContractKey) String() string {
	return fmt.Sprintf("%d:%d", a.account, a.contract)
}

func (a *AccountContractKey) Account() uint64 {
	return a.account
}
