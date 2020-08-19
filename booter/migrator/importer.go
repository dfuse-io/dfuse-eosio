package migrator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/eoscanada/eos-go/system"

	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go"

	rice "github.com/GeertJohan/go.rice"
	"github.com/eoscanada/eos-go/ecc"
	"go.uber.org/zap"
)

//go:generate rice embed-go

var nonceActionEntropy = func() string {
	return hex.EncodeToString(generateRandomNonce())
}

type Contract struct {
	RawABI []byte
	Code   []byte
}

func NewContract(abi []byte, code []byte) *Contract {
	return &Contract{
		RawABI: abi,
		Code:   code,
	}
}

type importer struct {
	common

	opPublicKey ecc.PublicKey
	actionChan  chan interface{}
	ctr         *Contract
	logger      *zap.Logger
}

func newImporter(opPublicKey ecc.PublicKey, dataDir string, actionChan chan interface{}, logger *zap.Logger) *importer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &importer{
		common:      common{dataDir: dataDir},
		opPublicKey: opPublicKey,
		actionChan:  actionChan,
		logger:      logger,
	}
}

func (i *importer) init() error {
	box := rice.MustFindBox("./code/build")
	abiCnt, err := readBoxFile(box, "migrator.abi")
	if err != nil {
		return fmt.Errorf("unable to open migration abi cnt: %w", err)
	}

	wasmCnt, err := readBoxFile(box, "migrator.wasm")
	if err != nil {
		return fmt.Errorf("unable to open migration wasm cnt: %w", err)
	}

	i.logger.Debug("read migrator contract")
	i.ctr = NewContract(abiCnt, wasmCnt)
	return nil
}

