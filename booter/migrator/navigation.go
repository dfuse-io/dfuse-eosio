package migrator

import (
	"os"
	"path/filepath"
	"strings"
)

func walkContracts(dataDir string, f func(contract string) error) {
	seenContracts := map[string]bool{}
	filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if shouldSkip(info) {
			return filepath.SkipDir
		} else if isContract(info) {
			chunks := strings.Split(path, "/")
			contract := chunks[len(chunks)-2]
			if _, found := seenContracts[contract]; !found {
				seenContracts[contract] = true
				return f(contract)
			}
		}
		return nil
	})
}

func walkScopes(dataDir string, f func(scope string) error) {
	filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if isScope(info) {
			chunks := strings.Split(path, "/")
			scope := chunks[len(chunks)-2]
			return f(scope)
		}
		return nil
	})
}

func isContract(file os.FileInfo) bool {
	return (file.Name() == "abi.json") ||
		(file.Name() == "code.wasm")
}

func isScope(file os.FileInfo) bool {
	return (file.Name() == "rows.json")
}

func shouldSkip(file os.FileInfo) bool {
	return (file.IsDir()) && (file.Name() == "tables")
}
