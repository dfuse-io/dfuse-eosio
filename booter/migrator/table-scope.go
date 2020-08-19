package migrator

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/eoscanada/eos-go"
)

type tableScope struct {
	account eos.AccountName
	table   eos.TableName
	scope   eos.ScopeName
	Payers  []string `json:"payers,omitempty"`

	rows map[string]*tableRow

	// this is use a reference to the primary key of the first row
	// we need to insert. This ensure that the RAM ops stay consistent
	hasTblPayerAsRow   bool
	tblScpPayerPrimKey string
	idxToPayers        map[uint64]string
}

func (t *tableScope) payer() string {
	if len(t.Payers) == 0 {
		return ""
	}
	return t.Payers[0]
}

func (t *tableScope) setupPayer() {
	t.Payers = make([]string, len(t.idxToPayers))
	for id, payer := range t.idxToPayers {
		if payer != "" {
			t.Payers[id] = payer
		}
	}
}

var (
	tableScopePayerNotFound = errors.New("table-scope payer not in rows")
)

var oneByte = []byte{0x01}

// this function returns actions to be execute before,
// and after the first table-scope row gets created
func (t *tableScope) payerActions(primKey string, abi *eos.ABI, logger *zap.Logger) (excl *string, preAct []*eos.Action, postAct []*eos.Action, err error) {
	if !t.hasTblPayerAsRow {
		payerRowKey := mustIncrementPrimKey(primKey)
		logger.Debug("table-scope payer not in rows, creating dummy trxs")
		// we create a dummy transaction to be inserted before the row to associate
		// the desired payer to the primary table and all the secondary index table
		// then once the row is inserted we will delete the dummy transaction we created
		// ASSUMPTION is that all the rows in the table-scope has all the same secondary indexes set
		for tblIdx, payer := range t.Payers {
			table := mustCreateIndexTable(string(t.table), uint64(tblIdx))
			preAct = append(preAct, newInjectAct(t.account, TN(table), t.scope, AN(payer), payerRowKey, oneByte))
			postAct = append(postAct, newEject(t.account, TN(table), t.scope, AN(payer), payerRowKey))
		}
		return nil, preAct, postAct, nil
	}
	if row, found := t.rows[t.tblScpPayerPrimKey]; found {
		// we don't want the rowToAction here to log anything
		actions, err := t.rowToActions(abi, row.Key, zap.NewNop())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to get table-scope payer row actions: %w", err)
		}
		logger.Debug("table-scope payer rows action",
			zap.Int("action_count", len(actions)),
			zap.String("primary_key", row.Key),
		)
		return s(row.Key), actions, nil, nil
	}
	return nil, nil, nil, fmt.Errorf("table-scope payer rows not found")
}

func (t *tableScope) rowToActions(abi *eos.ABI, primKey string, logger *zap.Logger) (out []*eos.Action, err error) {
	row, found := t.rows[primKey]
	if !found {
		return nil, fmt.Errorf("cannot find row in table-scope with primary key %q", primKey)
	}

	data, err := row.encode(abi, t.table)
	if err != nil {
		return nil, fmt.Errorf("unable to encode row %q: %w", primKey, err)
	}
	out = append(out, newInjectAct(t.account, t.table, t.scope, AN(row.Payer), eos.Name(row.Key), data))

	if traceEnable {
		logger.Debug("contract row inject",
			zap.String("account", string(t.account)),
			zap.String("table", string(t.table)),
			zap.String("scope", string(t.scope)),
			zap.String("key", row.Key),
			zap.String("payer", row.Payer),
			zap.Stringer("bytes", eos.HexBytes(data)),
		)
	}

	idxActs, err := row.indexesToAction(t.account, t.table, t.scope)
	if err != nil {
		return nil, fmt.Errorf("unable to creation index actions for table row %q: %w", primKey, err)
	}

	if traceEnable {
		logger.Debug("contract row idx",
			zap.Int("action_count", len(idxActs)),
			zap.String("account", string(t.account)),
			zap.String("table", string(t.table)),
			zap.String("scope", string(t.scope)),
			zap.String("key", row.Key),
		)
	}
	if len(idxActs) > 0 {
		out = append(out, idxActs...)
	}
	return out, nil
}

