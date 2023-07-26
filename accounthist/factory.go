package accounthist

import (
	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/bstream"
)

type AccountFactory struct {
}

func (f *AccountFactory) Collection() byte {
	return keyer.PrefixAccount
}

func (f *AccountFactory) NewFacet(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) Facet {
	return AccountFacet(account)
}

func (f *AccountFactory) NewCheckpointKey(shardNum byte) []byte {
	return keyer.EncodeAccountCheckpointKey(shardNum)
}

func (f *AccountFactory) DecodeRow(key []byte) (Facet, byte, uint64) {
	account, shard, seqNum := keyer.DecodeAccountKeySeqNum(key)
	return AccountFacet(account), shard, seqNum
}

func (f *AccountFactory) ActionFilter(act *pbcodec.ActionTrace) bool {
	// allow all actions to pass
	return true
}

type AccountContractFactory struct {
}

func (f *AccountContractFactory) Collection() byte {
	return keyer.PrefixAccountContract
}

func (f *AccountContractFactory) NewFacet(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) Facet {
	contractUint := eos.MustStringToName(act.Action.Account)
	return &AccountContractKey{
		account:  account,
		contract: contractUint,
	}
}

func (f *AccountContractFactory) NewCheckpointKey(shardNum byte) []byte {
	return keyer.EncodeAccountContractCheckpointKey(shardNum)
}

func (f *AccountContractFactory) DecodeRow(key []byte) (Facet, byte, uint64) {
	account, contract, shard, seqNum := keyer.DecodeAccountContractKeySeqNum(key)
	return &AccountContractKey{account, contract}, shard, seqNum
}

func (f *AccountContractFactory) ActionFilter(act *pbcodec.ActionTrace) bool {
	return (act.Action.Name == "transfer")
}
