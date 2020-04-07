// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var initCmd = &cobra.Command{Use: "init", Short: "Initializes dfusebox's local environment", RunE: dfuseInitE}

func init() {
	initCmd.Flags().StringP("genesis-file", "g", "", "Valid only for 'remote' mode, the genesis file needed to sync with remote peer, if empty (default), asked interactively")
	initCmd.Flags().StringSliceP("peer", "p", nil, "Valid only for 'remote' mode, a peer needed to sync the remote chain")
}

func dfuseInitE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	configFile := viper.GetString("global-config-file")
	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("starting init", zap.String("config-file", configFile), zap.String("data_dir", dataDir))

	runProducer, err := askProducer()
	if err != nil {
		return err
	}

	newConfig := &core.BoxConfig{RunProducer: runProducer, Version: "v1"}
	newConfig.ReaderNodeVersion = "v2.0.3-dm"

	if newConfig.RunProducer {
		newConfig.ReaderConfigIni = mindreaderLocalConfigIni
		// TODO: Make these dynamic? Maybe we should ask the user instead? It seems providing the standard
		// userLog.Printf("NOTE: Generating private and public key for ad-hoc network.  These *will* end up in the generated configuration file.")
		newConfig.GeneratedPrivateKey = "5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3"
		newConfig.GeneratedPublicKey = "EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV"
		newConfig.ProducerConfigIni = managerLocalConfigIni
		newConfig.ProducerNodeVersion = "v2.0.3-dm"
		newConfig.GenesisJSON = localGenesisJSON
		newConfig.NodeosAPIAddr = "http://localhost:8888"
	} else {
		err := initRemoteBox(newConfig)
		if err != nil {
			return err
		}
	}

	configBytes, err := yaml.Marshal(newConfig)
	if err != nil {
		return err
	}

	configBase := string(configBytes)

	userLog.Printf("Writing file %s\n", configFile)
	if err = ioutil.WriteFile(configFile, []byte(configBase), 0600); err != nil {
		return fmt.Errorf("writing config file %s: %w", configFile, err)
	}

	userLog.Printf("Your dfusebox has been initialized correctly, run 'dfusebox start' to start your environment")
	return nil
}

func initRemoteBox(conf *core.BoxConfig) (err error) {
	genesisFilePath := viper.GetString("init-cmd-genesis-file")
	if genesisFilePath == "" {
		genesisFilePath, err = askGenesisPath()
		if err != nil {
			return fmt.Errorf("failed to request genesis file path: %w", err)
		}
	}

	if _, err := os.Stat(genesisFilePath); err != nil && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("the genesis file %q does not exist", genesisFilePath)
	}

	genesisContent, err := ioutil.ReadFile(genesisFilePath)
	if err != nil {
		return fmt.Errorf("unable to read genesis file %q: %w", genesisFilePath, err)
	}

	conf.GenesisJSON = string(genesisContent)

	peers, err := askPeers()
	if err != nil {
		return err
	}

	conf.ReaderConfigIni = fmt.Sprintf(mindreaderRemoteConfigIniFormat, peersListConfigEntry(peers))

	api, err := askAPI()
	if err != nil {
		return err
	}

	conf.NodeosAPIAddr = api

	return nil
}

func askPeers() (peers []string, err error) {
	peers = viper.GetStringSlice("init-cmd-peer")
	if len(peers) == 0 {
		for {
			peer, err := askPeer(len(peers) == 0)
			if err != nil {
				return nil, fmt.Errorf("failed to request peers: %w", err)
			}
			if peer == "" {
				break
			}
			peers = append(peers, peer)
		}
	}
	return
}

func peersListConfigEntry(peers []string) string {
	entries := make([]string, len(peers))
	for i, peer := range peers {
		entries[i] = fmt.Sprintf("p2p-peer-address = %s", peer)
	}

	return strings.Join(entries, "\n")
}

func mkdirAllFolders(folders *core.EOSFolderStructure) error {
	var directories []string
	directories = append(directories, folders.ConfigDirs()...)
	directories = append(directories, folders.DataDirs()...)

	for _, directory := range directories {
		err := os.MkdirAll(directory, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %q: %w", directory, err)
		}
	}

	return nil
}

func askProducer() (bool, error) {
	userLog.Printf(`dfusebox can run a local test node configured for block production,
similar to what you use in development, with a clean blank chain and no contracts.

Alternatively, dfusebox can connect to an already existing network.`)

	prompt := promptui.Prompt{
		Label:     "Do you want dfusebox to run a producing node for you",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		return false, fmt.Errorf("unable to ask if we run producing node: %w", err)
	}

	return strings.ToLower(result) == "y", nil
}

func askGenesisPath() (string, error) {
	validate := func(input string) error {
		if !fileExists(input) {
			return errors.New("Genesis file does not exist")
		}
		return nil
	}

	defaultPath := "genesis.json"

	prompt := promptui.Prompt{
		Label:    "Path to genesis file",
		Validate: validate,
		Default:  defaultPath,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func askAPI() (string, error) {
	validate := func(input string) error {
		if len(input) < 8 {
			return errors.New("Invalid peer api, should start with http:// and be a bit longer...")
		}
		return nil
	}

	defaultAPI := "http://127.0.0.1:8888"
	label := "API address to connect to (useful to get your chain head info)"

	prompt := promptui.Prompt{
		Label:    label,
		Validate: validate,
		Default:  defaultAPI,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func askPeer(first bool) (string, error) {
	validate := func(input string) error {
		//@TODO improve validation
		if !first && input == "" {
			return nil
		}
		if len(input) < 8 {
			return errors.New("Invalid peer address")
		}
		return nil
	}

	defaultPeer := "127.0.0.1:9876"
	label := "First peer to connect"
	if !first {
		label = "Add another peer? (leave blank to skip)"
		defaultPeer = ""
	}

	prompt := promptui.Prompt{
		Label:    label,
		Validate: validate,
		Default:  defaultPeer,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
