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
	"fmt"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	dgrpc.Verbosity = 2
}

func setupCmd(cmd *cobra.Command) error {
	cmd.SilenceUsage = true

	cmds := extractCmd(cmd)
	subCommand := cmds[len(cmds)-1]

	forceConfigOn := []*cobra.Command{StartCmd}
	logToFileOn := []*cobra.Command{StartCmd}

	if configFile := viper.GetString("global-config-file"); configFile != "" {
		exists, err := fileExists(configFile)
		if err != nil {
			return fmt.Errorf("unable to check if config file exists: %w", err)
		}

		if !exists && isMatchingCommand(cmds, forceConfigOn) {
			return fmt.Errorf("Config file %q not found. Did you 'dfuseeos init'?", configFile)
		}

		if exists {
			if err := launcher.LoadConfigFile(configFile); err != nil {
				return fmt.Errorf("unable to read config file %q: %w", configFile, err)
			}
		}
	}

	subconf := launcher.DfuseConfig[subCommand]
	if subconf != nil {
		for k, v := range subconf.Flags {
			validFlag := false
			if _, ok := allFlags["global-"+k]; ok {
				viper.SetDefault("global-"+k, v)
				validFlag = true
			}
			if _, ok := allFlags[k]; ok {
				viper.SetDefault(k, v)
				validFlag = true
			}
			if !validFlag {
				return fmt.Errorf("invalid flag %s in config file under command %s", k, subCommand)
			}
		}
	}

	launcher.SetupLogger(&launcher.LoggingOptions{
		WorkingDir:    viper.GetString("global-data-dir"),
		Verbosity:     viper.GetInt("global-verbose"),
		LogFormat:     viper.GetString("global-log-format"),
		LogToFile:     isMatchingCommand(cmds, logToFileOn) && viper.GetBool("global-log-to-file"),
		LogListenAddr: viper.GetString("global-log-level-switcher-listen-addr"),
	})
	launcher.SetupTracing()
	launcher.SetupAnalyticsMetrics(viper.GetString("global-metrics-listen-addr"), viper.GetString("global-pprof-listen-addr"))

	return nil
}

func isMatchingCommand(cmds []string, runSetupOn []*cobra.Command) bool {
	for _, c := range runSetupOn {
		baseChunks := extractCmd(c)
		if strings.Join(cmds, ".") == strings.Join(baseChunks, ".") {
			return true
		}
	}
	return false
}

func extractCmd(cmd *cobra.Command) []string {
	cmds := []string{}
	for {
		if cmd == nil {
			break
		}
		cmds = append(cmds, cmd.Use)
		cmd = cmd.Parent()
	}

	out := make([]string, len(cmds))

	for itr, v := range cmds {
		newIndex := len(cmds) - 1 - itr
		out[newIndex] = v
	}
	return out
}

func wrapRunE(root *cobra.Command) {
	original := root.RunE

	root.RunE = nil
	root.Run = func(cmd *cobra.Command, args []string) {
		err := original(cmd, args)
		if err != nil {
			userLog.Error(err.Error())
		}
	}

	for _, child := range root.Commands() {
		wrapRunE(child)
	}
}

func fileExists(file string) (bool, error) {
	stat, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return !stat.IsDir(), nil
}
