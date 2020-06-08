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
	launcher2 "github.com/dfuse-io/dfuse-box/launcher"
	"path/filepath"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	_ "github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	dmeshClient "github.com/dfuse-io/dmesh/client"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var StartCmd = &cobra.Command{Use: "start", Short: "Starts `dfuse for EOSIO` services all at once", RunE: dfuseStartE, Args: cobra.ArbitraryArgs}

func init() {
	RootCmd.AddCommand(StartCmd)
}

func dfuseStartE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("dfuseeos binary started", zap.String("data_dir", dataDir))

	configFile := viper.GetString("global-config-file")
	userLog.Printf("Starting dfuse for EOSIO with config file '%s'", configFile)

	err = Start(configFile, dataDir, args)
	if err != nil {
		return fmt.Errorf("unable to launch: %w", err)
	}

	// If an error occurred, saying Goodbye is not greate
	userLog.Printf("Goodbye")
	return
}

func Start(configFile string, dataDir string, args []string) (err error) {

	config := &launcher2.DfuseConfig{}
	if configFile != "" {
		config, err = launcher2.ReadConfig(configFile)
		if err != nil {
			return fmt.Errorf("Error reading config file. Did you 'dfuseeos init' ?  Error: %w", err)
		}
	}

	dataDirAbs, err := filepath.Abs(dataDir)
	if err != nil {
		return fmt.Errorf("unable to setup directory structure: %w", err)
	}

	// TODO: directories are created in the app init funcs... but this does not belong to a specific application
	err = makeDirs([]string{dataDirAbs})
	if err != nil {
		return err
	}

	meshClient, err := dmeshClient.New(viper.GetString("search-common-mesh-dsn"))
	if err != nil {
		return fmt.Errorf("unable to create dmesh client: %w", err)
	}

	modules := &launcher2.RuntimeModules{
		SearchDmeshClient: meshClient,
	}

	err = bstream.ValidateRegistry()
	if err != nil {
		return fmt.Errorf("protocol specific hooks not configured correctly: %w", err)
	}

	launch := launcher2.NewLauncher(config, modules)
	userLog.Debug("launcher created")

	apps := launcher2.ParseAppsFromArgs(args)
	if len(args) == 0 {
		apps = launcher2.ParseAppsFromArgs(config.Start.Args)
	}

	// Set default values for flags in `start`
	for k, v := range config.Start.Flags {
		viper.SetDefault(k, v)
	}

	if containsApp(apps, "mindreader") {
		maybeCheckNodeosVersion()
	}

	userLog.Printf("Launching applications: %s", strings.Join(apps, ","))
	if err = launch.Launch(apps); err != nil {
		return err
	}

	printWelcomeMessage(apps)

	signalHandler := derr.SetupSignalHandler(0 * time.Second)
	select {
	case <-signalHandler:
		userLog.Printf("Received termination signal, quitting")
		go launch.Close()
	case appID := <-launch.Terminating():
		if launch.Err() == nil {
			userLog.Printf("Application %s triggered a clean shutdown, quitting", appID)
		} else {
			userLog.Printf("Application %s shutdown unexpectedly, quitting", appID)
			return launch.Err()
		}
	}

	launch.WaitForTermination()

	return
}

func printWelcomeMessage(apps []string) {
	hasDashboard := containsApp(apps, "dashboard")
	hasAPIProxy := containsApp(apps, "apiproxy")
	if !hasDashboard && !hasAPIProxy {
		// No welcome message to print, advanced usage
		return
	}

	format := "Your instance should be ready in a few seconds, here some relevant links:\n"
	var formatArgs []interface{}

	if hasDashboard {
		format += "\n"
		format += "  Dashboard:        http://localhost%s\n"
		formatArgs = append(formatArgs, DashboardHTTPListenAddr)
	}

	if hasAPIProxy {
		format += "\n"
		format += "  Explorer & APIs:  http://localhost%s\n"
		format += "  GraphiQL:         http://localhost%s/graphiql\n"
		formatArgs = append(formatArgs, APIProxyHTTPListenAddr, APIProxyHTTPListenAddr)
	}

	userLog.Printf(format, formatArgs...)
}

func containsApp(apps []string, searchedApp string) bool {
	for _, app := range apps {
		if app == searchedApp {
			return true
		}
	}

	return false
}
