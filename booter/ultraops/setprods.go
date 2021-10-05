package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.ultra.setprods", &OpSetprods{})
}

type OpSetprods struct {
	Schedule []eos.AccountName `json:"schedule"`
}

func (op *OpSetprods) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewUltraSetProds(op.Schedule))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpSetprods) RequireValidation() bool {
	return true
}
