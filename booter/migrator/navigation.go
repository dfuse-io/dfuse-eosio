package migrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (i *importer) retrieveAccounts(newAccountFunc func(account *Account) error) (out []*Account, err error) {
	seenAccounts := map[string]int{}
	err = filepath.Walk(i.dataDir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("no files found")
		}
		if shouldSkip(info) {
			return filepath.SkipDir
		} else if isAccount(info) {
			acctName := accountFromAccountPath(path)
			if _, found := seenAccounts[acctName]; found {
				return nil
			}
			acc, err := newAccount(i.dataDir, acctName)
			acc.SetLogger(i.logger)
			if err != nil {
				return fmt.Errorf("unable to create account %q: %w", acctName, err)

			}
			out = append(out, acc)
			seenAccounts[acctName] = len(out) - 1
			return newAccountFunc(acc)
		} else if isContract(info) {
			acctName := accountFromAccountPath(path)
			if index, found := seenAccounts[acctName]; found {
				out[index].hasCode = true
			} else {
				acc, err := newAccount(i.dataDir, acctName)
				acc.SetLogger(i.logger)
				if err != nil {
					return fmt.Errorf("unable to create account %q: %w", acctName, err)

				}
				acc.hasCode = true
				out = append(out, acc)
				seenAccounts[acctName] = len(out) - 1
				return newAccountFunc(acc)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to walk through all accounts: %w", err)
	}
	return out, nil
}

func walkScopes(dataDir string, f func(scope string) error) {
	filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if isScope(info) {
			return f(getScopeName(path))
		}
		return nil
	})
}

func isAccount(file os.FileInfo) bool {
	return file.Name() == "account.json"
}

func isContract(file os.FileInfo) bool {
	return (file.Name() == "abi.json") ||
		(file.Name() == "code.wasm")
}

func isScope(file os.FileInfo) bool {
	return file.Name() == "rows.json"
}

func shouldSkip(file os.FileInfo) bool {
	return (file.IsDir()) && (file.Name() == "tables")
}

func getScopeName(path string) string {
	chunks := strings.Split(path, "/")
	encodedScope := chunks[len(chunks)-2]
	return decodeName(encodedScope)
}
