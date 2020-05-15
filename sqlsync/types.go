package sqlsync

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eoscanada/eos-go"
	"github.com/tidwall/gjson"
)

// Must contain ALL fields. If ABI to JSON decoding didn't produce a given field (in ORDER, according to our ABI -> SQL mapping)
type Row []interface{}

var SimpleAssetsMappings map[string]Mappings // table to its mappings

type Mappings []Mapping
type MappingMap map[string]Mapping

type Mapping struct {
	ChainField string
	DBField    string
	Type       string
	KeepJSON   bool
}

type Table struct {
	chainName string
	dbName    string
	mappings  Mappings
}

func (t *Table) createTableStatement() string {
	stmt := "CREATE TABLE " + t.dbName + `(
  _scope varchar(13) NOT NULL,
  _key varchar(13) NOT NULL,
  _payer varchar(13) NOT NULL,
`
	for _, field := range t.mappings {
		stmt = stmt + " " + field.ChainField + " "
		if field.KeepJSON {
			stmt = stmt + "text NOT NULL,"
		} else {
			stmt = stmt + chainToSQLTypes[field.Type] + " NOT NULL,"
		}
	}

	stmt += ` PRIMARY KEY (_scope, _key)
);`
	return stmt
}

func (t *Table) insertStatement(db *DB, scope, key, payer string, jsonData gjson.Result) (string, []interface{}, error) {
	stmt := "INSERT INTO " + t.dbName + "(_scope, _key, _payer"
	for _, field := range t.mappings {
		stmt = stmt + ", " + field.DBField
	}
	stmt = stmt + ") VALUES " + db.paramsPlaceholderFunc(3+len(t.mappings))

	values := []interface{}{scope, key, payer}
	fieldValues, err := t.valuesList(jsonData)
	if err != nil {
		return "", nil, err
	}
	values = append(values, fieldValues...)

	return stmt, values, nil
}

func (t *Table) updateStatement(db *DB, scope, key, payer string, jsonData gjson.Result) (string, []interface{}, error) {
	s := db.paramsPlaceholderFunc(3 + len(t.mappings))
	s = strings.Trim(s, "()")
	placeHolders := strings.Split(s, ",")

	stmt := "UPDATE " + t.dbName + " SET _payer = " + placeHolders[0]
	for idx, field := range t.mappings {
		stmt = stmt + fmt.Sprintf(", %s = %s", field.DBField, placeHolders[idx+1])
	}
	stmt = stmt + " WHERE _scope = " + placeHolders[len(t.mappings)+1] + " AND _key = " + placeHolders[len(t.mappings)+2]

	values := []interface{}{payer}
	fieldValues, err := t.valuesList(jsonData)
	if err != nil {
		return "", nil, err
	}
	values = append(values, fieldValues...)
	values = append(values, scope, key)

	return stmt, values, nil
}

func (t *Table) valuesList(jsonData gjson.Result) (out []interface{}, err error) {
	for _, m := range t.mappings {
		val := jsonData.Get(m.ChainField)
		if m.KeepJSON {
			out = append(out, val.Raw)
		} else {
			convertedValue, err := mapToSQLType(val, m.Type)
			if err != nil {
				return nil, fmt.Errorf("converting raw JSON %q to %s: %w", val.Raw, m.Type, err)
			}
			out = append(out, convertedValue)
		}
	}
	return
}

func (t *Table) deleteStatement() string {
	return fmt.Sprintf("DELETE FROM %s WHERE _scope = $1 AND _key = $2", t.dbName)
}

type account struct {
	tables map[string]*Table // table name -> table def
	name   string
	abi    *eos.ABI
}

func (a *account) extractTables(tablePrefix string) {
	out := make(map[string]*Table)
	for _, table := range a.abi.Tables {
		var mappings []Mapping
		struc := a.abi.StructForName(table.Type)
		// TODO: add support for `base` type, therefore add fields for that one too.
		for _, field := range struc.Fields {
			mappings = append(mappings, Mapping{
				ChainField: field.Name,
				DBField:    field.Name,
				KeepJSON:   !stringInFilter(field.Type, parsableFieldTypes),
				Type:       field.Type,
			})
		}
		out[string(table.Name)] = &Table{
			chainName: string(table.Name),
			dbName:    tablePrefix + a.name + "_" + string(table.Name), // TODO: allow custom mapping of chain names to db name
			mappings:  mappings,
		}
	}
	a.tables = out
}

var parsableFieldTypes = []string{
	"name",
	"string",
	"symbol",
	"bool",
	"int64",
	"uint64",
	"int32",
	"uint32",
	"asset",
}

var chainToSQLTypes map[string]string

func mapToSQLType(val gjson.Result, typ string) (out interface{}, err error) {
	// WHAT-IF: We could short-circuit this TOTALLY by implementing
	// hooks in the ABI decoder so we could map native EOS types
	// directly to SQL interface{} value types.
	//
	// Right now, we'll be doing binary -> JSON, then JSON -> SQL
	// through its declared type in ABI + JSON representation (!!)
	notSupported := func() error {
		return fmt.Errorf("json type %s not supported for chain type %s", val.Type.String(), typ)
	}

	switch typ {
	case "name":
		out = val.String()
	case "string":
		out = val.String()
	case "bool":
		switch val.Type {
		case gjson.True:
			out = true
		case gjson.False:
			out = false
		default:
			err = notSupported()
		}
	case "asset":
		// TODO: check what we mean according to desired mapping
		// we could decide that an `asset` would be split into 2-3 fields,
		// where would this occur?
		out = val.String()
	case "int64":
		switch val.Type {
		case gjson.Null, gjson.False, gjson.True, gjson.JSON:
			err = notSupported()
		case gjson.String:
			// interprete the string
			out, err = strconv.ParseInt(val.Str, 10, 64)
		case gjson.Number:
			out, err = strconv.ParseInt(val.Raw, 10, 64)
		}
	case "uint64":
		switch val.Type {
		case gjson.Null, gjson.False, gjson.True, gjson.JSON:
			err = notSupported()
		case gjson.String:
			// interprete the string
			out, err = strconv.ParseUint(val.Str, 10, 64)
		case gjson.Number:
			out, err = strconv.ParseUint(val.Raw, 10, 64)
		}

	default:
		err = notSupported()
	}

	return
}
