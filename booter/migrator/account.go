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

type sendActionFunc func(action *eos.Action)
type sendTrxBoundaryFunc func()
type Account struct {
	name    string
	path    string
	hasCode bool

	abi  *eos.ABI
	ctr  *Contract
	info *AccountInfo

	logger *zap.Logger
}

func newAccount(dataDir string, account string) (*Account, error) {
	path, err := newAccountPath(dataDir, account)
	if err != nil {
		return nil, fmt.Errorf("unable to generate account data: %w", err)
	}
	return &Account{
		name:   account,
		path:   path,
		logger: zap.NewNop(),
	}, nil
}

func (a *Account) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

func (a *Account) getAccountName() eos.AccountName { return AN(a.name) }
func (a *Account) setupAccountInfo() error {
	accountInfo, err := a.readAccount()
	if err != nil {
		return fmt.Errorf("cannot get information to create account %q: %w", a.name, err)
	}

	accountInfo.setupIDtoPerm()
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
		return fmt.Errorf("unable to get account %q code: %w", a.name, err)
	}
	a.abi = abi // store for late use to encode rows
	a.ctr = NewContract(abiCnt, code)
	return nil
}

func (a *Account) migrateTable(table string, sendAction sendActionFunc, endTransaction sendTrxBoundaryFunc) error {
	walkScopes(a.tablePath(table), func(scope string) error {
		tableScope, err := a.readTableScope(table, scope)
		if err != nil {
			//ultra-duncan --- UB-1517 fix empty scope import
			//In case of scope is an empty string, it will assume the scope will be same as table name
			//So if this case fail, will assume scope is empty and do an additional read
			//NOTE: if scope = table data is existed, this will ignore empty scope. Need better solution in future.
			if table == scope {
				scope = ""
				tableScope, err = a.readTableScope(table, scope)
			}

			if err != nil {
				return fmt.Errorf("unable to retrieve table scope %q:%q: %w", table, scope, err)
			}
		}

		if len(tableScope.rows) == 0 {
			return nil
		}

		hasInjectedFirstRow := false
		var preActs []*eos.Action
		var postActs []*eos.Action
		var exclPrimKey *string
		for primKey, _ := range tableScope.rows {
			if !hasInjectedFirstRow {
				exclPrimKey, preActs, postActs, err = tableScope.payerActions(primKey, a.abi, a.logger)
				if err != nil {
					a.logger.Error("unable to setup table-scope payer",
						zap.String("account", a.name),
						zap.String("table", table),
						zap.String("scope", scope),
						zap.String("error", err.Error()),
					)
				} else if len(preActs) > 0 {
					if traceEnable {
						a.logger.Debug("executing pre-actions",
							zap.String("account", a.name),
							zap.String("table", table),
							zap.String("scope", scope),
							zap.Stringp("exclude_primary_key", exclPrimKey),
							zap.Int("pre_actions", len(preActs)),
							zap.Int("post_actions", len(postActs)),
						)
					}
					// send pre first row transaction
					for _, action := range preActs {
						sendAction(action)
					}
					endTransaction()
				} else {
					if traceEnable {
						a.logger.Debug("no pre-actions to execute")
					}
				}
			}

			if exclPrimKey != nil && *exclPrimKey == primKey {
				// skip already injected row
				hasInjectedFirstRow = true
				continue
			}

			actions, err := tableScope.rowToActions(a.abi, primKey, a.logger)
			if err != nil {
				return fmt.Errorf("unable to get actions for table-scope %s:%s: %w", table, scope, err)
			}

			for _, action := range actions {
				sendAction(action)
			}
			endTransaction()

			if !hasInjectedFirstRow {
				// send post first row transaction
				for _, action := range postActs {
					sendAction(action)
				}
				endTransaction()
			}
			hasInjectedFirstRow = true
		}
		return nil
	})
	return nil
}

func (a *Account) setContractActions() ([]*eos.Action, error) {
	return system.NewSetContractContent(AN(a.name), a.ctr.Code, a.ctr.RawABI)

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
			a.logger.Warn("unexpected file in tables folder",
				zap.String("account", a.name),
				zap.String("account_path", a.path),
				zap.String("filename", file.Name()),
			)
			continue
		}
		out = append(out, decodeName(file.Name()))
	}
	return
}

func (a *Account) readTableScope(table, scope string) (*tableScope, error) {
	tblScope, err := readTableScopeInfo(a.tableScopeInfoPath(table, scope))
	if err != nil {
		return nil, fmt.Errorf("cannot read table scope info: %w", err)
	}

	rows, err := readTableScopeRows(a.rowsPath(table, scope))
	if err != nil {
		return nil, fmt.Errorf("cannot read table scope rows: %w", err)
	}

	tblScope.account = AN(a.name)
	tblScope.table = TN(table)
	tblScope.scope = SN(scope)
	tblScope.rows = make(map[string]*tableRow, len(rows))

	tblpayer := tblScope.payer()
	for _, row := range rows {
		if row.Payer == tblpayer {
			tblScope.hasTblPayerAsRow = true
			tblScope.tblScpPayerPrimKey = row.Key
		}
		tblScope.rows[row.Key] = row
	}
	return tblScope, nil
}

func (a *Account) createDir() error {
	return os.MkdirAll(a.path, os.ModePerm)
}

func (a *Account) readABI() (abi *eos.ABI, abiCnt []byte, err error) {
	return readABI(a.abiPath())
}

func (a *Account) writeABI() error {
	return writeJSONFile(a.abiPath(), a.abi)
}

func (a *Account) abiPath() string {
	return filepath.Join(a.path, "abi.json")
}

func (a *Account) readCode() (code []byte, err error) {
	return readCode(a.codePath())
}

func (a *Account) writeCode() error {
	return ioutil.WriteFile(a.codePath(), a.ctr.Code, os.ModePerm)
}

func (a *Account) readAccount() (accInfo *AccountInfo, err error) {
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

func (a *Account) writeAccount() error {
	return writeJSONFile(a.accountPath(), a.info)
}

func (a *Account) accountPath() string {
	return filepath.Join(a.path, "account.json")
}

func (a *Account) codePath() string {
	return filepath.Join(a.path, "code.wasm")
}

func (a *Account) tablePath(table string) string {
	return filepath.Join(a.path, "tables", encodeName(table))
}

func (a *Account) scopePath(table string, scope string) string {
	return nestedPath(a.tablePath(table), encodeName(scope))
}

func (a *Account) rowsPath(table string, scope string) string {
	return filepath.Join(a.scopePath(table, scope), "rows.json")
}

func (a *Account) tableScopeInfoPath(table, scope string) string {
	return filepath.Join(a.scopePath(table, scope), "info.json")
}

func encodeName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}

func decodeName(name string) string {
	return strings.Replace(name, "_", ".", -1)
}
