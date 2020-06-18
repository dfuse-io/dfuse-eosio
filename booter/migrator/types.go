package migrator

import (
	"encoding/json"

	"github.com/eoscanada/eos-go"
)

//accts/dfu/se44shine/tables/posts/eo/scanadacom.json
//accts/{[acc]/[ountName]}/contract.wasm
//accts/{[acc]/[ountName]}/contract.abi
//accts/{[acc]/[ountName]}/resources.json
//accts/{[acc]/[ountName]}/permissions.json

//accts/{[acc]/[ountName]}/tables/[tableName]/{[sco]/[peName]}.json
//Scope, TableName, Contract
//Payer, Data,

//account dfuse.boot setCode wasm.contract
type TableRow struct {
	Key   string      `json:"key"`
	Payer string      `json:"payer"`
	Data  interface{} `json:"data"`
}

// Transfer represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName
	Scope eos.ScopeName
	Payer string
	Key   string
}

func (i *Inject) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Scope uint64 `json:"scope"`
		Table uint64 `json:"table"`
		Payer uint64 `json:"payer"`
		Id    uint64 `json:"id"`
	}{
		Scope: UINT64(string(i.Scope)),
		Table: UINT64(string(i.Table)),
		Payer: UINT64(i.Payer),
		Id:    UINT64(i.Key),
	})
}
