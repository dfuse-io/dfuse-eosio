package migrator

import (
	"fmt"
	"path/filepath"
	"strings"
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

	account = encodeName(account)

	return nestedPath(dataDir, account), nil
}

func accountFromAccountPath(path string) string {
	chunks := strings.Split(path, "/")
	encodedAccountName := chunks[len(chunks)-2]
	return decodeName(encodedAccountName)
}
