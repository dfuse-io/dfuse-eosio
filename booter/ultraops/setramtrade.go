package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.setramtrade", &OpSetramtrade{})
}

type OpSetramtrade struct {
	State bool `json:"state"`
}

func (op *OpSetramtrade) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewSetramtrade(op.State))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpSetramtrade) RequireValidation() bool {
	return true
}
