package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.newactconfig", &OpNewactconfig{})
}

type OpNewactconfig struct {
	Cost                   eos.Asset       `json:"cost"`
	Oracle                 eos.AccountName `json:"oracle"`
	CandidateMovingAverage []eos.Asset     `json:"candicate_moving_average"`
}

func (op *OpNewactconfig) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewNewactconfig(op.Cost, op.Oracle, op.CandidateMovingAverage))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpNewactconfig) RequireValidation() bool {
	return true
}
