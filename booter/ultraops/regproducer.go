package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.regproducer", &OpRegproducer{})
}

type OpRegproducer struct {
	Producer    eos.AccountName `json:"producer"`
	ProducerKey string          `json:"producer_key"`
	URL         string          `json:"url"`
	Location    uint16          `json:"location"`
}

func (op *OpRegproducer) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	pubKey, err := decodeOpPublicKey(c, op.ProducerKey)
	if err != nil {
		return err
	}
	in <- (*bootops.TransactionAction)(system.NewRegProducer(op.Producer, pubKey, op.URL, op.Location))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpRegproducer) RequireValidation() bool {
	return true
}