type tableRow struct {
	Key              string            `json:"key"`
	Payer            string            `json:"payer"`
	DataJSON         json.RawMessage   `json:"json_data,omitempty"`
	DataHex          eos.HexBytes      `json:"hex_data,omitempty"`
	SecondaryIndexes []*secondaryIndex `json:"secondary_indexes,omitempty"`

	idToSecIndex map[uint64]*secondaryIndex
}

func (t *tableRow) setupSecondaryIndexes() {
	t.SecondaryIndexes = make([]*secondaryIndex, len(t.idToSecIndex))
	for id, index := range t.idToSecIndex {
		t.SecondaryIndexes[id] = index
	}
}

func (t *tableRow) encode(abi *eos.ABI, table eos.TableName) ([]byte, error) {
	if len(t.DataHex) > 0 {
		return t.DataHex, nil
	}

	tableDef := findTableDefInABI(abi, table)
	if tableDef == nil {
		return nil, fmt.Errorf("unable to find table definition %q in ABI", table)
	}

	// TODO: need to check for a if type is alias...
	return abi.EncodeStruct(tableDef.Type, t.DataJSON)
}

func (t *tableRow) indexesToAction(account eos.AccountName, table eos.TableName, scope eos.ScopeName) (out []*eos.Action, err error) {
	for id, index := range t.SecondaryIndexes {

		indexTable := mustCreateIndexTable(string(table), uint64(id))
		action, err := index.idxToAction(account, TN(indexTable), scope, eos.Name(t.Key))
		if err != nil {
			return nil, fmt.Errorf("unable to get action for secondary index for key %q at index %d: %w", t.Key, id, err)
		}

		out = append(out, action)
	}
	return out, nil
}

type secondaryIndexKind string

const (
	secondaryIndexKindUI64       secondaryIndexKind = "ui64"
	secondaryIndexKindUI128      secondaryIndexKind = "ui128"
	secondaryIndexKindUI256      secondaryIndexKind = "ui256"
	secondaryIndexKindDouble     secondaryIndexKind = "dbl"
	secondaryIndexKindLongDouble secondaryIndexKind = "ldbl"
)

type secondaryIndex struct {
	Kind  secondaryIndexKind `json:"kind,omitempty"`
	Value json.RawMessage    `json:"value,omitempty"`
	Payer string             `json:"payer,omitempty"`
}

func (s *secondaryIndex) idxToAction(account eos.AccountName, tableName eos.TableName, scope eos.ScopeName, primKey eos.Name) (*eos.Action, error) {
	switch s.Kind {
	case secondaryIndexKindUI64:
		var value eos.Name
		err := json.Unmarshal([]byte(s.Value), &value)
		if err != nil {
			return nil, err
		}
		return newIdxi(account, tableName, scope, AN(s.Payer), primKey, value), nil
	case secondaryIndexKindUI128:
		var value eos.Uint128
		err := json.Unmarshal([]byte(s.Value), &value)
		if err != nil {
			return nil, err
		}
		return newIdxii(account, tableName, scope, AN(s.Payer), primKey, value), nil
	case secondaryIndexKindUI256:
		var value eos.Checksum256
		err := json.Unmarshal([]byte(s.Value), &value)
		if err != nil {
			return nil, err
		}
		return newIdxc(account, tableName, scope, AN(s.Payer), primKey, value), nil
	case secondaryIndexKindDouble:
		var value eos.Float64
		err := json.Unmarshal([]byte(s.Value), &value)
		if err != nil {
			return nil, err
		}
		return newIdxdbl(account, tableName, scope, AN(s.Payer), primKey, float64(value)), nil
	case secondaryIndexKindLongDouble:
		var value eos.Float128
		err := json.Unmarshal([]byte(s.Value), &value)
		if err != nil {
			return nil, err
		}
		return newIdxldbl(account, tableName, scope, AN(s.Payer), primKey, value), nil
	default:
		return nil, fmt.Errorf("unexpected secondary index type: %q", s.Kind)
	}
}
