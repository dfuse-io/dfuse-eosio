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
	"path/filepath"
	"strings"
	"time"

	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	_ "github.com/dfuse-io/bstream/codecs/deos"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/dfuse-eosio/metrics"
	dmeshClient "github.com/dfuse-io/dmesh/client"
	_ "github.com/dfuse-io/kvdb/eosdb/kv"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var startCmd = &cobra.Command{Use: "start", Short: "Starts dfusebox's services all at once", RunE: dfuseStartE}

func init() {
	startCmd.Flags().Bool("send-to-bigquery", false, "Send data to big query")
}

func dfuseStartE(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	configFile := viper.GetString("global-config-file")
	userLog.Printf("Starting dfusebox '%s'", configFile)

	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("dfuse single binary started", zap.String("data_dir", dataDir))

	nodeosPath := viper.GetString("global-nodeos-path")

	boxConfig, err := core.ReadConfig(configFile)
	if err != nil {
		userLog.Error("dfusebox not initialized. Please run 'dfusebox init'")
		return nil
	}

	if boxConfig.Version != "v1" {
		userLog.Error("dfusebox not initialized with this version. Please run 'dfusebox init'")
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

	// construct root config used by dfuse app, passed down to be used by all sub apps
	config := &core.RuntimeConfig{
		BoxConfig:                boxConfig,
		DmeshServiceVersion:      "v1",
		DmeshNamespace:           "local",
		NetworkID:                "eos-local",
		DataDir:                  dataDirAbs,
		StartBlock:               0,
		StopBlock:                0,
		NodeExecutable:           nodeosPath,
		ShardSize:                200,
		NodeosAPIAddr:            ":8888", // This is the address where the nodeos is serving its API. it is defined in config.ini
		MindreaderNodeosAPIAddr:  ":9888", // This is the address where the dm-nodeos is serving its API. it is defined in config.ini
		EosManagerHTTPAddr:       ":13008",
		EosMindreaderHTTPAddr:    ":13009",
		MindreaderGRPCAddr:       ":13010",
		RelayerServingAddr:       ":13011",
		MergerServingAddr:        ":13012",
		AbiServingAddr:           ":13013",
		TokenmetaServingAddr:     ":9001", // Not implemented yet, present for booting purposes, does not work!
		BlockmetaServingAddr:     ":13015",
		ArchiveServingAddr:       ":13016",
		LiveServingAddr:          ":13017",
		RouterServingAddr:        ":13018",
		DashboardHTTPListenAddr:  ":8080",
		DgraphqlHTTPServingAddr:  ":13019",
		DgraphqlGrpcServingAddr:  ":13020",
		DashboardGrpcServingAddr: ":13731",
		IndexerServingAddr:       ":13024",
		IndexerHTTPServingAddr:   ":13025",
		ArchiveHTTPServingAddr:   ":13026",
		RouterHTTPServingAddr:    ":13027",
		KvdbHTTPServingAddr:      ":13028",
		EoswsHTTPServingAddr:     ":13029",
		EosqHTTPServingAddress:   ":8081",
		FluxDBServingAddr:        ":13030",
		// TODO: clean this one up...
		//KvdbDSN: fmt.Sprintf("sqlite3://%s/kvdb_db.db?cache=shared&mode=memory&createTables=true", filepath.Join(dataDirAbs, "kvdb")),
		KvdbDSN: fmt.Sprintf("badger://%s/kvdb_badger.db?compression=zstd", filepath.Join(dataDirAbs, "kvdb")),
		//KvdbDSN:               "tikv://pd0:2379?keyPrefix=01000001", // 01 = kvdb, 000001 = eos-mainnet
		//KvdbDSN:               "bigkv://dev.dev/kvdb?createTable=true",
		Protocol:              pbbstream.Protocol_EOS,
		BootstrapDataURL:      "",
		NodeosTrustedProducer: "",
		NodeosShutdownDelay:   0 * time.Second,
		NodeosExtraArgs:       make([]string, 0),
	}

	modules := &core.RuntimeModules{
		SearchDmeshClient: dmeshClient.NewLocalClient(),
		MetricManager:     metrics.NewManager("http://localhost:9102/metrics", []string{"head_block_time_drift", "head_block_number"}, 5*time.Second, core.GetMetricAppMeta()),
	}

	launcher := core.NewLauncher(config, modules)
	userLog.Debug("launcher created")

	apps := []string{}

	// Producer node
	if config.BoxConfig.RunProducer {
		apps = append(apps, "manager")
	}
	apps = append(apps, "mindreader", "relayer", "merger", "kvdb-loader", "fluxdb", "indexer", "blockmeta", "abicodec", "router", "archive", "live", "dgraphql", "eosws", "dashboard", "eosq")
	// apps = append(apps, "mindreader", "relayer", "merger", "kvdb-loader", "fluxdb")

	userLog.Printf("Launching all applications...")
	launcher.Launch(apps)

	go modules.MetricManager.Launch()

	printWelcomeMessage(config)

	signalHandler := derr.SetupSignalHandler(0 * time.Second)
	select {
	case <-signalHandler:
		userLog.Printf("Received termination signal, quitting")
	case <-launcher.Terminating():
		userLog.Printf("One of the applications shutdown unexpectedly, quitting")
	}

	// all sub apps will be shut down by launcher when dfuse shut down
	go launcher.Shutdown(nil)

	// wait for all sub apps to terminate
	launcher.WaitForTermination()

	userLog.Printf("Goodbye")
	return
}

func printWelcomeMessage(config *core.RuntimeConfig) {
	message := strings.TrimLeft(`
Your instance should be ready in a few seconds, here some relevant links:

		Dashboard: http://localhost%s
		GraphiQL: http://localhost%s/graphiql
		Eosq: http://localhost%s
`, "\n")

	userLog.Printf(message, config.DashboardHTTPListenAddr, config.DashboardHTTPListenAddr, config.EosqHTTPServingAddress)

}
