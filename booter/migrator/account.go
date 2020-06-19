package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type AccountPath string
type TablePath string
type ScopePath string

type sendFunc func(action *eos.Action)
type AccountData struct {
	name string
	Path string
	abi  *eos.ABI
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

func (m *AccountData) Migrate(send sendFunc) error {
	abi, err := m.readABI()
	if err != nil {
		return fmt.Errorf("unable to get account %q ABI: %w", m.name, err)
	}
	m.abi = abi // store for late use to encode rows

	//code, err := m.readCode()
	//if err != nil {
	//	return fmt.Errorf("unable to get account %q Code: %w", m.name, err)
	//}
	tables, err := m.readTableList()
	if err != nil {
		return fmt.Errorf("unable to get table list for account %q: %w", m.name, err)
	}

	//zlog.Debug("processing tables", zap.String("account", m.name), zap.Int("table_count", len(tables)))

	for _, table := range tables {
		tablePath, err := m.TablePath(table)
		if err != nil {
			return fmt.Errorf("unable to create table path: %w", err)
		}

		scopes, err := m.readScopeList(tablePath)
		if err != nil {
			return fmt.Errorf("unable to read scopes: %w", err)
		}

		zlog.Debug("processing table scopes", zap.String("account", m.name), zap.String("table", table), zap.Int("scope_count", len(scopes)))

		for _, scope := range scopes {
			scopePath, err := m.ScopePath(tablePath, scope)
			if err != nil {
				return fmt.Errorf("unable to create scope path: %w", err)
			}

			rows, err := m.readRows(scopePath)
			if err != nil {
				return fmt.Errorf("unable to read rows contract %q, table %q scope %q: %w", m.name, table, scope, err)
			}

			for _, row := range rows {
				action, err := m.detailedTableRowToAction(&DetailedTableRow{
					TableRow: row,
					account:  AN(m.name),
					table:    TN(table),
					scope:    SN(scope),
				})
				if err != nil {
					return fmt.Errorf("unable to creation action for table row: %w", err)
				}
				send(action)
			}
		}
	}
	return nil
}

func (m *AccountData) readABI() (abi *eos.ABI, err error) {
	file, err := os.Open(m.ABIPath())
	if err != nil {
		return nil, fmt.Errorf("unable to read ABI for contract %q at path %q: %w", m.name, m.Path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&abi)
	if err != nil {
		return nil, fmt.Errorf("unable decode ABI for contract %q at path %q: %w", m.name, m.Path, err)
	}

	return abi, nil
}

func (m *AccountData) readCode() (code []byte, err error) {
	cnt, err := ioutil.ReadFile(m.CodePath())
	if err != nil {
		return nil, fmt.Errorf("unable to read code for contract %q at path %q: %w", m.name, m.Path, err)
	}

	return cnt, nil
}

func (m *AccountData) readTableList() (out []string, err error) {
	files, err := ioutil.ReadDir(filepath.Join(m.Path, "tables"))
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			zlog.Warn("unexpected file in tables folder",
				zap.String("account", m.name),
				zap.String("account_path", m.Path),
				zap.String("filename", file.Name()),
			)
			continue
		}
		out = append(out, file.Name())
	}
	return
}

func (m *AccountData) readScopeList(table TablePath) ([]string, error) {
	path := m.ScopeListPath(table)
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

func (m *AccountData) readRows(scpPath ScopePath) ([]TableRow, error) {
	path := m.RowsPath(scpPath)
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

func (m *AccountData) detailedTableRowToAction(row *DetailedTableRow) (*eos.Action, error) {
	data, err := m.decodeDetailedTableRow(row)
	if err != nil {
		return nil, fmt.Errorf("unable to decode row %q/%q/%q: %w", row.account, row.table, row.scope, err)
	}
	action := &eos.Action{
		Account: AN(m.name),
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

func (m *AccountData) decodeDetailedTableRow(row *DetailedTableRow) ([]byte, error) {
	tableDef := m.findTableDef(row.table)
	if tableDef == nil {
		return nil, fmt.Errorf("unable to find table definition %q in ABI for account: %q", row.table, row.account)
	}
	// TODO: need to check for a if type is alias...
	return m.abi.EncodeStruct(tableDef.Type, row.TableRow.Data)
}

// ABI helpers
func (m *AccountData) findTableDef(table eos.TableName) *eos.TableDef {
	for _, t := range m.abi.Tables {
		if t.Name == table {
			return &t
		}
	}
	return nil
}

// path helpers
func (m *AccountData) ABIPath() string {
	return filepath.Join(m.Path, "abi.json")
}

func (m *AccountData) CodePath() string {
	return filepath.Join(m.Path, "code.wasm")
}

func (m *AccountData) TablePath(table string) (TablePath, error) {
	if len(table) == 0 {
		return "", fmt.Errorf("received an empty table")
	}

	return TablePath(filepath.Join(m.Path, "tables", table)), nil
}

func (m *AccountData) ScopeListPath(tblPath TablePath) string {
	return filepath.Join(string(tblPath), "scopes.json")
}

func (m *AccountData) ScopePath(tblPath TablePath, scope string) (ScopePath, error) {
	if len(scope) == 0 {
		return "", fmt.Errorf("received an empty scope")
	}

	path := nestedPath(string(tblPath), scope)
	return ScopePath(path), nil
}

func (m *AccountData) RowsPath(scpPath ScopePath) string {
	return filepath.Join(string(scpPath), "rows.json")
}