// TODO: cannot call this import :(
func (i *importer) inject() error {
	accounts, err := i.retrieveAccounts(func(account *Account) error {
		if isNativeChainAccount(account) {
			i.logger.Info("Skipping the creation of native account",
				zap.String("account", account.name),
			)
			return nil
		}
		i.createAccount(account)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to create chain accounts: %w", err)
	}

	for _, account := range accounts {
		i.logger.Debug("processing account", zap.String("account", account.name))

		err := account.setupAccountInfo()
		if err != nil {
			return fmt.Errorf("unable to setup account %q: %w", account.name, err)
		}

		err = i.createPermissions(account)
		if err != nil {
			return fmt.Errorf("unable to create permissions for accounts %q: %w", account.name, err)
		}
		if !account.hasCode {
			continue
		}

		err = i.migrateContract(account)
		if err != nil {
			i.logger.Error("unable to process account",
				zap.String("account", account.name),
				zap.Error(err),
			)
		}
	}

	// cleanup
	importerAuthority := i.importerAuthority()
	for _, account := range accounts {
		i.logger.Debug("cleaning up account", zap.String("account", account.name))
		err = i.setPermissions(account, &importerAuthority)
		if err != nil {
			return fmt.Errorf("unable to create permissions for accounts %q: %w", account.name, err)
		}
	}

	return nil
}

func (i *importer) migrateContract(accountData *Account) error {
	err := accountData.setupAbi()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", accountData.name, err)
	}

	i.logger.Debug("setting importer contract")
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
		i.logger.Debug("migrating table",
			zap.String("account", accountData.name),
			zap.String("table", table),
		)
		err = accountData.migrateTable(
			table,
			func(action *eos.Action) {
				i.actionChan <- (*bootops.TransactionAction)(action)
			},
			func() {
				i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
			},
		)
		if err != nil {
			i.logger.Warn("unable to migrate account tables",
				zap.String("account", accountData.name),
				zap.String("table", table),
				zap.Error(err),
			)
			continue
		}
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
	actions, err := system.NewSetContractContent(account, i.ctr.Code, i.ctr.RawABI)
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

func (i *importer) createAccount(account *Account) {
	i.actionChan <- (*bootops.TransactionAction)(system.NewNewAccount("eosio", account.getAccountName(), i.opPublicKey))
	i.actionChan <- (*bootops.TransactionAction)(system.NewSetalimits(account.getAccountName(), -1, -1, -1))
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
}

func (i *importer) createPermissions(account *Account) error {
	if isNativeChainAccount(account) {
		i.logger.Info("skipping permission creation for native account",
			zap.String("account", account.name),
		)
		return nil
	}
	currentParent := PN("")
	for _, permission := range account.info.sortPermissions() {
		parent := PN("")
		if p, found := account.info.nameToPerm[permission.Parent]; found {
			parent = p.Name
		}

		// Small optimization here to push all permission that are on the same level (a.k.a have the same parent) in the same transaction
		if (currentParent != "") && (currentParent != permission.Parent) {
			i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
		}
		currentParent = parent

		// NOTE: even though the permission are correctly ordered in creation we neeed to ensure that the parent
		// so we cannot push them all in a transaction
		i.actionChan <- (*bootops.TransactionAction)(system.NewUpdateAuth(account.getAccountName(), permission.Name, parent, i.importerAuthority(), PN("owner")))
	}
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
	return nil
}

func (i *importer) setPermissions(account *Account, importerAuthority *eos.Authority) error {
	if isNativeChainAccount(account) {
		i.logger.Info("skipping permission setting for native account",
			zap.String("account", account.name),
		)
		return nil
	}
	// the link auth is signed with active account so lets perform this first before potentially updating the active account
	for _, linkAuth := range account.info.LinkAuths {
		i.actionChan <- (*bootops.TransactionAction)(system.NewLinkAuth(account.getAccountName(), AN(linkAuth.Contract), eos.ActionName(linkAuth.Action), PN(linkAuth.Permission)))
	}
	i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction

	var ownerPermission *PermissionObject
	for _, permission := range account.info.sortPermissions() {
		if i.shouldSetPermission(importerAuthority, permission.Authority) {
			if permission.Name == "owner" {
				// we will only update the owner permission once all the permission for said account has been update
				// since we are "signing" the actions with the current owner permission
				ownerPermission = permission
				continue
			}

			parentName := PN("")
			if parent, found := account.info.nameToPerm[permission.Parent]; found {
				parentName = parent.Name
			}
			i.actionChan <- (*bootops.TransactionAction)(system.NewUpdateAuth(account.getAccountName(), permission.Name, parentName, *permission.Authority, PN("owner")))
			i.actionChan <- (*bootops.TransactionAction)(newNonceAction())
			i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
		}
	}

	if ownerPermission != nil {
		if i.shouldSetPermission(importerAuthority, ownerPermission.Authority) {
			i.actionChan <- (*bootops.TransactionAction)(system.NewUpdateAuth(account.getAccountName(), ownerPermission.Name, "", *ownerPermission.Authority, PN("owner")))
			i.actionChan <- (*bootops.TransactionAction)(newNonceAction())
			i.actionChan <- bootops.EndTransaction(i.opPublicKey) // end transaction
		}
	}

	return nil

}

func (i *importer) shouldSetPermission(importerAuthority, authority *eos.Authority) bool {
	// TODO: this is temporary since the protocol features are not activated
	for _, key := range authority.Keys {
		if strings.HasPrefix(key.PublicKey.String(), "PUB_WA") {
			return false
		}
	}

	return !reflect.DeepEqual(importerAuthority, authority)
}

func (i *importer) importerAuthority() eos.Authority {
	return eos.Authority{
		Threshold: 1,
		Keys: []eos.KeyWeight{
			{
				PublicKey: i.opPublicKey,
				Weight:    1,
			},
		},
	}
}

func newNonceAction() *eos.Action {
	return &eos.Action{
		Account: eos.AN("eosio.null"),
		Name:    eos.ActN("nonce"),
		ActionData: eos.NewActionData(system.Nonce{
			Value: nonceActionEntropy(),
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

func isNativeChainAccount(account *Account) bool {
	nativeAccount := []string{"eosio", "eosio.null", "eosio.prods"}
	for _, acc := range nativeAccount {
		if acc == account.name {
			return true
		}
	}
	return false
}
