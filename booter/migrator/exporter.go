package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/eoscanada/eos-go"
	eossnapshot "github.com/eoscanada/eos-go/snapshot"
	"go.uber.org/zap"
)

type exporter struct {
	snapshotPath  string
	logger        *zap.Logger
	outputDataDir string

	accounts      map[eos.AccountName]*Account
	codeSequences map[string][]eos.AccountName

	tableScopes  map[string]*tableScope // (account:table:scope)
	currentTable *eossnapshot.TableIDObject
}

type Option func(e *exporter) *exporter

func WithLogger(logger *zap.Logger) Option {
	return func(e *exporter) *exporter {
		e.logger = logger
		return e
	}
}

func NewExporter(snapshotPath, dataDir string, opts ...Option) (*exporter, error) {
	if !fileExists(snapshotPath) {
		return nil, fmt.Errorf("snapshot file not found %q", snapshotPath)
	}

	e := &exporter{
		snapshotPath:  snapshotPath,
		outputDataDir: dataDir,
		accounts:      map[eos.AccountName]*Account{},
		codeSequences: map[string][]eos.AccountName{},
		tableScopes:   map[string]*tableScope{},
	}

	for _, opt := range opts {
		e = opt(e)
	}

	return e, nil
}

func (e *exporter) Export() error {
	reader, err := eossnapshot.NewDefaultReader(e.snapshotPath)
	if err != nil {
		return fmt.Errorf("unable to create a snapshot reader: %w", err)
	}
	defer func() {
		reader.Close()
	}()

	for {
		err := reader.NextSection()
		if err == io.EOF {
			break
		}
		if err != nil {
			e.logger.Error("failed reading snapshot",
				zap.String("snapshot_path", e.snapshotPath),
				zap.Error(err),
			)
			return err
		}

		currentSection := reader.CurrentSection
		if traceEnable {
			e.logger.Debug("new section",
				zap.String("section_name", string(currentSection.Name)),
				zap.Uint64("row_count", currentSection.RowCount),
				zap.Uint64("buffer_size", currentSection.BufferSize),
				zap.Uint64("offset", currentSection.Offset),
			)
		}

		switch currentSection.Name {
		case eossnapshot.SectionNameAccountObject:
			e.logger.Info("reading snapshot account objects")
			err = reader.ProcessCurrentSection(e.processAccountObject)
		case eossnapshot.SectionNameAccountMetadataObject:
			e.logger.Info("reading snapshot account metadata objects")
			err = reader.ProcessCurrentSection(e.processAccountMetadataObject)
		case eossnapshot.SectionNamePermissionObject:
			e.logger.Info("reading snapshot permission objects")
			err = reader.ProcessCurrentSection(e.processPermissionObject)
		case eossnapshot.SectionNamePermissionLinkObject:
			e.logger.Info("reading snapshot permission link objects")
			err = reader.ProcessCurrentSection(e.processPermissionLinkObject)
		case eossnapshot.SectionNameCodeObject:
			e.logger.Info("reading snapshot code objects")
			err = reader.ProcessCurrentSection(e.processCodeObject)
		case eossnapshot.SectionNameContractTables:
			e.logger.Info("reading snapshot contract tables")
			err = reader.ProcessCurrentSection(e.processContractTable)
		}

		if err == eossnapshot.ErrSectionHandlerNotFound {
			e.logger.Warn("section handler not found",
				zap.String("section_name", string(currentSection.Name)),
			)
			break
		}

		if err != nil {
			e.logger.Error("failed processing snapshot section",
				zap.String("section_name", string(currentSection.Name)),
				zap.Error(err),
			)
			return err
		}
	}
	e.logger.Info("reading snapshot file completed",
		zap.Int("account_count", len(e.accounts)),
		zap.Int("table_count", len(e.tableScopes)),
	)

	e.logger.Info("exporting accounts")
	for accountName, account := range e.accounts {
		err = e.exportAccount(accountName, account)
		if err != nil {
			return fmt.Errorf("failed to export account %q: %w", string(accountName), err)
		}
	}

	e.logger.Info("exporting table scopes")
	for key, tblScope := range e.tableScopes {
		err = e.exportTableScope(tblScope)
		if err != nil {
			return fmt.Errorf("failed to export table-scope %s : %w", key, err)
		}
	}
	return nil
}

