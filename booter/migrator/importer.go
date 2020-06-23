package migrator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/eoscanada/eos-go/system"

	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go"

	rice "github.com/GeertJohan/go.rice"
	"github.com/eoscanada/eos-go/ecc"
	"go.uber.org/zap"
)

//go:generate rice embed-go

const boxPath = "./code/build"

type contract struct {
	abi  []byte
	code []byte
}

type importer struct {
	common

	opPublicKey ecc.PublicKey
	actionChan  chan interface{}
	ctr         *contract
}

func newImporter(opPublicKey ecc.PublicKey, dataDir string, actionChan chan interface{}) *importer {
	return &importer{
		common:      common{dataDir: dataDir},
		opPublicKey: opPublicKey,
		actionChan:  actionChan,
	}
}

func (i *importer) init() error {
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
	i.ctr = &contract{abi: abiCnt, code: wasmCnt}
	return nil
}

// TODO: cannot call this import :(
func (i *importer) inject() error {
	contracts, err := i.retrieveContractAccounts(func(account string) error {
		i.createAccount(AN(account))
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to create chain accounts: %w", err)
	}

	for _, contract := range contracts {
		err = i.migrateAccount(contract)
		if err != nil {
			zlog.Error("unable to process account",
				zap.String("account", contract.name),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (i *importer) migrateAccount(accountData *Account) error {

	zlog.Debug("processing account", zap.String("account", accountData.name))

	err := accountData.setupAbi()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", accountData.name, err)
	}

	err = i.setImporterContract(AN(accountData.name))
	if err != nil {
		return fmt.Errorf("unable to set migrator code for account: %w", err)
	}

	tables, err := accountData.readTableList()
	if err != nil {
		return fmt.Errorf("unable to get table list for account %q: %w", accountData.name, err)
	}

	for _, table := range tables {
		// we need to create the payers account first before we can create the table rows
		err = accountData.migrateTable(
			table,
			func(action *eos.Action) {
				i.actionChan <- (*bootops.TransactionAction)(action)
				i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
			},
		)
		i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
	}

	if err != nil {
		return fmt.Errorf("unable to migrate account: %w", err)
	}

	err = i.resetAccountContract(accountData)
	if err != nil {
		return fmt.Errorf("unable to set account %q to original contract: %w", accountData.name, err)
	}

	return nil
}

func (i *importer) resetAccountContract(act *Account) error {
	actions, err := act.setContractActions()
	if err != nil {
		return fmt.Errorf("unable to set account contract %q: %w", act.name, err)
	}

	for _, action := range actions {
		i.actionChan <- (*bootops.TransactionAction)(action)
	}
	i.actionChan <- (*bootops.TransactionAction)(newNonceAction())
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
	return nil
}

func (i *importer) setImporterContract(account eos.AccountName) error {
	zlog.Debug("setting importer contract")
	actions, err := system.NewSetContractContent(account, i.ctr.code, i.ctr.abi)
	if err != nil {
		return fmt.Errorf("unable to create set contract actions: %w", err)
	}

	for _, action := range actions {
		i.actionChan <- (*bootops.TransactionAction)(action)
	}
	i.actionChan <- (*bootops.TransactionAction)(newNonceAction())
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction

	return nil
}

func (i *importer) createAccount(account eos.AccountName) {
	i.actionChan <- (*bootops.TransactionAction)(system.NewNewAccount("eosio", account, i.opPublicKey))
	i.actionChan <- (*bootops.TransactionAction)(system.NewSetalimits(account, -1, -1, -1))
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
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
