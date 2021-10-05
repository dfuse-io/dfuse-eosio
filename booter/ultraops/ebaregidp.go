package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	eba "github.com/eoscanada/eos-go/eba"
	"github.com/eoscanada/eos-go/ecc"
)

func init() {
	bootops.Register("eba.regidp", &OpEBARegidp{})
}

type OpEBARegidp struct {
	Providers []providerKeyString `json:"id_providers"`
}

func (op *OpEBARegidp) RequireValidation() bool {
	return true
}

func (op *OpEBARegidp) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	var providers []eba.Provider

	for _, key := range op.Providers {
		provider := eba.Provider{
			Account: key.Account,
		}
		pubKey, err := decodeOpPublicKey(c, key.Key)
		if err != nil {
			return err
		}
		provider.Key = pubKey
		providers = append(providers, provider)
	}

	in <- (*bootops.TransactionAction)(eba.NewRegidp(providers))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

type providerKeyString struct {
	Account eos.AccountName `json:"account"`
	Key     string          `json:"key"`
}
