package launcher

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type DfuseConfig struct {
	Start struct {
		Args  []string          `json:"args"`
		Flags map[string]string `json:"flags"`
	} `json:"start"`
}

// NewConfig creates a new initialized config structure. If `configFile` is non-empty and
// the file exists, initialize the config from the file content. If the config file is present
// and was read correctly, initialize Viper default values for those flags.
func NewConfig(configFile string, readOnlyIfExists bool) (conf *DfuseConfig, err error) {
	config := &DfuseConfig{}
	if configFile != "" {
		if shouldReadConfigFile(configFile, readOnlyIfExists) {
			userLog.Debug("reading config file", zap.String("file", configFile))
			config, err = ReadConfig(configFile)
			if err != nil {
				return nil, err
			}

			// Set default values for flags in `start`
			for k, v := range config.Start.Flags {
				viper.SetDefault(k, v)
			}
		}
	}

	return config, nil
}

// Load reads a YAML config, and returns the raw JSON plus a
// top-level Config object. Use the raw JSON form to provide to the
// different plugins and apps for them to load their config.
func ReadConfig(filename string) (conf *DfuseConfig, err error) {
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlBytes, &conf)
	if err != nil {
		return nil, fmt.Errorf("reading json: %s", err)
	}

	return conf, nil
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
