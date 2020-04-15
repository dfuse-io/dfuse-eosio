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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	_ "github.com/dfuse-io/dfuse-eosio/eosdb/kv"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/dfuse-eosio/metrics"
	dmeshClient "github.com/dfuse-io/dmesh/client"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var startCmd = &cobra.Command{Use: "start", Short: "Starts `dfuse for EOSIO` services all at once", RunE: dfuseStartE}

func init() {
	startCmd.Flags().Bool("send-to-bigquery", false, "Send data to big query")
}

func dfuseStartE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	configFile := viper.GetString("global-config-file")
	userLog.Printf("Starting dfuse for EOSIO '%s'", configFile)

	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("dfuse single binary started", zap.String("data_dir", dataDir))

	boxConfig, err := launcher.ReadConfig(configFile)
	if err != nil {
		userLog.Error("dfuse for EOSIO not initialized. Please run 'dfuseeos init'")
		return nil
	}

	if boxConfig.Version != "v1" {
		userLog.Error("dfuse for EOSIO not initialized with this version. Please run 'dfuseeos init'")
		return nil
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

	modules := &launcher.RuntimeModules{
		SearchDmeshClient: dmeshClient.NewLocalClient(),
		MetricManager:     metrics.NewManager("http://localhost:9102/metrics", []string{"head_block_time_drift", "head_block_number"}, 5*time.Second, launcher.GetMetricAppMeta()),
	}

	err = bstream.ValidateRegistry()
	if err != nil {
		userLog.Error("Protocol specific hooks not configured correctly", zap.Error(err))
		os.Exit(1)
	}

	launcher := launcher.NewLauncher(boxConfig, modules)
	userLog.Debug("launcher created")

	apps := []string{}

	// Producer node
	if boxConfig.RunProducer {
		apps = append(apps, "manager")
	}
	//apps = append(apps, "mindreader", "relayer", "merger", "kvdb-loader", "fluxdb", "abicodec", "eosws")
	apps = append(apps, "mindreader", "relayer", "merger", "kvdb-loader", "fluxdb", "indexer", "blockmeta", "abicodec", "router", "archive", "live", "dgraphql", "eosws", "dashboard", "eosq")

	userLog.Printf("Launching all applications...")
	err = launcher.Launch(apps)
	if err != nil {
		userLog.Error("unable to launch", zap.Error(err))
		os.Exit(1)
	}

	go modules.MetricManager.Launch()

	printWelcomeMessage()

	signalHandler := derr.SetupSignalHandler(0 * time.Second)
	select {
	case <-signalHandler:
		userLog.Printf("Received termination signal, quitting")
	case <-launcher.Terminating():
		userLog.Printf("One of the applications shutdown unexpectedly, quitting")
		err = errors.New("unexpected termination")
	}

	// all sub apps will be shut down by launcher when dfuse shut down
	go launcher.Shutdown(nil)

	// wait for all sub apps to terminate
	launcher.WaitForTermination()

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

		Dashboard: http://localhost%s
		GraphiQL: http://localhost%s/graphiql
		Eosq: http://localhost%s
`, "\n")

	userLog.Printf(message, DashboardHTTPListenAddr, EosqHTTPServingAddr)

}
