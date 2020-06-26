package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

var traceEnable = false

func init() {
	traceEnable = os.Getenv("TRACE") == "true"
}

type AccountPath string
type TablePath string
type ScopePath string

type sendActionFunc func(action *eos.Action)
type Account struct {
	name        string
	path        string
	hasContract bool
	abi         *eos.ABI
	ctr         *contract
	info        *accountInfo
}

func newAccount(dataDir string, account string) (*Account, error) {
	path, err := newAccountPath(dataDir, account)
	if err != nil {
		return nil, fmt.Errorf("unable to generate account data: %w", err)
	}
	return &Account{
		name: account,
		path: path,
	}, nil
}

func (a *Account) getAccountName() eos.AccountName { return AN(a.name) }
func (a *Account) setupAccountInfo() error {
	accountInfo, err := a.readAccount()
	if err != nil {
		return fmt.Errorf("cannot get information to create account %q: %w", a.name, err)
	}
	a.info = accountInfo
	return nil
}
func (a *Account) setupAbi() error {
	abi, abiCnt, err := a.readABI()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", a.name, err)
	}

	code, err := a.readCode()
	if err != nil {
		return fmt.Errorf("unable to get account %q Code: %w", a.name, err)
	}
	a.abi = abi // store for late use to encode rows
	a.ctr = &contract{
		abi:  abiCnt,
		code: code,
	}
	return nil
}

func (a *Account) migrateTable(table string, sendAction sendActionFunc) error {
	tablePath, err := a.TablePath(table)
	if err != nil {
		return fmt.Errorf("unable to create table path: %w", err)
	}

	walkScopes(string(tablePath), func(scope string) error {
		scopePath, err := a.ScopePath(tablePath, scope)
		if err != nil {
			return fmt.Errorf("unable to create scope path: %w", err)
		}

		rows, err := a.readRows(scopePath)
		if err != nil {
			return fmt.Errorf("unable to read rows contract %q, table %q scope %q: %w", a.name, table, scope, err)
		}

		for _, row := range rows {
			action, err := a.detailedTableRowToAction(&DetailedTableRow{
				tableRow: row,
				account:  AN(a.name),
				table:    TN(table),
				scope:    SN(scope),
			})
			if err != nil {
				return fmt.Errorf("unable to creation action for table row: %w", err)
			}
			sendAction(action)
		}
		return nil
	})

	return nil
}

func (a *Account) setContractActions() ([]*eos.Action, error) {
	return system.NewSetContractContent(AN(a.name), a.ctr.code, a.ctr.abi)

}

func (a *Account) readTableList() (out []string, err error) {
	path := filepath.Join(a.path, "tables")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// the tables folder doesn't exist no tables to read
		return out, nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			zlog.Warn("unexpected file in tables folder",
				zap.String("account", a.name),
				zap.String("account_path", a.path),
				zap.String("filename", file.Name()),
			)
			continue
		}
		out = append(out, file.Name())
	}
	return
}

func (a *Account) readRows(scpPath ScopePath) ([]tableRow, error) {
	path := a.RowsPath(scpPath)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read scope rows %q: %w", string(scpPath), err)
	}
	defer file.Close()

	var rows []tableRow

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rows)
	if err != nil {
		return nil, fmt.Errorf("unable decode rows %q: %w", path, err)
	}

	return rows, nil
}

func (a *Account) detailedTableRowToAction(row *DetailedTableRow) (*eos.Action, error) {
	data, err := a.decodeDetailedTableRow(row)
	if err != nil {
		return nil, fmt.Errorf("unable to decode row %q/%q/%q: %w", row.account, row.table, row.scope, err)
	}

	action := &eos.Action{
		Account: AN(a.name),
		Name:    ActN("inject"),
		Authorization: []eos.PermissionLevel{
			{Actor: AN(row.Payer), Permission: PN("active")},
		},
		ActionData: eos.NewActionData(Inject{Table: row.table, Scope: row.scope, Payer: eos.Name(row.Payer), Key: eos.Name(row.Key), Data: data}),
	}
	if traceEnable {
		zlog.Debug("action data",
			zap.String("scope", string(row.scope)),
			zap.String("payer", row.Payer),
			zap.String("key", row.Key),
			zap.String("table", string(row.table)),
			zap.Stringer("bytes", eos.HexBytes(data)),
		)
	}
	return action, nil
}

