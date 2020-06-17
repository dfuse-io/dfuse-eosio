package migrator

import (
	"fmt"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/eoscanada/eos-go"

	rice "github.com/GeertJohan/go.rice"
	system "github.com/eoscanada/eos-go/system"
)

//go:generate rice embed-go

type Migrator struct {
	box      *rice.Box
	contract eos.AccountName
}

func newMigrator() *Migrator {
	return &Migrator{
		box:      rice.MustFindBox("./code/build"),
		contract: eos.AN("dfuse.mgrt"),
	}
}

func (m *Migrator) newAccountActions(publicKey ecc.PublicKey, in chan interface{}) (err error) {
	in <- system.NewNewAccount("eosio", m.contract, publicKey)
	in <- system.NewBuyRAMBytes("eosio", m.contract, 100000)
	return
}

func (m *Migrator) setContractActions(contract eos.AccountName, in chan interface{}) error {
	abiCnt, err := readBoxFile(m.box, "migrator.abi")
	if err != nil {
		return fmt.Errorf("unable to open migration abi cnt: %w", err)
	}

	wasmCnt, err := readBoxFile(m.box, "migrator.wasm")
	if err != nil {
		return fmt.Errorf("unable to open migration wasm cnt: %w", err)
	}

	actions, err := system.NewSetContractContent(contract, wasmCnt, abiCnt)
	if err != nil {
		return fmt.Errorf("unable to create set contract actions: %w", err)
	}

	for _, action := range actions {
		in <- action
	}

	return nil
}

func (m *Migrator) processContractTable(contract eos.AccountName, tableName eos.TableName, table contractTable, in chan interface{}) error {
	for scope, rows := range table {
		for _, row := range rows {
			in <- &eos.Action{
				Account: AN("eosio.token"),
				Name:    ActN("inject"),
				Authorization: []eos.PermissionLevel{
					{Actor: m.contract, Permission: PN("active")},
				},
				ActionData: eos.NewActionData(&Inject{
					Table: tableName,
					Scope: scope,
					Payer: row.Payer,
					Key:   row.Key,
				}),
			}
		}

	}
	return nil
}
