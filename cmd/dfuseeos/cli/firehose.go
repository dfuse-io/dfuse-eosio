package cli

import (
	"strings"

	"github.com/dfuse-io/bstream"
	firehoseApp "github.com/dfuse-io/dfuse-eosio/firehose/app/firehose"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-blocks-store-url and common-blockstream-addr",
		MetricsID:   "merged-filter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/firehose.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the firehose will listen")
			cmd.Flags().StringSlice("firehose-blocks-store-urls", nil, "If non-empty, overrides common-blocks-store-url with a list of blocks stores")
			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			tracker := runtime.Tracker.Clone()
			blockstreamAddr := viper.GetString("common-blockstream-addr")
			tracker.AddGetter(bstream.BlockStreamLIBTarget, bstream.StreamLIBBlockRefGetter(blockstreamAddr))

			firehoseBlocksStoreURLs := viper.GetStringSlice("firehose-blocks-store-urls")
			if len(firehoseBlocksStoreURLs) == 0 {
				firehoseBlocksStoreURLs = []string{viper.GetString("common-blocks-store-url")}
			} else if len(firehoseBlocksStoreURLs) == 1 && strings.Contains(firehoseBlocksStoreURLs[0], ",") {
				// Providing multiple elements from config doesn't work with `viper.GetStringSlice`, so let's also handle the case where a single element has separator
				firehoseBlocksStoreURLs = strings.Split(firehoseBlocksStoreURLs[0], ",")
			}

			for _, url := range firehoseBlocksStoreURLs {
				url = mustReplaceDataDir(dfuseDataDir, url)
			}

			return firehoseApp.New(&firehoseApp.Config{
				BlocksStoreURLs:         firehoseBlocksStoreURLs,
				UpstreamBlockStreamAddr: blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				AuthPlugin:              viper.GetString("common-auth-plugin"),
				MeteringPlugin:          viper.GetString("common-metering-plugin"),
			}, &firehoseApp.Modules{
				Tracker: tracker,
			}), nil
		},
	})
}
