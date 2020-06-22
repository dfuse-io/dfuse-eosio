package migrator

import (
	"fmt"

	rice "github.com/GeertJohan/go.rice"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	system "github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

//go:generate rice embed-go

const boxPath = "./code/build"

type contract struct {
	abi  []byte
	code []byte
}

type Migrator struct {
	name         eos.AccountName
	opPublicKey  ecc.PublicKey
	actionChan   chan interface{}
	dataDir      string
	accountCache map[eos.AccountName]bool
	ctr          *contract
}

func newMigrator(account eos.AccountName, opPublicKey ecc.PublicKey, dataDir string, actionChan chan interface{}) *Migrator {
	return &Migrator{
		name:         account,
		dataDir:      dataDir,
		opPublicKey:  opPublicKey,
		actionChan:   actionChan,
		accountCache: map[eos.AccountName]bool{},
	}
}

func (m *Migrator) init() error {
	box := rice.MustFindBox(boxPath)
	abiCnt, err := readBoxFile(box, "migrator.abi")
	if err != nil {
		return fmt.Errorf("unable to open migration abi cnt: %w", err)
	}

	wasmCnt, err := readBoxFile(box, "migrator.wasm")
	if err != nil {
		return fmt.Errorf("unable to open migration wasm cnt: %w", err)
	}

	zlog.Debug("setup migrator contract")
	m.ctr = &contract{abi: abiCnt, code: wasmCnt}

	zlog.Info("setting injector account", zap.String("account", string(m.name)))
	m.actionChan <- (*bootops.TransactionAction)(system.NewNewAccount("eosio", m.name, m.opPublicKey))
	m.actionChan <- (*bootops.TransactionAction)(system.NewBuyRAMBytes("eosio", m.name, 100000))
	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction

	return nil
}

func (m *Migrator) migrate() {
	contracts, err := ReadContractList(m.dataDir)
	if err != nil {
		zlog.Error("unable to read contract list", zap.Error(err))
		return
	}

	zlog.Info("retrieved contract list", zap.Int("contract_count", len(contracts)))

	for _, contract := range contracts {

		account, err := NewAccountData(m.dataDir, contract)
		if err != nil {
			zlog.Error("unable to initiate account migration", zap.String("contract", contract))
			continue
		}

		err = m.migrateAccount(account)
		if err != nil {
			zlog.Error("unable to process account", zap.String("contract", contract), zap.Error(err))
			continue
		}
	}
}

func (m *Migrator) migrateAccount(accountData *AccountData) error {

	zlog.Debug("processing account", zap.String("account", accountData.name))

	err := m.setMigratorCode(AN(accountData.name))
	if err != nil {
		return fmt.Errorf("unable to set migrator code for account: %w", err)
	}

	err = accountData.setupAbi()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", m.name, err)
	}

	tables, err := accountData.readTableList()
	if err != nil {
		return fmt.Errorf("unable to get table list for account %q: %w", m.name, err)
	}

	for _, table := range tables {
		// we need to create the payers account first before we can create the table rows
		talbleRowsAction := []*bootops.TransactionAction{}
		err = accountData.migrateTable(
			table,
			func(name eos.AccountName) {
				m.setupAccount(name)
			},
			func(action *eos.Action) {
				talbleRowsAction = append(talbleRowsAction, (*bootops.TransactionAction)(action))
			},
		)

		for _, trxAct := range talbleRowsAction {
			m.actionChan <- trxAct
		}
		m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction
	}

	if err != nil {
		return fmt.Errorf("unable to migrate account: %w", err)
	}

	return nil
}

func (m *Migrator) setMigratorCode(account eos.AccountName) error {

	zlog.Debug("setting migrator code", zap.String("account", string(account)))
	actions, err := system.NewSetContractContent(account, m.ctr.code, m.ctr.abi)
	if err != nil {
		return fmt.Errorf("unable to create set contract actions: %w", err)
	}

	for _, action := range actions {
		m.actionChan <- (*bootops.TransactionAction)(action)
	}

	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction

	return nil
}

func (m *Migrator) setupAccount(account eos.AccountName) {
	if _, ok := m.accountCache[account]; ok {
		return
	}
	if traceEnable {
		zlog.Info("setting user account", zap.String("account", string(account)))
	}

	m.actionChan <- (*bootops.TransactionAction)(system.NewNewAccount("eosio", account, m.opPublicKey))
	m.actionChan <- (*bootops.TransactionAction)(system.NewBuyRAMBytes("eosio", account, 100000))
	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction

	m.accountCache[account] = true

	return
}
