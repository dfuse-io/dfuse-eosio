package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.setdeftier", &OpSetdeftier{})
}

type OpSetdeftier struct {
	Tier eos.AccountName `json:"tier"`
}

func (op *OpSetdeftier) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewSetdeftier(op.Tier))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpSetdeftier) RequireValidation() bool {
	return true
}
