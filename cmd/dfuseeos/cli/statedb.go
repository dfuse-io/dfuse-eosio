package cli

import (
	"strings"

	statedbApp "github.com/dfuse-io/dfuse-eosio/statedb/app/statedb"
	"github.com/dfuse-io/dlauncher/launcher"
	fluxdbApp "github.com/dfuse-io/fluxdb/app/fluxdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "statedb",
		Title:       "StateDB",
		Description: "Temporal chain state store",
		MetricsID:   "statedb",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/(fluxdb.*|dfuse-eosio/statedb.*)", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("statedb-enable-server-mode", true, "Enables flux server mode, launch a server")
			cmd.Flags().Bool("statedb-enable-inject-mode", true, "Enables flux inject mode, writes into its database")
			cmd.Flags().Bool("statedb-enable-reproc-sharder-mode", false, "[BATCH] Enables flux reproc shard mode, exclusive option, cannot be set if either server, injector or reproc-injector mode is set")
			cmd.Flags().Bool("statedb-enable-reproc-injector-mode", false, "[BATCH] Enables flux reproc injector mode, exclusive option, cannot be set if either server, injector or reproc-shard mode is set")
			cmd.Flags().Bool("statedb-enable-pipeline", true, "Enables fluxdb without a blocks pipeline, useful for running a development server (**do not** use this in prod)")
			cmd.Flags().String("statedb-store-dsn", StateDBDSN, "kvdb connection string to State database")
			cmd.Flags().String("statedb-http-listen-addr", StateDBHTTPServingAddr, "Address to listen for incoming HTTP requests")
			cmd.Flags().String("statedb-grpc-listen-addr", StateDBGRPCServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("statedb-reproc-shard-store-url", "file://{dfuse-data-dir}/statedb/reproc-shards", "[BATCH] Storage url where all reproc shard write requests should be written to")
			cmd.Flags().String("statedb-reproc-shard-scratch-directory", "", "[BATCH] Provide a scratch directory where sharder while write each element composing a shard to a temporary file instead of holding everything in RAM, trade-off between I/O bound and RAM bound")
			cmd.Flags().Uint64("statedb-reproc-shard-count", 0, "[BATCH] Number of shards to split in (in 'reproc-sharder' mode), or join (in 'reproc-injector' mode)")
			cmd.Flags().Uint64("statedb-reproc-shard-start-block-num", 0, "[BATCH] Start processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("statedb-reproc-shard-stop-block-num", 0, "[BATCH] Stop processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("statedb-reproc-injector-shard-index", 0, "[BATCH] Index of the shard to perform injection for, should be lower than shard-count")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			scratchDirectory := viper.GetString("statedb-reproc-shard-scratch-directory")
			if scratchDirectory != "" {
				scratchDirectory = mustReplaceDataDir(dfuseDataDir, scratchDirectory)
				scratchDirectory = strings.TrimPrefix(scratchDirectory, "file://")
			}

			return statedbApp.New(&statedbApp.Config{
				Config: &fluxdbApp.Config{
					EnableServerMode:              viper.GetBool("statedb-enable-server-mode"),
					EnableInjectMode:              viper.GetBool("statedb-enable-inject-mode"),
					EnableReprocSharderMode:       viper.GetBool("statedb-enable-reproc-sharder-mode"),
					EnableReprocInjectorMode:      viper.GetBool("statedb-enable-reproc-injector-mode"),
					EnablePipeline:                viper.GetBool("statedb-enable-pipeline"),
					StoreDSN:                      mustReplaceDataDir(dfuseDataDir, viper.GetString("statedb-store-dsn")),
					BlockStreamAddr:               viper.GetString("common-blockstream-addr"),
					BlockStoreURL:                 mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
					ReprocShardStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("statedb-reproc-shard-store-url")),
					ReprocShardCount:              viper.GetUint64("statedb-reproc-shard-count"),
					ReprocSharderScratchDirectory: scratchDirectory,
					ReprocSharderStartBlockNum:    viper.GetUint64("statedb-reproc-shard-start-block-num"),
					ReprocSharderStopBlockNum:     viper.GetUint64("statedb-reproc-shard-stop-block-num"),
					ReprocInjectorShardIndex:      viper.GetUint64("statedb-reproc-injector-shard-index"),
				},

				HTTPListenAddr: viper.GetString("statedb-http-listen-addr"),
				GRPCListenAddr: viper.GetString("statedb-grpc-listen-addr"),
			}, &statedbApp.Modules{
				BlockFilter:        runtime.BlockFilter.TransformInPlace,
				StartBlockResolver: runtime.Tracker.ResolveStartBlock,
			}), nil
		},
	})
}
