package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.createtier", &OpCreateTier{})
}

type OpCreateTier struct {
	Tier   eos.AccountName `json:"tier"`
	Perm   uint64          `json:"max_free_permission_objects"`
	Shared uint64          `json:"max_free_shared_keys"`
	Level  uint64          `json:"max_free_permission_levels"`
	Waits  uint64          `json:"max_free_waits"`
	Object uint64          `json:"max_free_permission_link_objects"`
}

func (op *OpCreateTier) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewCreatetier(op.Tier, op.Perm, op.Shared, op.Level, op.Waits, op.Object))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpCreateTier) RequireValidation() bool {
	return true
}
