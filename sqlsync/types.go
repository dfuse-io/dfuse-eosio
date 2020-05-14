package sqlsync

import "github.com/eoscanada/eos-go"

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
	name     string
	mappings Mappings
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
			name:     a.name + "_" + string(table.Name),
			mappings: mappings,
		}
	}
	a.tables = out
}

func init() {
	//	r := Row{}
	//	abiDecoded := abi.DecodeStruct("sasset", binaryFromFluxDB)
	//	for _, mapping := range tables {
	//		result := gjson.ParseBytes(abiDecoded, mapping.ChainField)
	//		switch mapping.Type {
	//		case "string":
	//			r = append(r, result.String())
	//		case "raw":
	//			r = append(r, result.Raw)
	//		case "simpleznumber":
	//
	//			r = append(r, result.Number)
	//		}
	//		if mapping.KeepJSON {
	//			r = append(r, result.Raw)
	//		} else {
	//			// or check the field from ABI to pick another type
	//			r = append(r, result.Str)
	//		}
	//	}
}
