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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	_ "github.com/dfuse-io/dfuse-eosio/eosdb/kv"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	dmeshClient "github.com/dfuse-io/dmesh/client"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var startCmd = &cobra.Command{Use: "start", Short: "Starts `dfuse for EOSIO` services all at once", RunE: dfuseStartE, Args: cobra.ArbitraryArgs}

func init() {
	RootCmd.AddCommand(startCmd)
}

func dfuseStartE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("dfuseeos binary started", zap.String("data_dir", dataDir))

	configFile := viper.GetString("global-config-file")
	userLog.Printf("Starting dfuse for EOSIO with config file '%s'", configFile)

	config := &launcher.DfuseConfig{}
	if configFile != "" {
		config, err = launcher.ReadConfig(configFile)
		if err != nil {
			userLog.Error(fmt.Sprintf("Error reading config file. Did you 'dfuseeos init' ?  Error: %s", err))
			return nil
		}
	}

	dataDirAbs, err := filepath.Abs(dataDir)
	if err != nil {
		userLog.Error("Unable to setup directory structure")
		return nil
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

	modules := &launcher.RuntimeModules{
		SearchDmeshClient: meshClient,
	}

	err = bstream.ValidateRegistry()
	if err != nil {
		userLog.Error("Protocol specific hooks not configured correctly", zap.Error(err))
		os.Exit(1)
	}

	launch := launcher.NewLauncher(config, modules)
	userLog.Debug("launcher created")

	apps := launcher.ParseAppsFromArgs(args)
	if len(args) == 0 {
		apps = launcher.ParseAppsFromArgs(config.Start.Args)
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
		userLog.Error("unable to launch", zap.Error(err))
		os.Exit(1)
	}

	printWelcomeMessage()

	signalHandler := derr.SetupSignalHandler(0 * time.Second)
	select {
	case <-signalHandler:
		userLog.Printf("Received termination signal, quitting")
	case <-launch.Terminating():
		if launch.FirstShutdownAppError == nil {
			userLog.Printf("Application %s triggered a Clean Shutdown, quitting", launch.FirstShutdownAppName)
		} else {
			userLog.Printf("Application %s shutdown unexpectedly, quitting", launch.FirstShutdownAppName)
			userLog.Printf("One of the applications shutdown unexpectedly, quitting")
			err = errors.New("unexpected termination")
		}
	}

	// all sub apps will be shut down by launcher when dfuse shut down
	go launch.Shutdown(nil)

	// wait for all sub apps to terminate
	launch.WaitForTermination()

	userLog.Printf("Goodbye")

	// At this point, everything is terminated, if we got an error
	// we exit right away with status code 1. If we let the error go
	// up on Cobra, it prints the error message.
	if err != nil {
		os.Exit(1)
	}

	return
}

func printWelcomeMessage() {
	message := strings.TrimLeft(`
Your instance should be ready in a few seconds, here some relevant links:

  Dashboard:        http://localhost%s

  Explorer & APIs:  http://localhost%s
  GraphiQL:         http://localhost%s/graphiql
`, "\n")

	userLog.Printf(message, DashboardHTTPListenAddr, APIProxyHTTPListenAddr, APIProxyHTTPListenAddr)
}

func containsApp(apps []string, searchedApp string) bool {
	for _, app := range apps {
		if app == searchedApp {
			return true
		}
	}

	return false
}