func (e *exporter) processAccountObject(obj interface{}) error {
	acc, ok := obj.(eossnapshot.AccountObject)
	if !ok {
		return fmt.Errorf("failed processing account object: unexpected object type: %T", obj)
	}

	if _, found := e.accounts[acc.Name]; found {
		return fmt.Errorf("failed processing account object: received seen account %q", string(acc.Name))
	}

	if traceEnable {
		e.logger.Debug("processing account object: adding new account", zap.String("account_name", string(acc.Name)))
	}

	account, err := e.newAccount(acc)
	if err != nil {
		return fmt.Errorf("failed processing account object: unable to create account %q: %w", string(acc.Name), err)
	}

	e.accounts[acc.Name] = account
	return nil
}

var emptyCodeHash = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func (e *exporter) processAccountMetadataObject(obj interface{}) error {
	accMeta, ok := obj.(eossnapshot.AccountMetadataObject)
	if !ok {
		return fmt.Errorf("failed processing account metadata object: unexpected object type: %T", obj)
	}

	codeKey := accMeta.CodeHash.String()
	accName := accMeta.Name

	if bytes.Equal(accMeta.CodeHash, emptyCodeHash) {
		if traceEnable {
			e.logger.Debug("skipping blank code hash",
				zap.String("account", string(accName)),
			)
		}
		return nil
	}
	accounts, found := e.codeSequences[codeKey]
	if !found {
		accounts = []eos.AccountName{}
		e.codeSequences[codeKey] = accounts
	}

	accounts = append(accounts, accName)

	if traceEnable {
		e.logger.Debug("processing account metadata object: storing code sequence",
			zap.String("account", string(accName)),
			zap.String("code_key", codeKey),
			zap.Int("account_count", len(accounts)),
		)
	}

	return nil
}

func (e *exporter) processPermissionObject(obj interface{}) error {
	perm, ok := obj.(eossnapshot.PermissionObject)
	if !ok {
		return fmt.Errorf("process permission object: unexpected object type: %T", obj)
	}
	e.updatePermissionObject(perm)
	return nil
}

func (e *exporter) processPermissionLinkObject(obj interface{}) error {
	permLink, ok := obj.(eossnapshot.PermissionLinkObject)
	if !ok {
		return fmt.Errorf("process permission link object: unexpected object type: %T", obj)
	}
	e.updatePermissionLinkObject(permLink)
	return nil
}

func (e *exporter) processCodeObject(obj interface{}) error {
	code, ok := obj.(eossnapshot.CodeObject)
	if !ok {
		return fmt.Errorf("process code object: unexpected object type: %T", obj)
	}
	e.updateCodeObject(code)
	return nil
}

func (e *exporter) newAccount(acc eossnapshot.AccountObject) (*Account, error) {
	account, err := newAccount(e.outputDataDir, string(acc.Name))
	if err != nil {
		return nil, err
	}

	account.info = &AccountInfo{}
	account.logger = e.logger

	if len(acc.RawABI) > 0 {
		account.ctr = NewContract(acc.RawABI, nil)

		abi := new(eos.ABI)
		err = eos.UnmarshalBinary(acc.RawABI, abi)
		if err != nil {
			e.logger.Warn("unable to decode ABI",
				zap.String("account", string(acc.Name)),
				zap.Error(err),
			)
			return account, nil
		}
		account.abi = abi
	}

	return account, nil

}

