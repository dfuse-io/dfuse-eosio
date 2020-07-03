package cli

import (
	"path/filepath"

	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "fluxdb",
		Title:       "FluxDB",
		Description: "Temporal chain state store",
		MetricsID:   "fluxdb",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/fluxdb.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("fluxdb-enable-server-mode", true, "Enables flux server mode, launch a server")
			cmd.Flags().Bool("fluxdb-enable-inject-mode", true, "Enables flux inject mode, writes into its database")
			cmd.Flags().Bool("fluxdb-enable-reproc-sharder-mode", false, "[BATCH] Enables flux reproc shard mode, exclusive option, cannot be set if either server, injector or reproc-injector mode is set")
			cmd.Flags().Bool("fluxdb-enable-reproc-injector-mode", false, "[BATCH] Enables flux reproc injector mode, exclusive option, cannot be set if either server, injector or reproc-shard mode is set")
			cmd.Flags().Bool("fluxdb-enable-pipeline", true, "Enables fluxdb without a blocks pipeline, useful for running a development server (**do not** use this in prod)")
			cmd.Flags().String("fluxdb-statedb-dsn", FluxDSN, "kvdb connection string to State database")
			cmd.Flags().Int("fluxdb-max-threads", 2, "Number of threads of parallel processing")
			cmd.Flags().String("fluxdb-http-listen-addr", FluxDBServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("fluxdb-reproc-shard-store-url", "file://{dfuse-data-dir}/statedb/reproc-shards", "[BATCH] Storage url where all reproc shard write requests should be written to")
			cmd.Flags().Uint64("fluxdb-reproc-shard-count", 0, "[BATCH] Number of shards to split in (in 'reproc-sharder' mode), or join (in 'reproc-injector' mode)")
			cmd.Flags().Uint64("fluxdb-reproc-shard-start-block-num", 0, "[BATCH] Start processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("fluxdb-reproc-shard-stop-block-num", 0, "[BATCH] Stop processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("fluxdb-reproc-injector-shard-index", 0, "[BATCH] Index of the shard to perform injection for, should be lower than shard-count")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}
			return fluxdbApp.New(&fluxdbApp.Config{
				EnableServerMode:           viper.GetBool("fluxdb-enable-server-mode"),
				EnableInjectMode:           viper.GetBool("fluxdb-enable-inject-mode"),
				EnableReprocSharderMode:    viper.GetBool("fluxdb-enable-reproc-sharder-mode"),
				EnableReprocInjectorMode:   viper.GetBool("fluxdb-enable-reproc-injector-mode"),
				EnablePipeline:             viper.GetBool("fluxdb-enable-pipeline"),
				StoreDSN:                   mustReplaceDataDir(absDataDir, viper.GetString("fluxdb-statedb-dsn")),
				BlockStreamAddr:            viper.GetString("common-blockstream-addr"),
				BlockStoreURL:              mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				ThreadsNum:                 viper.GetInt("fluxdb-max-threads"),
				HTTPListenAddr:             viper.GetString("fluxdb-http-listen-addr"),
				ReprocShardStoreURL:        mustReplaceDataDir(dfuseDataDir, viper.GetString("fluxdb-reproc-shard-store-url")),
				ReprocShardCount:           viper.GetUint64("fluxdb-reproc-shard-count"),
				ReprocSharderStartBlockNum: viper.GetUint64("fluxdb-reproc-shard-start-block-num"),
				ReprocSharderStopBlockNum:  viper.GetUint64("fluxdb-reproc-shard-stop-block-num"),
				ReprocInjectorShardIndex:   viper.GetUint64("fluxdb-reproc-injector-shard-index"),
			}), nil
		},
	})
}
