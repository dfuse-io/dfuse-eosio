package cli

import (
	"math"

	trxdbLoaderApp "github.com/dfuse-io/dfuse-eosio/trxdb-loader/app/trxdb-loader"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "trxdb-loader",
		Title:       "DB loader",
		Description: "Main blocks and transactions database",
		MetricsID:   "trxdb-loader",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/trxdb-loader.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("trxdb-loader-processing-type", "live", "The actual processing type to perform, either `live`, `batch` or `patch`")
			cmd.Flags().Uint64("trxdb-loader-batch-size", 1, "Number of blocks batched together for database write")
			cmd.Flags().Uint64("trxdb-loader-start-block-num", 0, "[BATCH] Block number where we start processing")
			cmd.Flags().Uint64("trxdb-loader-stop-block-num", math.MaxUint32, "[BATCH] Block number where we stop processing")
			cmd.Flags().Uint64("trxdb-loader-num-blocks-before-start", 300, "[BATCH] Number of blocks to fetch before start block")
			cmd.Flags().String("trxdb-loader-http-listen-addr", KvdbHTTPServingAddr, "Listen address for /healthz endpoint")
			cmd.Flags().Int("trxdb-loader-parallel-file-download-count", 2, "Maximum number of files to download in parallel")
			cmd.Flags().Bool("trxdb-loader-allow-live-on-empty-table", true, "[LIVE] force pipeline creation if live request and table is empty")
			cmd.Flags().Bool("trxdb-loader-truncation-enabled", false, "Enables the creation of truncation marker on writes")
			cmd.Flags().Uint64("trxdb-loader-truncation-each", 0, "Truncates data that is older the defined X block number")
			cmd.Flags().Uint64("trxdb-loader-truncation-ttl", 0, "Truncates at every X block interval")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			return trxdbLoaderApp.New(&trxdbLoaderApp.Config{
				ChainID:                   viper.GetString("common-chain-id"),
				ProcessingType:            viper.GetString("trxdb-loader-processing-type"),
				BlockStoreURL:             mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				KvdbDsn:                   mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:           viper.GetString("common-blockstream-addr"),
				BatchSize:                 viper.GetUint64("trxdb-loader-batch-size"),
				StartBlockNum:             viper.GetUint64("trxdb-loader-start-block-num"),
				StopBlockNum:              viper.GetUint64("trxdb-loader-stop-block-num"),
				NumBlocksBeforeStart:      viper.GetUint64("trxdb-loader-num-blocks-before-start"),
				AllowLiveOnEmptyTable:     viper.GetBool("trxdb-loader-allow-live-on-empty-table"),
				HTTPListenAddr:            viper.GetString("trxdb-loader-http-listen-addr"),
				ParallelFileDownloadCount: viper.GetInt("trxdb-loader-parallel-file-download-count"),
				EnableTruncationMarker:    viper.GetBool("trxdb-loader-truncation-enabled"),
				TruncationTTL:             viper.GetUint64("trxdb-loader-truncation-ttl"),
				PurgerInterval:            viper.GetUint64("trxdb-loader-truncation-each"),
			}, &trxdbLoaderApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
			}), nil
		},
	})
}