func (e *exporter) updatePermissionObject(perm eossnapshot.PermissionObject) error {
	acc, found := e.accounts[perm.Owner]
	if !found {
		return fmt.Errorf("failed updating account permission object: unknown account %q", string(perm.Owner))
	}

	if traceEnable {
		e.logger.Debug("adding permission object to account",
			zap.String("account", string(perm.Owner)),
			zap.Reflect("permission", perm),
		)
	}

	acc.info.Permissions = append(acc.info.Permissions, &PermissionObject{
		Parent:    perm.Parent,
		Owner:     perm.Owner,
		Name:      perm.Name,
		Authority: &perm.Auth,
	})
	return nil
}

func (e *exporter) updatePermissionLinkObject(permLink eossnapshot.PermissionLinkObject) error {
	acc, found := e.accounts[permLink.Account]
	if !found {
		return fmt.Errorf("failed updating account permission link object: unknown account %q", string(permLink.Account))
	}

	if traceEnable {
		e.logger.Debug("adding permission link to account",
			zap.String("account", string(permLink.Account)),
			zap.Reflect("permission_link", permLink),
		)
	}

	acc.info.LinkAuths = append(acc.info.LinkAuths, &LinkAuth{
		Permission: string(permLink.RequiredPermission),
		Contract:   string(permLink.Code),
		Action:     string(permLink.MessageType),
	})
	return nil
}

func (e *exporter) updateCodeObject(code eossnapshot.CodeObject) error {
	codeKey := code.CodeHash.String()
	accounts, found := e.codeSequences[codeKey]
	if !found {
		return fmt.Errorf("failed updating account code: unknown code ref: %d", uint64(code.CodeRefCount))
	}

	for _, accName := range accounts {
		acc, found := e.accounts[accName]
		if !found {
			return fmt.Errorf("failed updating account code: unknown account: %q", string(accName))
		}

		acc.ctr.Code = code.Code
		if traceEnable {
			e.logger.Debug("adding code to account",
				zap.String("account", string(accName)),
				zap.String("cod_ref", codeKey),
			)
		}

	}
	return nil
}

func (e *exporter) processContractTable(o interface{}) error {
	tableId, ok := o.(*eossnapshot.TableIDObject)
	if ok {
		if err := e.processTableID(tableId); err != nil {
			return err
		}
		e.currentTable = tableId
		return nil
	}

	if e.currentTable == nil {
		return fmt.Errorf("cannot process contract row without having a current table set")
	}

	var kind secondaryIndexKind
	var secKey interface{}
	var primKey string
	var payer string
	switch obj := o.(type) {
	case *eossnapshot.KeyValueObject:
		return e.processContractRow(obj)
	case *eossnapshot.Index64Object:
		kind = secondaryIndexKindUI64
		primKey = obj.PrimKey
		secKey = obj.SecondaryKey
		payer = obj.Payer
	case *eossnapshot.Index128Object:
		kind = secondaryIndexKindUI128
		primKey = obj.PrimKey
		secKey = obj.SecondaryKey
		payer = obj.Payer
	case *eossnapshot.Index256Object:
		kind = secondaryIndexKindUI256
		primKey = obj.PrimKey
		secKey = obj.SecondaryKey
		payer = obj.Payer
	case *eossnapshot.IndexDoubleObject:
		kind = secondaryIndexKindDouble
		primKey = obj.PrimKey
		secKey = obj.SecondaryKey
		payer = obj.Payer
	case *eossnapshot.IndexLongDoubleObject:
		kind = secondaryIndexKindLongDouble
		primKey = obj.PrimKey
		secKey = obj.SecondaryKey
		payer = obj.Payer
	}
	return e.processContractRowIndex(primKey, kind, secKey, payer)
}

