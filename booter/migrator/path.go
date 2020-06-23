package migrator

import (
	"fmt"
	"path/filepath"
)

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
