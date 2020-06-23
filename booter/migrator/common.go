package migrator

import (
	"os"
	"path/filepath"
)

// TODO: hate the name of this should change it
type common struct {
	dataDir string
}

func (c *common) accountListPath() string {
	return filepath.Join(c.dataDir, "accounts.json")
}

func (c *common) createDataDir() error {
	return os.MkdirAll(c.dataDir, os.ModePerm)
}
