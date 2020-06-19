package migrator

import (
	"encoding/json"

	"github.com/eoscanada/eos-go"
)

type DetailedTableRow struct {
	TableRow

	account eos.AccountName
	table   eos.TableName
	scope   eos.ScopeName
}

//account dfuse.boot setCode wasm.contract
type TableRow struct {
	Key   string          `json:"key"`
	Payer string          `json:"payer"`
	Data  json.RawMessage `json:"data"`
}

// Transfer represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName `json:"table"`
	Scope eos.ScopeName `json:"scope"`
	Payer eos.Name      `json:"payer"`
	Key   eos.Name      `json:"id"`
	Data  eos.HexBytes  `json:"data"`
}
