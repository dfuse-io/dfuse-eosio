package launcher

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Configuration extracted from the `dfuse.yaml` file. User-driven.
type BoxConfig struct {
	// Either GenesisJSON or GenesisFile
	GenesisJSON string `yaml:"genesis_json"`
	GenesisFile string `yaml:"genesis_file,omitempty"`

	RunProducer         bool   `yaml:"run_producer"`
	GeneratedPublicKey  string `yaml:"generated_public_key,omitempty"`
	GeneratedPrivateKey string `yaml:"generated_private_key,omitempty"`
	ProducerConfigIni   string `yaml:"producer_config_ini,omitempty"`
	ProducerNodeVersion string `yaml:"producer_node_version,omitempty"`

	ReaderConfigIni   string `yaml:"reader_config_ini"`
	ReaderNodeVersion string `yaml:"reader_node_version"`
	Version           string `yaml:"version"` // to determine if you need to dfuseeos init again
}

// Load reads a YAML config, and returns the raw JSON plus a
// top-level Config object. Use the raw JSON form to provide to the
// different plugins and apps for them to load their config.
func ReadConfig(filename string) (conf *BoxConfig, err error) {
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlBytes, &conf)
	if err != nil {
		return nil, fmt.Errorf("reading json: %s", err)
	}

	if conf.GenesisFile != "" {
		genesis, err := ioutil.ReadFile(conf.GenesisFile)
		if err != nil {
			return nil, fmt.Errorf("reading genesis file: %s", err)
		}

		conf.GenesisJSON = string(genesis)
	}

	return conf, nil
}
