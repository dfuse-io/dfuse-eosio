package migrator

import (
	"encoding/json"

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
	Key      string          `json:"key"`
	Payer    string          `json:"payer"`
	DataJSON json.RawMessage `json:"json_data,omitempty"`
	DataHex  eos.HexBytes    `json:"hex_data,omitempty"`
}

// Transfer represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName `json:"table"`
	Scope eos.ScopeName `json:"scope"`
	Payer eos.Name      `json:"payer"`
	Key   eos.Name      `json:"id"`
	Data  eos.HexBytes  `json:"data"`
}
