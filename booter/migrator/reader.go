package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/eoscanada/eos-go"
)

func readCode(path string) (code []byte, err error) {
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read code for at %q: %w", path, err)
	}

	return cnt, nil
}

func readABI(path string) (abi *eos.ABI, abiCnt []byte, err error) {
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read ABI for at %q: %w", path, err)
	}

	err = json.Unmarshal(cnt, &abi)
	if err != nil {
		return nil, nil, fmt.Errorf("unable decode ABI at %q: %w", path, err)
	}

	return abi, cnt, nil
}

func readTableScopeRows(path string) ([]*tableRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read table scope rows at %q: %w", path, err)
	}
	defer file.Close()

	var rows []*tableRow

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rows)
	if err != nil {
		return nil, fmt.Errorf("unable decode rows tbl scope rows at %q: %w", path, err)
	}

	return rows, nil
}

func readTableScopeInfo(path string) (*tableScope, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read table scope info at %q: %w", path, err)
	}
	defer file.Close()

	var tblScope *tableScope

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tblScope)
	if err != nil {
		return nil, fmt.Errorf("unable decode rows tbl scope info at %q: %w", path, err)
	}

	return tblScope, nil
}
