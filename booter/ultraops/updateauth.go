package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
)

func init() {
	bootops.Register("system.updateauth", &OpUpdateAuth{})
}

type OpUpdateAuth struct {
	Account         eos.AccountName    `json:"account"`
	Permission      eos.PermissionName `json:"permission"`
	Parent          eos.PermissionName `json:"parent"`
	Auth            eos.Authority      `json:"auth"`
	UsingPermission eos.PermissionName `json:"using_permission"`
}

func (op *OpUpdateAuth) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(system.NewUpdateAuth(op.Account, op.Permission, op.Parent, op.Auth, op.UsingPermission))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpUpdateAuth) RequireValidation() bool {
	return true
}
