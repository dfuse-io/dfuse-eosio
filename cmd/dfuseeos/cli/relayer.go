package cli

import (
	"time"

	"github.com/dfuse-io/dlauncher/launcher"
	relayerApp "github.com/dfuse-io/relayer/app/relayer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Relayer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "relayer",
		Title:       "Relayer",
		Description: "Serves blocks as a stream, with a buffer",
		MetricsID:   "relayer",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/relayer.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("relayer-grpc-listen-addr", RelayerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().StringSlice("relayer-source", []string{MindreaderGRPCAddr}, "List of Blockstream sources (mindreaders) to connect to for live block feeds (repeat flag as needed)")
			cmd.Flags().String("relayer-merger-addr", MergerServingAddr, "Address for grpc merger service")
			cmd.Flags().Int("relayer-buffer-size", 300, "Number of blocks that will be kept and sent immediately on connection")
			cmd.Flags().Duration("relayer-max-drift", 300*time.Second, "Max delay between live blocks before we die in hope of a better world")
			cmd.Flags().Uint64("relayer-min-start-offset", 120, "Number of blocks before HEAD where we want to start for faster buffer filling (missing blocks come from files/merger)")
			cmd.Flags().Duration("relayer-max-source-latency", 1*time.Minute, "Max latency tolerated to connect to a source")
			cmd.Flags().Duration("relayer-init-time", 1*time.Minute, "Time before we start looking for max drift")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			return relayerApp.New(&relayerApp.Config{
				SourcesAddr:      viper.GetStringSlice("relayer-source"),
				GRPCListenAddr:   viper.GetString("relayer-grpc-listen-addr"),
				MergerAddr:       viper.GetString("relayer-merger-addr"),
				BufferSize:       viper.GetInt("relayer-buffer-size"),
				MaxDrift:         viper.GetDuration("relayer-max-drift"),
				MaxSourceLatency: viper.GetDuration("relayer-max-source-latency"),
				InitTime:         viper.GetDuration("relayer-init-time"),
				MinStartOffset:   viper.GetUint64("relayer-min-start-offset"),
				SourceStoreURL:   mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
			}, &relayerApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
			}), nil
		},
	})
}
