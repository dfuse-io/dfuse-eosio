package migrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ReadContractList(dataDir string) ([]string, error) {
	path := ContractListPath(dataDir)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read contract list: %w", err)
	}
	defer file.Close()

	var contracts []string

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&contracts)
	if err != nil {
		return nil, fmt.Errorf("unable decode contract %q list: %w", path, err)
	}
	return contracts, nil
}

func ContractListPath(dataDir string) string {
	return filepath.Join(dataDir, "contracts.json")
}

func nestedPath(parentPath string, entityName string) string {
	if len(entityName) <= 2 {
		return filepath.Join(parentPath, entityName)
	} else if len(entityName) <= 4 {
		return filepath.Join(parentPath, entityName[0:2], entityName)
	} else {
		return filepath.Join(parentPath, entityName[0:2], entityName[2:4], entityName)
	}
}

func newAccountPath(dataDir string, account string) (string, error) {
	if len(account) == 0 {
		return "", fmt.Errorf("an empty account name")
	}
	return nestedPath(dataDir, account), nil
}
