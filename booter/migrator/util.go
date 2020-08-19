package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	rice "github.com/GeertJohan/go.rice"
	"github.com/eoscanada/eos-go"
)

var AN = eos.AN
var PN = eos.PN
var ActN = eos.ActN

func TN(in string) eos.TableName { return eos.TableName(in) }
func SN(in string) eos.ScopeName { return eos.ScopeName(in) }

func readBoxFile(box *rice.Box, filename string) ([]byte, error) {
	f, err := box.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open migration %q: %w", filename, err)
	}
	defer f.Close()
	cnt, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read box file %q content: %w", filename, err)
	}
	return cnt, nil
}

func writeJSONFile(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(v)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	return !info.IsDir()
}

func mustExtractIndexNumber(tableName string) (table string, index uint64) {
	name, err := eos.StringToName(tableName)
	if err != nil {
		panic(fmt.Sprintf("unable to convert table name %q to uint64: %s", name, err))
	}
	// The last 4 bits of a tableName represents the count of the index
	return eos.NameToString(name & 0xfffffffffffffff0), (name & 0x0f)
}

func mustCreateIndexTable(tableName string, indexId uint64) (table string) {
	name, err := eos.StringToName(tableName)
	if err != nil {
		panic(fmt.Sprintf("unable to convert table name %q to uint64: %s", name, err))
	}
	return eos.NameToString((name & 0xfffffffffffffff0) | (indexId & 0x0f))
}

func findTableDefInABI(abi *eos.ABI, table eos.TableName) *eos.TableDef {
	for _, t := range abi.Tables {
		if t.Name == table {
			return &t
		}
	}
	return nil
}

var primKeyEntropyFunc = func() uint64 {
	return uint64(rand.Intn(20) + 2)
}

func mustIncrementPrimKey(primKey string) eos.Name {
	// stringToName can never panic
	i, err := eos.StringToName(primKey)
	if err != nil {
		panic(fmt.Sprintf("unable to convert table primary key %q to uint64: %s", primKey, err))
	}

	return eos.Name(eos.NameToString(i + primKeyEntropyFunc()))
}

func s(str string) *string {
	return &str
}
