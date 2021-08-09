package cli

import (
	"strings"
	"time"

	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	relayerApp "github.com/streamingfast/relayer/app/relayer"
)

func init() {
	// Relayer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "relayer",
		Title:       "Relayer",
		Description: "Serves blocks as a stream, with a buffer",
		MetricsID:   "relayer",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/relayer.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("relayer-grpc-listen-addr", RelayerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().StringSlice("relayer-source", []string{MindreaderGRPCAddr}, "List of Blockstream sources (mindreaders) to connect to for live block feeds (repeat flag as needed)")
			cmd.Flags().Int("relayer-source-request-burst", 90, "Block burst requested by relayer (useful when chaining relayers together, because normally a mindreader won't have a block buffer)")
			cmd.Flags().String("relayer-merger-addr", MergerServingAddr, "Address for grpc merger service")
			cmd.Flags().Int("relayer-buffer-size", 350, "Number of blocks that will be kept and sent immediately on connection")
			cmd.Flags().Uint64("relayer-min-start-offset", 120, "Number of blocks before HEAD where we want to start for faster buffer filling (missing blocks come from files/merger)")
			cmd.Flags().Duration("relayer-max-source-latency", 10*time.Minute, "Max latency tolerated to connect to a source")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			sourcesAddr := viper.GetStringSlice("relayer-source")
			if len(sourcesAddr) == 1 {
				sourcesAddr = strings.Split(sourcesAddr[0], ",")
			}

			return relayerApp.New(&relayerApp.Config{
				SourcesAddr:        sourcesAddr,
				GRPCListenAddr:     viper.GetString("relayer-grpc-listen-addr"),
				MergerAddr:         viper.GetString("relayer-merger-addr"),
				BufferSize:         viper.GetInt("relayer-buffer-size"),
				MaxSourceLatency:   viper.GetDuration("relayer-max-source-latency"),
				SourceRequestBurst: viper.GetInt("relayer-source-request-burst"),
				MinStartOffset:     viper.GetUint64("relayer-min-start-offset"),
				SourceStoreURL:     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
			}, &relayerApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
			}), nil
		},
	})
}
