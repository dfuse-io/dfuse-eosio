package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.setramcurve", &OpSetramcurve{})
}

type OpSetramcurve struct {
	CoreReserve     eos.Asset `json:"core_reserve"`
	RamSupply       int64     `json:"ram_supply"`
	ConnectorWeight float64   `json:"connector_weight"`
}

func (op *OpSetramcurve) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewSetramcurve(op.CoreReserve, op.RamSupply, op.ConnectorWeight))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpSetramcurve) RequireValidation() bool {
	return true
}
