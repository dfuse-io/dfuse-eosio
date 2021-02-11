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

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	_ "github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	"github.com/dfuse-io/dlauncher/launcher"
	dmeshClient "github.com/dfuse-io/dmesh/client"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/netkv"
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
	dataDir := viper.GetString("global-data-dir")
	userLog.Debug("dfuseeos binary started", zap.String("data_dir", dataDir))

	configFile := viper.GetString("global-config-file")
	userLog.Printf("Starting dfuse for EOSIO with config file '%s'", configFile)

	err = Start(dataDir, args)
	if err != nil {
		return fmt.Errorf("unable to launch: %w", err)
	}

	// If an error occurred, saying Goodbye is not greate
	userLog.Printf("Goodbye")
	return
}

func Start(dataDir string, args []string) (err error) {
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

	blockFilter, err := filtering.NewBlockFilter(
		strings.Split(viper.GetString("common-include-filter-expr"), ";;;"),
		strings.Split(viper.GetString("common-exclude-filter-expr"), ";;;"),
		strings.Split(viper.GetString("common-system-actions-include-filter-expr"), ";;;"),
	)
	if err != nil {
		return fmt.Errorf("unable to create block filter: %w", err)
	}

	zlog.Info("configured block filter", zap.Stringer("block_filter", blockFilter))

	// Block meta & chain tracker
	var blockMeta pbblockmeta.BlockIDClient
	tracker := bstream.NewTracker(250)

	blockmetaAddr := viper.GetString("common-blockmeta-addr")
	if blockmetaAddr != "" {
		conn, err := dgrpc.NewInternalClient(blockmetaAddr)
		if err != nil {
			userLog.Warn("cannot get grpc connection to blockmeta, some services will not leverage it, expect rough edges", zap.Error(err), zap.String("blockmeta_addr", blockmetaAddr))
		} else {
			zlog.Info("adding blockmeta as a start block resolver in tracker")
			blockMeta = pbblockmeta.NewBlockIDClient(conn)
			tracker.AddResolver(pbblockmeta.StartBlockResolver(blockMeta))
		}
	}

	blocksStoreURL := mustReplaceDataDir(dataDirAbs, viper.GetString("common-blocks-store-url"))
	blocksStore, err := dstore.NewDBinStore(blocksStoreURL)
	if err != nil {
		userLog.Warn("cannot get setup blockstore, disabling this startBlockResolver", zap.Error(err), zap.String("blocks_store_url", blocksStoreURL))
	} else {
		zlog.Info("adding block store as a start block resolver in tracker")
		tracker.AddResolver(codec.BlockstoreStartBlockResolver(blocksStore))
	}

	modules := &launcher.Runtime{
		SearchDmeshClient: meshClient,
		BlockFilter:       blockFilter,
		BlockMeta:         blockMeta,
		AbsDataDir:        dataDirAbs,
		Tracker:           tracker,
	}

	err = bstream.ValidateRegistry()
	if err != nil {
		return fmt.Errorf("protocol specific hooks not configured correctly: %w", err)
	}

	launch := launcher.NewLauncher(modules)
	userLog.Debug("launcher created")

	runByDefault := func(file string) bool {
		return true
	}

	apps := launcher.ParseAppsFromArgs(args, runByDefault)
	if len(args) == 0 {
		apps = launcher.ParseAppsFromArgs(launcher.DfuseConfig["start"].Args, runByDefault)
	}

	if containsApp(apps, "mindreader") {
		maybeCheckNodeosVersion()
	}

	userLog.Printf("Launching applications: %s", strings.Join(apps, ","))
	if err = launch.Launch(apps); err != nil {
		return err
	}

	printWelcomeMessage(apps)

	signalHandler := derr.SetupSignalHandler(viper.GetDuration("common-system-shutdown-signal-delay"))
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

	format := "Your instance should be ready in a few seconds, here are some relevant links:\n"
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
