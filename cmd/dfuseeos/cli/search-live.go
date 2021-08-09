package cli

import (
	"fmt"
	"time"

	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	liveApp "github.com/streamingfast/search/app/live"
)

func init() {
	// Search Live
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-live",
		Title:       "Search live",
		Description: "Serves live search queries",
		MetricsID:   "live",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/search/(live|app/live).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {

			cmd.Flags().Uint32("search-live-tier-level", 100, "Level of the search tier")
			cmd.Flags().String("search-live-grpc-listen-addr", LiveServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-live-live-indices-path", "{dfuse-data-dir}/search/live", "Location for live indexes (ideally a ramdisk)")
			cmd.Flags().Int("search-live-truncation-threshold", 1, "number of available dmesh peers that should serve irreversible blocks before we truncate them from this backend's memory")
			cmd.Flags().Duration("search-live-realtime-tolerance", 1*time.Minute, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Duration("search-live-shutdown-delay", 0*time.Second, "On shutdown, time to wait before actually leaving, to try and drain connections")
			cmd.Flags().Uint64("search-live-start-block-drift-tolerance", 500, "allowed number of blocks between search archive and network head to get start block from the search archive")
			cmd.Flags().Uint64("search-live-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().Int("search-live-preprocessor-concurrent-threads", 8, "Number of thread used to run file source preprocessor function")
			cmd.Flags().Int("search-live-hub-channel-size", 1000, "Search live hub channel size")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			mapper, err := eosSearch.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create block mapper: %w", err)
			}

			eosSearch.RegisterHandlers(mapper.IndexedTerms())

			blockmetaAddr := viper.GetString("common-blockmeta-addr")
			blockstreamAddr := viper.GetString("common-blockstream-addr")

			return liveApp.New(&liveApp.Config{
				ServiceVersion:           viper.GetString("search-common-mesh-service-version"),
				TierLevel:                viper.GetUint32("search-live-tier-level"),
				GRPCListenAddr:           viper.GetString("search-live-grpc-listen-addr"),
				PublishInterval:          viper.GetDuration("search-common-mesh-publish-interval"),
				BlockmetaAddr:            blockmetaAddr,
				BlocksStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				BlockstreamAddr:          blockstreamAddr,
				HeadDelayTolerance:       viper.GetUint64("search-live-head-delay-tolerance"),
				StartBlockDriftTolerance: viper.GetUint64("search-live-start-block-drift-tolerance"),
				ShutdownDelay:            viper.GetDuration("search-live-shutdown-delay"),
				LiveIndexesPath:          mustReplaceDataDir(dfuseDataDir, viper.GetString("search-live-live-indices-path")),
				TruncationThreshold:      viper.GetInt("search-live-truncation-threshold"),
				RealtimeTolerance:        viper.GetDuration("search-live-realtime-tolerance"),
				HubChannelSize:           viper.GetInt("search-live-hub-channel-size"),
				PreProcConcurrentThreads: viper.GetInt("search-live-preprocessor-concurrent-threads"),
			}, &liveApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
				BlockMapper: mapper,
				Dmesh:       runtime.SearchDmeshClient,
				Tracker:     runtime.Tracker,
			}), nil
		},
	})
}
