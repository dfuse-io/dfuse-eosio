package launcher

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var DfuseConfig map[string]*DfuseCommandConfig

type DfuseCommandConfig struct {
	Args  []string          `json:"args"`
	Flags map[string]string `json:"flags"`
}

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
