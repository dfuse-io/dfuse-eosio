package ultraops

import (
	"fmt"

	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	token "github.com/eoscanada/eos-go/token"
)

func init() {
	bootops.Register("token.open", &OpTokenOpen{})
}

type OpTokenOpen struct {
	Owner    eos.AccountName `json:"owner"`
	Symbol   string          `json:"symbol"`
	RamPayer eos.AccountName `json:"ram_payer"`
}

func (op *OpTokenOpen) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	symbol, err := eos.StringToSymbol(op.Symbol)
	if err != nil {
		return fmt.Errorf("unable to convert system.init core %q to symbol: %w", op.Symbol, err)
	}

	in <- (*bootops.TransactionAction)(token.NewOpen(op.Owner, symbol, op.RamPayer))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpTokenOpen) RequireValidation() bool {
	return true
}