func (e *exporter) processTableID(obj *eossnapshot.TableIDObject) error {
	tableName, index := mustExtractIndexNumber(obj.TableName)

	tableScopeKey := tableScopeKey(obj.Code, tableName, obj.Scope)
	tblScope, found := e.tableScopes[tableScopeKey]
	if found && (index == 0) { // a table should only have index 0
		return fmt.Errorf("received table id for a seen code-table-scope %s", tableScopeKey)
	}

	if !found && (index > 0) {
		return fmt.Errorf("received table id index for a unseen code-table-scope%s", tableScopeKey)
	}

	if found && (index > 0) {
		if traceEnable {
			e.logger.Debug("processing index table",
				zap.String("account", obj.Code),
				zap.String("table_name", tableName),
				zap.String("raw_table_name", obj.TableName),
				zap.String("scope", obj.Scope),
				zap.Uint64("index", index),
			)
		}
		tblScope.idxToPayers[index] = obj.Payer
		return nil
	}

	tblScope = &tableScope{
		account:     AN(obj.Code),
		table:       TN(tableName),
		scope:       SN(obj.Scope),
		rows:        map[string]*tableRow{},
		idxToPayers: map[uint64]string{},
	}
	tblScope.idxToPayers[index] = obj.Payer
	e.tableScopes[tableScopeKey] = tblScope

	if traceEnable {
		e.logger.Info("process contract row: added table-scope",
			zap.String("account", obj.Code),
			zap.String("table_name", tableName),
			zap.String("scope", obj.Scope),
			zap.String("payer", obj.Payer),
		)
	}
	return nil
}

func (e *exporter) processContractRow(obj *eossnapshot.KeyValueObject) error {
	account, found := e.accounts[AN(e.currentTable.Code)]
	if !found {
		return fmt.Errorf("unable to process contract row: unknown account %q", AN(e.currentTable.Code))
	}

	tableScopeKey := tableScopeKey(e.currentTable.Code, e.currentTable.TableName, e.currentTable.Scope)
	tblScope, found := e.tableScopes[tableScopeKey]
	if !found {
		return fmt.Errorf("received contract row for unseen code-table-scope: %s", tableScopeKey)
	}

	tblRow := &tableRow{
		Key:          obj.PrimKey,
		Payer:        obj.Payer,
		idToSecIndex: map[uint64]*secondaryIndex{},
	}

	if _, found := tblScope.rows[obj.PrimKey]; found {
		return fmt.Errorf("failed to process contract table row %s: primary key already seen %q", tableScopeKey, obj.PrimKey)
	}

	if account.abi != nil {
		data, err := e.decodeTableRow(account.abi, obj)
		if err != nil {
			e.logger.Debug("unable to decode table row",
				zap.String("account", e.currentTable.Code),
				zap.String("table", e.currentTable.TableName),
				zap.String("scope", e.currentTable.Scope),
				zap.String("primary_key", obj.PrimKey),
				zap.String("error", err.Error()),
			)
		} else {
			tblRow.DataJSON = data
		}
	}

	if tblRow.DataJSON == nil {
		tblRow.DataHex = obj.Value
	}

	tblScope.rows[obj.PrimKey] = tblRow
	return nil
}

func (e *exporter) decodeTableRow(abi *eos.ABI, obj *eossnapshot.KeyValueObject) ([]byte, error) {
	tableName := TN(e.currentTable.TableName)
	tablDef := findTableDefInABI(abi, tableName)
	if tablDef == nil {
		return nil, fmt.Errorf("cannot find table definition %q", tableName)
	}
	cnt, err := abi.DecodeTableRow(tableName, obj.Value)
	if err != nil {
		return nil, fmt.Errorf("unable to decode the data falling back on hex: %w", err)
	}
	_, err = abi.EncodeStruct(tablDef.Type, cnt)
	if err != nil {
		return nil, fmt.Errorf("unable to re-encoded the data falling back on hex: %w", err)
	}
	return json.RawMessage(cnt), nil
}

