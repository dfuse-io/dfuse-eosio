package cli

import (
	"github.com/dfuse-io/bstream"
	firehoseApp "github.com/dfuse-io/dfuse-eosio/firehose/app/firehose"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dlauncher/launcher"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks. Requires --common-include-filter-expr, --common-exclude-filter-expr and --common-blockmeta-addr",
		MetricsID:   "merged-filter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/firehose.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-blocks-store-url", MergedBlocksStoreURL, "Object Store from where we will read blocks. Blocks can be pre-filtered for performance.")
			cmd.Flags().String("firehose-blockstream-addr", RelayerServingAddr, "Address from which we pull real-time blocks. Can be relayer or filtered relayer")
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address from which we pull real-time blocks. Can be relayer or filtered relayer")
			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			//		&Config{
			//			BlocksStoreURL:          "gs://dfuseio-global-blocks-us/eos-mainnet/nospam-v3",
			//			UpstreamBlockStreamAddr: "relayer-filtered-v3.eos-mainnet.svc.cluster.local:9000",
			//			GRPCListenAddr:          ":9000",
			//			GRPCInsecure:            false,
			//			blockmetaAddr: "blockmeta-v3.eos-mainnet.svc.cluster.local:9000",

			//	a := New(conf, &Modules{
			//		Tracker: newTracker(conf.blockmetaAddr),
			//	})
			//
			//	if err := a.Run(); err != nil {
			//		zlog.Error("app failed", zap.Error(err))
			//	}
			//
			//	<-a.Terminated()
			//
			//	zlog.Info("terminated", zap.Error(a.Err()))
			//	zlog.Sync()
			blockmetaAddr := viper.GetString("common-blockmeta-addr")
			blockstreamAddr := viper.GetString("firehose-blockstream-addr")
			tracker := newTracker(blockmetaAddr, blockstreamAddr)

			return firehoseApp.New(&firehoseApp.Config{
				BlocksStoreURL:          mustReplaceDataDir(dfuseDataDir, viper.GetString("firehose-blocks-store-url")),
				UpstreamBlockStreamAddr: blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				BlockmetaAddr:           blockmetaAddr,
			}, &firehoseApp.Modules{
				Tracker: tracker,
			}), nil
		},
	})
}

func newTracker(blockmetaAddr, blockstreamAddr string) *bstream.Tracker {
	tracker := bstream.NewTracker(50)

	if blockmetaAddr != "" {
		conn, err := dgrpc.NewInternalClient(blockmetaAddr)
		if err != nil {
			zlog.Warn("cannot get grpc connection to blockmeta, disabling this startBlockResolver", zap.Error(err), zap.String("blockmeta_addr", blockmetaAddr))
		} else {
			blockmetaCli := pbblockmeta.NewBlockIDClient(conn)
			tracker.AddResolver(pbblockmeta.StartBlockResolver(blockmetaCli))
		}
	}

	if blockstreamAddr != "" {
		tracker.AddGetter(bstream.BlockStreamHeadTarget, bstream.StreamHeadBlockRefGetter(blockstreamAddr))
	}

	return tracker
}
