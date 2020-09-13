package accounthist

import (
	"encoding/hex"
	"time"

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

type CheckpointKeyEncoderFunc func(shardNum byte) []byte
type KeyEncoderFunc func(blk *bstream.Block, act *pbcodec.ActionTrace, account uint64) ActionKey
type RowKeyDecoderFunc func(key []byte) (ActionKey, byte, uint64)

type ActionKey interface {
	String() string
	Account() uint64
	Row(shard byte, seqData uint64) RowKey
	Range(shard byte) (startKey RowKey, endKey RowKey)
}

type RowKey []byte

func (k RowKey) String() string {
	return hex.EncodeToString(k)
}
