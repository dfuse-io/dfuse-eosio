package sqlsync

import "github.com/tidwall/gjson"

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

func init() {
	r := Row{}
	abiDecoded := abi.DecodeStruct("sasset", binaryFromFluxDB)
	for _, mapping := range tables {
		result := gjson.ParseBytes(abiDecoded, mapping.ChainField)
		switch mapping.Type {
		case "string":
			r = append(r, result.String())
		case "raw":
			r = append(r, result.Raw)
		case "simpleznumber":

			r = append(r, result.Number)
		}
		if mapping.KeepJSON {
			r = append(r, result.Raw)
		} else {
			// or check the field from ABI to pick another type
			r = append(r, result.Str)
		}
	}
	SimpleAssetsMappings = map[string]Mappings{
		"sassets": Mappings{
			Mapping{"id", "id", "string"},
			Mapping{"owner", "owner", "string"},
			Mapping{"author", "author", "string"},
			Mapping{"category", "category", "string"},
			Mapping{"idata", "idata", "string"},
			Mapping{"mdata", "mdata", "string"},
			Mapping{"container", "container", "raw"},
			Mapping{"containerf", "containerf", "raw"},
		},
	}
}
