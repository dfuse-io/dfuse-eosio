package migrator

import (
	"os"
)

// TODO: hate the name of this should change it
type common struct {
	dataDir string
}

func (c *common) createDataDir() error {
	return os.MkdirAll(c.dataDir, os.ModePerm)
}
