package migrator

import (
	"encoding/json"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

	"github.com/eoscanada/eos-go"
)

type DetailedTableRow struct {
	tableRow

	account eos.AccountName
	table   eos.TableName
	scope   eos.ScopeName
}

//account dfuse.boot setCode wasm.contract
type tableRow struct {
	Key   string          `json:"key"`
	Payer string          `json:"payer"`
	Data  json.RawMessage `json:"data"`
}

// account.json
/*
{
	permissions: [
		{ name: "owner", owner: "", authoriy: obcode.Authority }
	],
}
*/

type linkAuth struct {
	permission string `json:"permission"`
	contract   string `json:"contract"`
	action     string `json:"action"`
}

type accountInfo struct {
	permissions     []pbcodec.PermissionObject
	linkPermissions []linkAuth
}

// Transfer represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName `json:"table"`
	Scope eos.ScopeName `json:"scope"`
	Payer eos.Name      `json:"payer"`
	Key   eos.Name      `json:"id"`
	Data  eos.HexBytes  `json:"data"`
}
