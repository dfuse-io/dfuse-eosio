package migrator

import (
	"encoding/json"

	"github.com/eoscanada/eos-go"
)

type detailedTableRow struct {
	tableRow

	account eos.AccountName
	table   eos.TableName
	scope   eos.ScopeName
}

type tableScope struct {
	account eos.AccountName
	table   eos.TableName
	scope   eos.ScopeName
	Payers  []string `json:"payers,omitempty"`
	rows    map[string]*tableRow
}

const (
	secondaryIndexKindUI64       string = "ui64"
	secondaryIndexKindUI128      string = "ui128"
	secondaryIndexKindUI256      string = "ui256"
	secondaryIndexKindDouble     string = "dbl"
	secondaryIndexKindLongDouble string = "ldbl"
)

type secondaryIndex struct {
	Kind  string      `json:"kind,omitempty"`
	Value interface{} `json:"value,omitempty"`
	Payer string      `json:"payer,omitempty"`
}

type tableRow struct {
	Key              string            `json:"key"`
	Payer            string            `json:"payer"`
	DataJSON         json.RawMessage   `json:"json_data,omitempty"`
	DataHex          eos.HexBytes      `json:"hex_data,omitempty"`
	SecondaryIndexes []*secondaryIndex `json:"secondary_indexes,omitempty"`
}

// Transfer represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName `json:"table"`
	Scope eos.ScopeName `json:"scope"`
	Payer eos.Name      `json:"payer"`
	Key   eos.Name      `json:"id"`
	Data  eos.HexBytes  `json:"data"`
}
