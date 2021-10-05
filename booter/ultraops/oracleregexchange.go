package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	oracle "github.com/eoscanada/eos-go/oracle"
)

func init() {
	bootops.Register("oracle.regexchange", &OpOracleRegexchange{})
}

type OpOracleRegexchange struct {
	Exchange eos.AccountName `json:"exchange"`
}

func (op *OpOracleRegexchange) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(oracle.NewRegexchange(op.Exchange))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpOracleRegexchange) RequireValidation() bool {
	return true
}
