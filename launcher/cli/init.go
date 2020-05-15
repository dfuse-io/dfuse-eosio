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

var initCmd = &cobra.Command{Use: "init", Short: "Initializes dfuse's local environment", RunE: dfuseInitE}

func init() {
	RootCmd.AddCommand(initCmd)
}

func dfuseInitE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	configFile := viper.GetString("global-config-file")
	userLog.Debug("starting init", zap.String("config-file", configFile))

	maybeCheckNodeosVersion()

	runProducer, err := askProducer()
	if err != nil {
		return err
	}

	toRun := []string{"all", "-sqlsync"}
	if !runProducer {
		toRun = append(toRun, "-node-manager")
	}

	apps := launcher.ParseAppsFromArgs(toRun)
	conf := &launcher.DfuseConfig{}
	conf.Start.Args = apps
	conf.Start.Flags = map[string]string{}

	if runProducer {
		userLog.Printf("")

		// FIXME: would we create an `eosc-vault` ?
		if err := os.MkdirAll("./producer", 0755); err != nil {
			return fmt.Errorf("mkdir producer: %s", err)
		}

		userLog.Printf("Writing 'producer/config.ini'")
		if err = ioutil.WriteFile("./producer/config.ini", []byte(producerLocalConfigIni), 0600); err != nil {
			return fmt.Errorf("writing ./producer/config.ini file: %s", err)
		}

		userLog.Printf("Writing 'producer/genesis.json'")
		if err = ioutil.WriteFile("./producer/genesis.json", []byte(localGenesisJSON), 0644); err != nil {
			return fmt.Errorf("writing ./producer/genesis.json file: %s", err)
		}

		if err := os.MkdirAll("./mindreader", 0755); err != nil {
			return fmt.Errorf("mkdir mindreader: %s", err)
		}

		userLog.Printf("Writing 'mindreader/config.ini'")
		if err = ioutil.WriteFile("./mindreader/config.ini", []byte(mindreaderLocalConfigIni), 0644); err != nil {
			return fmt.Errorf("writing mindreader/config.ini file: %s", err)
		}

		userLog.Printf("Writing 'mindreader/genesis.json'")
		if err = ioutil.WriteFile("./mindreader/genesis.json", []byte(localGenesisJSON), 0644); err != nil {
			return fmt.Errorf("writing ./mindreader/genesis.json file: %s", err)
		}
	} else {
		peers, err := askPeers()
		if err != nil {
			return err
		}

		userLog.Printf("")

		if err := os.MkdirAll("./mindreader", 0755); err != nil {
			return fmt.Errorf("mkdir mindreader: %s", err)
		}

		userLog.Printf("Writing 'mindreader/config.ini'")
		mindreaderConfig := fmt.Sprintf(mindreaderRemoteConfigIniFormat, peersListConfigEntry(peers))
		if err = ioutil.WriteFile("./mindreader/config.ini", []byte(mindreaderConfig), 0644); err != nil {
			return fmt.Errorf("writing mindreader/config.ini file: %s", err)
		}
	}

	configBytes, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	userLog.Printf("Writing config '%s'", strings.TrimPrefix(configFile, "./"))
	if err = ioutil.WriteFile(configFile, configBytes, 0644); err != nil {
		return fmt.Errorf("writing config file %s: %w", configFile, err)
	}

	if runProducer {
		userLog.Printf("")
		userLog.Printf("Here the key pair controlling 'eosio' to interact with your local chain:")
		userLog.Printf("")
		userLog.Printf("  Public Key:  EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV")
		userLog.Printf("  Private Key: 5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3")
	} else {
		userLog.Printf("")
		userLog.Printf("IMPORANT: Move the remote network's 'genesis.json' file in './mindreader' directory")
	}

	userLog.Printf("")
	userLog.Printf("Initialization completed, to kickstart your environment run:")
	userLog.Printf("")
	userLog.Printf("  dfuseeos start")

	return nil
}

func askPeers() (peers []string, err error) {
	peers = viper.GetStringSlice("peer")
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

func askProducer() (bool, error) {
	userLog.Printf(`dfuse for EOSIO can run a local test node configured for block production,
similar to what you use in development, with a clean blank chain and no contracts.

Alternatively, dfuse for EOSIO can connect to an already existing network.`)

	prompt := promptui.Prompt{
		Label:     "Do you want dfuse for EOSIO to run a producing node for you",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		return false, fmt.Errorf("unable to ask if we run producing node: %w", err)
	}

	return strings.ToLower(result) == "y", nil
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
