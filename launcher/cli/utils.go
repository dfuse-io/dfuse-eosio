package cli

import (
	"path/filepath"
	"strings"
)

func buildStoreURL(dataDir, storeURL string) string {
	parts := strings.Split(storeURL, "://")

	if len(parts) > 1 {
		return storeURL
	}

	if strings.HasPrefix(parts[0], "/") {
		// absolute path
		return storeURL
	}

	return filepath.Join(dataDir, storeURL)
}
