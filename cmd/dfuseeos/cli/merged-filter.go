package cli

import (
	mergedFilterApp "github.com/dfuse-io/dfuse-eosio/merged-filter/app/merged-filter"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "merged-filter",
		Title:       "Merged Filter",
		Description: "Consumed merged block files, filters them and produced smaller merged blocks files. Requires --common-include-filter-expr, --common-exclude-filter-expr and --common-blockstream-addr",
		MetricsID:   "merged-filter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/merged-filter.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("merged-filter-truncation-enabled", true, "[NOT IMPLEMENTED] Will delete filtered merged blocks files after the truncation window period.")
			cmd.Flags().Uint64("merged-filter-truncation-window", 0, "Number of blocks to keep history of filtered merged blocks. Used as start position when no filtered files exist.")
			cmd.Flags().String("merged-filter-destination-blocks-store-url", FilteredBlocksStoreURL, "Object Store where to write filtered blocks store.  Sources from --common-blocks-store-url.")

			cmd.Flags().Bool("merged-filter-batch-mode", false, "Use this to explicitly set start/stop block numbers for processing and ignore current chain head")
			cmd.Flags().Uint64("merged-filter-batch-start-block", 0, "When running in batch mode, block number (rounded) where to start processing")
			cmd.Flags().Uint64("merged-filter-batch-stop-block", 0, "When running in batch mode, block number (rounded) where to stop processing")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) (err error) {
			dfuseDataDir := runtime.AbsDataDir

			if err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("merged-filter-destination-blocks-store-url"))); err != nil {
				return
			}

			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			return mergedFilterApp.New(&mergedFilterApp.Config{
				DestBlocksStoreURL:             mustReplaceDataDir(dfuseDataDir, viper.GetString("merged-filter-destination-blocks-store-url")),
				SourceBlocksStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				TruncationEnabled:              viper.GetBool("merged-filter-truncation-enabled"),
				TruncationWindow:               viper.GetUint64("merged-filter-truncation-window"),
				BatchMode:                      viper.GetBool("merged-filter-batch-mode"),
				BatchStartBlock:                viper.GetUint64("merged-filter-batch-start-block"),
				BatchStopBlock:                 viper.GetUint64("merged-filter-batch-stop-block"),
				IncludeFilterExpr:              viper.GetString("common-include-filter-expr"),
				ExcludeFilterExpr:              viper.GetString("common-exclude-filter-expr"),
				SystemActionsIncludeFilterExpr: viper.GetString("common-system-actions-include-filter-expr"),
				BlockstreamAddr:                viper.GetString("common-blockstream-addr"),
			}), nil
		},
	})
}
