package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

type AccountPath string
type TablePath string
type ScopePath string

type setupAccount func(name eos.AccountName)
type sendActionFunc func(action *eos.Action)
type AccountData struct {
	name string
	Path string
	abi  *eos.ABI
	ctr  *contract
}

var traceEnable = false

func init() {
	traceEnable = os.Getenv("TRACE") == "true"
}

func NewAccountData(dataDir string, account string) (*AccountData, error) {
	path, err := newAccountPath(dataDir, account)
	if err != nil {
		return nil, fmt.Errorf("unable to generate account data: %w", err)
	}
	return &AccountData{
		name: account,
		Path: path,
	}, nil
}

func (a *AccountData) setupAbi() error {
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

func (a *AccountData) migrateTable(table string, sendAction sendActionFunc) error {
	tablePath, err := a.TablePath(table)
	if err != nil {
		return fmt.Errorf("unable to create table path: %w", err)
	}

	scopes, err := a.readScopeList(tablePath)
	if err != nil {
		return fmt.Errorf("unable to read scopes: %w", err)
	}

	zlog.Debug("processing table scopes", zap.String("account", a.name), zap.String("table", table), zap.Int("scope_count", len(scopes)))

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
				TableRow: row,
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

func (a *AccountData) setContractActions() ([]*eos.Action, error) {
	return system.NewSetContractContent(AN(a.name), a.ctr.code, a.ctr.abi)

}

func (a *AccountData) readABI() (abi *eos.ABI, abiCnt []byte, err error) {
	cnt, err := ioutil.ReadFile(a.ABIPath())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read ABI for contract %q at path %q: %w", a.name, a.Path, err)
	}

	err = json.Unmarshal(cnt, &abi)
	if err != nil {
		return nil, nil, fmt.Errorf("unable decode ABI for contract %q at path %q: %w", a.name, a.Path, err)
	}

	return abi, cnt, nil
}

func (a *AccountData) readCode() (code []byte, err error) {
	cnt, err := ioutil.ReadFile(a.CodePath())
	if err != nil {
		return nil, fmt.Errorf("unable to read code for contract %q at path %q: %w", a.name, a.Path, err)
	}

	return cnt, nil
}

func (a *AccountData) readTableList() (out []string, err error) {
	files, err := ioutil.ReadDir(filepath.Join(a.Path, "tables"))
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			zlog.Warn("unexpected file in tables folder",
				zap.String("account", a.name),
				zap.String("account_path", a.Path),
				zap.String("filename", file.Name()),
			)
			continue
		}
		out = append(out, file.Name())
	}
	return
}

func (a *AccountData) readScopeList(table TablePath) ([]string, error) {
	path := a.ScopeListPath(table)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read scope list %q: %w", string(table), err)
	}
	defer file.Close()

	var scopes []string

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&scopes)
	if err != nil {
		return nil, fmt.Errorf("unable decode scopes %q list: %w", path, err)
	}
	return scopes, nil
}

func (a *AccountData) readRows(scpPath ScopePath) ([]TableRow, error) {
	path := a.RowsPath(scpPath)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read scope rows %q: %w", string(scpPath), err)
	}
	defer file.Close()

	var rows []TableRow

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rows)
	if err != nil {
		return nil, fmt.Errorf("unable decode rows %q: %w", path, err)
	}

	return rows, nil
}

func (a *AccountData) detailedTableRowToAction(row *DetailedTableRow) (*eos.Action, error) {
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

func (a *AccountData) decodeDetailedTableRow(row *DetailedTableRow) ([]byte, error) {
	tableDef := a.findTableDef(row.table)
	if tableDef == nil {
		return nil, fmt.Errorf("unable to find table definition %q in ABI for account: %q", row.table, row.account)
	}
	// TODO: need to check for a if type is alias...
	return a.abi.EncodeStruct(tableDef.Type, row.TableRow.Data)
}

// ABI helpers
func (a *AccountData) findTableDef(table eos.TableName) *eos.TableDef {
	for _, t := range a.abi.Tables {
		if t.Name == table {
			return &t
		}
	}
	return nil
}

// path helpers
func (a *AccountData) ABIPath() string {
	return filepath.Join(a.Path, "abi.json")
}

func (a *AccountData) CodePath() string {
	return filepath.Join(a.Path, "code.wasm")
}

func (a *AccountData) TablePath(table string) (TablePath, error) {
	if len(table) == 0 {
		return "", fmt.Errorf("received an empty table")
	}

	return TablePath(filepath.Join(a.Path, "tables", table)), nil
}

func (a *AccountData) ScopeListPath(tblPath TablePath) string {
	return filepath.Join(string(tblPath), "scopes.json")
}

func (a *AccountData) ScopePath(tblPath TablePath, scope string) (ScopePath, error) {
	if len(scope) == 0 {
		return "", fmt.Errorf("received an empty scope")
	}

	path := nestedPath(string(tblPath), scope)
	return ScopePath(path), nil
}

func (a *AccountData) RowsPath(scpPath ScopePath) string {
	return filepath.Join(string(scpPath), "rows.json")
}
