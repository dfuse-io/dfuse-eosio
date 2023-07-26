package cache

import (
	"time"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/bstream"
)

type Cache interface {
	Tokens() []*pbtokenmeta.Token
	IsTokenContract(contract eos.AccountName) bool
	TokenContract(contract eos.AccountName, code eos.SymbolCode) *pbtokenmeta.Token
	AccountBalances(account eos.AccountName, opts ...AccountBalanceOption) []*OwnedAsset
	TokenBalances(contract eos.AccountName, opts ...TokenBalanceOption) []*OwnedAsset
	Apply(mutationsBatch *MutationsBatch, processedBlock bstream.BlockRef) []error
	SaveToFile() error
	AtBlockRef() bstream.BlockRef
	SetHeadBlockTime(t time.Time)
	GetHeadBlockTime() time.Time
}

const EOSTokenContract = eos.AccountName("eosio.token")

type SortingOrder int32

const (
	ASC SortingOrder = iota
	DESC
)

type TokenBalanceOption int

const (
	EOSIncludeStakedTokOpt TokenBalanceOption = iota
)

type AccountBalanceOption int

const (
	EOSIncludeStakedAccOpt AccountBalanceOption = iota
)
