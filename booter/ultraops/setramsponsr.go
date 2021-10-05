package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.setramsponsr", &OpSetramsponsr{})
}

type OpSetramsponsr struct {
	RAMSponsor eos.AccountName `json:"ram_sponsor"`
}

func (op *OpSetramsponsr) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewSetramsponsr(op.RAMSponsor))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpSetramsponsr) RequireValidation() bool {
	return true
}
