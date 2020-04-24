package cli

import (
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func replaceDataDir(dataDir, in string) string {
	//fmt.Println("REPLACE", dataDir, in)
	in = strings.Replace(in, "{dfuse-data-dir}", dataDir, -1)
	return in
}

func mkdirStorePathIfLocal(storeURL string) (err error) {
	userLog.Debug("creating directory and its parent(s)", zap.String("directory", storeURL))
	if dirs := getDirsToMake(storeURL); len(dirs) > 0 {
		err = makeDirs(dirs)
	}
	return
}

func getDirsToMake(storeURL string) []string {
	parts := strings.Split(storeURL, "://")
	if len(parts) > 1 {
		if parts[0] != "file" {
			// Not a local store, nothing to do
			return nil
		}
		storeURL = parts[1]
	}

	// Some of the store URL are actually a file directly, let's try our best to cope for that case
	filename := filepath.Base(storeURL)
	if strings.Contains(filename, ".") {
		storeURL = filepath.Dir(storeURL)
	}

	// If we reach here, it's a local store path
	return []string{storeURL}

}
