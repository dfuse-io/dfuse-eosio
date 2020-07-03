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

	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/viper"
)

func init() {
	dgrpc.Verbosity = 2
}

func setup(subCommand string) error {
	if subCommand != "init" {
		if configFile := viper.GetString("global-config-file"); configFile != "" {
			if err := launcher.LoadConfigFile(configFile); err != nil {
				return fmt.Errorf("Error reading config file. Did you 'dfuseeos init' ?  Error: %w", err)
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

	}

	launcher.SetupLogger(&launcher.LoggingOptions{
		WorkingDir:    viper.GetString("global-data-dir"),
		Verbosity:     viper.GetInt("global-verbose"),
		LogFormat:     viper.GetString("global-log-format"),
		LogToFile:     viper.GetBool("global-log-to-file"),
		LogListenAddr: viper.GetString("global-log-level-switcher-listen-addr"),
	})
	launcher.SetupTracing()
	launcher.SetupAnalyticsMetrics()

	// The zlog are wrapped, they need to be re-configured with newly set base instance to work correctly
	userLog.ReconfigureReference()

	return nil

}