func (e *exporter) processContractRowIndex(primaryKey string, kind secondaryIndexKind, value interface{}, payer string) error {
	tableName, id := mustExtractIndexNumber(e.currentTable.TableName)
	tableScopeKey := tableScopeKey(e.currentTable.Code, tableName, e.currentTable.Scope)
	tblScope, found := e.tableScopes[tableScopeKey]
	if !found {
		return fmt.Errorf("cannot process contract row index for unseen code-table-scope: %s", tableScopeKey)
	}

	tblRow, found := tblScope.rows[primaryKey]
	if !found {
		return fmt.Errorf("cannot process contract row index %s for unseen primary key %q", tableScopeKey, primaryKey)
	}

	rValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cannot marshal secondary %q: %w", primaryKey, err)
	}

	tblRow.idToSecIndex[id] = &secondaryIndex{
		Kind:  kind,
		Value: json.RawMessage(rValue),
		Payer: payer,
	}
	return nil
}

func tableScopeKey(Account, TableName, Scope string) string {
	return fmt.Sprintf("%s:%s:%s", Account, TableName, Scope)
}

func (e *exporter) exportAccount(accountName eos.AccountName, account *Account) error {
	if traceEnable {
		e.logger.Debug("exporting account", zap.String("account", string(accountName)))
	}

	if err := account.createDir(); err != nil {
		return fmt.Errorf("unable to create account dir for %q: %w", accountName, err)
	}

	if err := account.writeAccount(); err != nil {
		return fmt.Errorf("unable to write account for %q: %w", accountName, err)
	}

	if account.ctr != nil {
		if err := account.writeCode(); err != nil {
			return fmt.Errorf("unable to write ABI for %q: %w", accountName, err)
		}

		if err := account.writeABI(); err != nil {
			return fmt.Errorf("unable to write ABI for %q: %w", accountName, err)
		}
	}

	return nil
}

func (s *exporter) exportTableScope(tableScope *tableScope) error {
	if traceEnable {
		s.logger.Debug("writing table scope",
			zap.String("account", string(tableScope.account)),
			zap.String("table", string(tableScope.table)),
			zap.String("scope", string(tableScope.scope)),
			zap.Int("row_count", len(tableScope.rows)),
		)
	}

	tableScope.setupPayer()

	account, found := s.accounts[tableScope.account]
	if !found {
		return fmt.Errorf("cannot export table-scope for unknown account %q", string(tableScope.account))
	}

	scopePath := account.scopePath(string(tableScope.table), string(tableScope.scope))

	if err := os.MkdirAll(string(scopePath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create table scope storage path: %w", err)
	}

	if err := s.writeTableInfo(account.tableScopeInfoPath(string(tableScope.table), string(tableScope.scope)), tableScope); err != nil {
		return fmt.Errorf("unable to write table rows: %s:%s:%s: %w", string(tableScope.account), string(tableScope.table), string(tableScope.scope), err)
	}

	if err := s.writeTableRows(account.rowsPath(string(tableScope.table), string(tableScope.scope)), tableScope.rows); err != nil {
		return fmt.Errorf("unable to write table rows: %s:%s:%s: %w", string(tableScope.account), string(tableScope.table), string(tableScope.scope), err)
	}
	return nil
}

func (s *exporter) writeTableInfo(tableInfoPath string, table *tableScope) error {
	return writeJSONFile(tableInfoPath, table)
}

func (s *exporter) writeTableRows(rowsPath string, rows map[string]*tableRow) error {
	file, err := os.Create(rowsPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	lastIndex := len(rows) - 1
	file.WriteString("[")
	itr := 0
	for primKey, tabletRow := range rows {
		tabletRow.setupSecondaryIndexes()

		encoder := json.NewEncoder(file)
		encoder.SetEscapeHTML(false)

		file.WriteString("\n  ")

		err := encoder.Encode(tabletRow)
		if err != nil {
			return fmt.Errorf("unable to encode row %q: %w", primKey, err)
		}

		if itr != lastIndex {
			file.WriteString(",")
		}
		itr = itr + 1
	}
	file.WriteString("]")
	return nil
}
