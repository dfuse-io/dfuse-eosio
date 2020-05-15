package sqlsync

import (
	"fmt"
	"strconv"

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

type account struct {
	tables map[eos.TableName]*Table
	name   string
	abi    *eos.ABI
}

func (a *account) extractTables() {
	out := make(map[eos.TableName]*Table)
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
		out[table.Name] = &Table{
			chainName: string(table.Name),
			dbName:    a.name + "_" + string(table.Name), // TODO: allow custom mapping of chain names to db name
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

var chainToSQLTypes = map[string]string{
	"name":   "varchar(13) NOT NULL",
	"string": "varchar(1024) NOT NULL",
	"symbol": "varchar(8) NOT NULL",
	"bool":   "boolean",
	"int64":  "int NOT NULL",
	"uint64": SQL_UINT64,
	"int32":  "int NOT NULL", // make smaller
	"uint32": SQL_UINT32,
	"asset":  "varchar(64) NOT NULL",
}

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
