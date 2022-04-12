package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
)

func NewOpLinkAuth(account, code eos.AccountName, actionName eos.ActionName, requiredPermission eos.PermissionName) *eos.Action {
	a := &eos.Action{
		Account: AN("eosio"),
		Name:    ActN("linkauth"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      account,
				Permission: eos.PermissionName("owner"),
			},
		},
		ActionData: eos.NewActionData(OpLinkAuth{
			Account:     account,
			Code:        code,
			Type:        actionName,
			Requirement: requiredPermission,
		}),
	}

	return a
}

func init() {
	bootops.Register("system.linkauth", &OpLinkAuth{})
}

type OpLinkAuth struct {
	Account     eos.AccountName    `json:"account"`
	Code        eos.AccountName    `json:"code"`
	Type        eos.ActionName     `json:"type"`
	Requirement eos.PermissionName `json:"requirement"`
}

func (op *OpLinkAuth) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(NewOpLinkAuth(op.Account, op.Code, op.Type, op.Requirement))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpLinkAuth) RequireValidation() bool {
	return true
}
