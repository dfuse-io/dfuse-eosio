package launcher

import (
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var DfuseConfig map[string]*DfuseCommandConfig

type DfuseCommandConfig struct {
	Args  []string          `json:"args"`
	Flags map[string]string `json:"flags"`
}

// Load reads a YAML config, and sets the global DfuseConfig variable
// Use the raw JSON form to provide to the
// different plugins and apps for them to load their config.
func LoadConfigFile(filename string) (err error) {
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlBytes, &DfuseConfig)
	if err != nil {
		return fmt.Errorf("reading json: %s", err)
	}

	return nil
}

func shouldReadConfigFile(configFile string, readOnlyIfExists bool) bool {
	if !readOnlyIfExists {
		return true
	}

	return fileExists(configFile)
}

func fileExists(file string) bool {
	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		userLog.Debug("unable to check if file exists", zap.String("file", file))
		return false
	}

	return !stat.IsDir()
}
