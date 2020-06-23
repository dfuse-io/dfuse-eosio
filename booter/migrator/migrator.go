package migrator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

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
	err := m.createChainAccounts()
	if err != nil {
		zlog.Error("unable to create chain accounts", zap.Error(err))
		return
	}

	contracts, err := ReadContractList(m.dataDir)
	if err != nil {
		zlog.Error("unable to read contract list", zap.Error(err))
		return
	}

	zlog.Info("retrieved contract list", zap.Int("contract_count", len(contracts)))

	walkContracts(m.dataDir, func(contract string) error {
		account, err := NewAccountData(m.dataDir, contract)
		if err != nil {
			zlog.Error("unable to initiate account migration", zap.String("contract", contract))
			return nil
		}

		err = m.migrateAccount(account)
		if err != nil {
			zlog.Error("unable to process account", zap.String("contract", contract), zap.Error(err))
			return nil
		}
		return nil
	})
}

func (m *Migrator) migrateAccount(accountData *AccountData) error {

	zlog.Debug("processing account", zap.String("account", accountData.name))

	err := accountData.setupAbi()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", accountData.name, err)
	}

	err = m.setMigratorCode(AN(accountData.name))
	if err != nil {
		return fmt.Errorf("unable to set migrator code for account: %w", err)
	}

	tables, err := accountData.readTableList()
	if err != nil {
		return fmt.Errorf("unable to get table list for account %q: %w", m.name, err)
	}

	for _, table := range tables {
		// we need to create the payers account first before we can create the table rows
		err = accountData.migrateTable(
			table,
			func(action *eos.Action) {
				m.actionChan <- (*bootops.TransactionAction)(action)
				m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction
			},
		)
		m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction
	}

	if err != nil {
		return fmt.Errorf("unable to migrate account: %w", err)
	}

	err = m.resetAccountContract(accountData)
	if err != nil {
		return fmt.Errorf("unable to set account %q to original contract: %w", accountData.name, err)
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
	m.actionChan <- (*bootops.TransactionAction)(newNonceAction())
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
	m.actionChan <- (*bootops.TransactionAction)(system.NewSetalimits(account, -1, -1, -1))
	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction

	m.accountCache[account] = true

	return
}

func (m *Migrator) resetAccountContract(act *AccountData) error {
	actions, err := act.setContractActions()
	if err != nil {
		return fmt.Errorf("unable to set account contract %q: %w", act.name, err)
	}

	for _, action := range actions {
		m.actionChan <- (*bootops.TransactionAction)(action)
	}
	m.actionChan <- (*bootops.TransactionAction)(newNonceAction())
	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction
	return nil
}

func (m *Migrator) createChainAccounts() error {
	accounts, err := ReadAccountList(m.dataDir)
	if err != nil {
		return fmt.Errorf("unable to read account list: %w", err)
	}

	for _, account := range accounts {
		m.setupAccount(AN(account))
	}

	return nil
}

func newNonceAction() *eos.Action {
	return &eos.Action{
		Account: eos.AN("eosio.null"),
		Name:    eos.ActN("nonce"),
		ActionData: eos.NewActionData(system.Nonce{
			Value: hex.EncodeToString(generateRandomNonce()),
		}),
	}
}

func generateRandomNonce() []byte {
	// Use 48 bits of entropy to generate a valid random
	nonce := make([]byte, 6)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(fmt.Sprintf("unable to correctly generate nonce: %s", err))
	}
	return nonce
}