func (a *Account) decodeDetailedTableRow(row *DetailedTableRow) ([]byte, error) {
	tableDef := a.findTableDef(row.table)
	if tableDef == nil {
		return nil, fmt.Errorf("unable to find table definition %q in ABI for account: %q", row.table, row.account)
	}

	if len(row.tableRow.DataHex) > 0 {
		return row.tableRow.DataHex, nil
	}

	// TODO: need to check for a if type is alias...
	return a.abi.EncodeStruct(tableDef.Type, row.tableRow.DataJSON)
}

func (a *Account) createDir() error {
	return os.MkdirAll(a.path, os.ModePerm)
}

func (a *Account) readABI() (abi *eos.ABI, abiCnt []byte, err error) {
	cnt, err := ioutil.ReadFile(a.abiPath())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read ABI for contract %q at path %q: %w", a.name, a.path, err)
	}

	err = json.Unmarshal(cnt, &abi)
	if err != nil {
		return nil, nil, fmt.Errorf("unable decode ABI for contract %q at path %q: %w", a.name, a.path, err)
	}

	return abi, cnt, nil
}

func (a *Account) writeABI(abi *eos.ABI) error {
	return writeJSONFile(a.abiPath(), abi)
}

func (a *Account) abiPath() string {
	return filepath.Join(a.path, "abi.json")
}

func (a *Account) readCode() (code []byte, err error) {
	cnt, err := ioutil.ReadFile(a.codePath())
	if err != nil {
		return nil, fmt.Errorf("unable to read code for contract %q at path %q: %w", a.name, a.codePath(), err)
	}

	return cnt, nil
}

func (a *Account) writeCode(code []byte) error {
	return writeJSONFile(a.codePath(), code)
}

func (a *Account) readAccount() (accInfo *accountInfo, err error) {
	cnt, err := ioutil.ReadFile(a.accountPath())
	if err != nil {
		return nil, fmt.Errorf("unable to read account information %q at path %q: %w", a.name, a.accountPath(), err)
	}

	err = json.Unmarshal(cnt, &accInfo)
	if err != nil {
		return nil, fmt.Errorf("unable decode account information %q at path %q: %w", a.name, a.accountPath(), err)
	}

	return accInfo, err
}

func (a *Account) writeAccount(accInfo *accountInfo) error {
	return writeJSONFile(a.accountPath(), accInfo)
}

func (a *Account) accountPath() string {
	return filepath.Join(a.path, "account.json")
}

// ABI helpers
func (a *Account) findTableDef(table eos.TableName) *eos.TableDef {
	for _, t := range a.abi.Tables {
		if t.Name == table {
			return &t
		}
	}
	return nil
}

func (a *Account) codePath() string {
	return filepath.Join(a.path, "code.wasm")
}

func (a *Account) TablePath(table string) (TablePath, error) {
	if len(table) == 0 {
		return "", fmt.Errorf("received an empty table")
	}

	table = encodeName(table)
	return TablePath(filepath.Join(a.path, "tables", table)), nil
}

func (a *Account) ScopePath(tblPath TablePath, scope string) (ScopePath, error) {
	if len(scope) == 0 {
		return "", fmt.Errorf("received an empty scope")
	}

	scope = encodeName(scope)
	path := nestedPath(string(tblPath), scope)
	return ScopePath(path), nil
}

func (a *Account) RowsPath(scpPath ScopePath) string {
	return filepath.Join(string(scpPath), "rows.json")
}

func encodeName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}

func decodeName(name string) string {
	return strings.Replace(name, "_", ".", -1)
}
