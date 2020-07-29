package migrator

import (
	"github.com/eoscanada/eos-go"
)

// Inject represents the `inject` struct on `migration` contract.
type Inject struct {
	Table eos.TableName `json:"table"`
	Scope eos.ScopeName `json:"scope"`
	Payer eos.Name      `json:"payer"`
	Key   eos.Name      `json:"id"`
	Data  eos.HexBytes  `json:"data"`
}

func newInjectAct(account eos.AccountName, table eos.TableName, scope eos.ScopeName, payer eos.AccountName, key eos.Name, data eos.HexBytes) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("inject"),
		Authorization: []eos.PermissionLevel{
			{Actor: payer, Permission: PN("active")},
		},
		ActionData: eos.NewActionData(Inject{Table: table, Scope: scope, Payer: eos.Name(payer), Key: key, Data: data}),
	}
}

// Idxi represents the `Idxi` struct on `migration` contract.
type Idxi struct {
	Table     eos.TableName `json:"table"`
	Scope     eos.ScopeName `json:"scope"`
	Payer     eos.Name      `json:"payer"`
	Key       eos.Name      `json:"id"`
	Secondary eos.Name      `json:"secondary"`
}

func newIdxi(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name, value eos.Name) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("idxi"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Idxi{
			Table:     tableName,
			Scope:     scope,
			Payer:     eos.Name(payer),
			Key:       primKey,
			Secondary: value,
		}),
	}
}

// Idxii represents the `Idxii` struct on `migration` contract.
type Idxii struct {
	Table     eos.TableName `json:"table"`
	Scope     eos.ScopeName `json:"scope"`
	Payer     eos.Name      `json:"payer"`
	Key       eos.Name      `json:"id"`
	Secondary eos.Uint128   `json:"secondary"`
}

func newIdxii(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name, value eos.Uint128) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("idxii"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Idxii{
			Table:     tableName,
			Scope:     scope,
			Payer:     eos.Name(payer),
			Key:       primKey,
			Secondary: value,
		}),
	}
}

// Idxc represents the `Idxc` struct on `migration` contract.
type Idxc struct {
	Table     eos.TableName   `json:"table"`
	Scope     eos.ScopeName   `json:"scope"`
	Payer     eos.Name        `json:"payer"`
	Key       eos.Name        `json:"id"`
	Secondary eos.Checksum256 `json:"secondary"`
}

func newIdxc(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name, value eos.Checksum256) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("idxc"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Idxc{
			Table:     tableName,
			Scope:     scope,
			Payer:     eos.Name(payer),
			Key:       primKey,
			Secondary: value,
		}),
	}
}

// Idxdbl represents the `Idxdbl` struct on `migration` contract.
type Idxdbl struct {
	Table     eos.TableName `json:"table"`
	Scope     eos.ScopeName `json:"scope"`
	Payer     eos.Name      `json:"payer"`
	Key       eos.Name      `json:"id"`
	Secondary float64       `json:"secondary"`
}

func newIdxdbl(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name, value float64) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("idxdbl"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Idxdbl{
			Table:     tableName,
			Scope:     scope,
			Payer:     eos.Name(payer),
			Key:       primKey,
			Secondary: value,
		}),
	}
}

// Idxldbl represents the `Idxldbl` struct on `migration` contract.
type Idxldbl struct {
	Table     eos.TableName `json:"table"`
	Scope     eos.ScopeName `json:"scope"`
	Payer     eos.Name      `json:"payer"`
	Key       eos.Name      `json:"id"`
	Secondary eos.Float128  `json:"secondary"`
}

func newIdxldbl(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name, value eos.Float128) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("idxldbl"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Idxldbl{
			Table:     tableName,
			Scope:     scope,
			Payer:     eos.Name(payer),
			Key:       primKey,
			Secondary: value,
		}),
	}
}

// Delete represents the `Delete` struct on `migration` contract.
type Eject struct {
	Account eos.AccountName `json:"account"`
	Table   eos.TableName   `json:"table"`
	Scope   eos.ScopeName   `json:"scope"`
	Key     eos.Name        `json:"id"`
}

func newEject(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, payer eos.AccountName, primKey eos.Name) *eos.Action {
	return &eos.Action{
		Account: account,
		Name:    ActN("eject"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      payer,
				Permission: PN("active"),
			},
		},
		ActionData: eos.NewActionData(Eject{
			Account: account,
			Table:   tableName,
			Scope:   scope,
			Key:     primKey,
		}),
	}
}
