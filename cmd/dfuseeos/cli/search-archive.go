package cli

import (
	"fmt"
	"time"

	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	archiveApp "github.com/streamingfast/search/app/archive"
)

func init() {
	// Search Archive
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-archive",
		Title:       "Search archive",
		Description: "Serves historical search queries",
		MetricsID:   "archive",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/search/(archive|app/archive).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			// These flags are scoped to search, since they are shared betwween search-router, search-live, search-archive, etc....
			cmd.Flags().String("search-archive-grpc-listen-addr", ArchiveServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-archive-http-listen-addr", ArchiveHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("search-archive-memcache-addr", "", "Empty results cache's memcache server address")
			cmd.Flags().Bool("search-archive-enable-empty-results-cache", false, "Enable roaring-bitmap-based empty results caching")
			cmd.Flags().Uint32("search-archive-tier-level", 50, "Level of the search tier")
			cmd.Flags().Bool("search-archive-enable-moving-tail", false, "Enable moving tail, requires a relative --start-block (negative number)")
			cmd.Flags().Uint64("search-archive-shard-size", 200, "Number of blocks to store in a given Bleve index")
			cmd.Flags().Int("search-archive-start-block", 0, "Start at given block num, the initial sync and polling")
			cmd.Flags().Uint("search-archive-stop-block", 0, "Stop before given block num, the initial sync and polling")
			cmd.Flags().Bool("search-archive-index-polling", true, "Populate local indexes using indexes store polling.")
			cmd.Flags().Bool("search-archive-sync-from-storage", false, "Download missing indexes from --indexes-store before starting")
			cmd.Flags().Int("search-archive-sync-max-indexes", 100000, "Maximum number of indexes to sync. On production, use a very large number.")
			cmd.Flags().Int("search-archive-indices-dl-threads", 1, "Number of indices files to download from the GS input store and decompress in parallel. In prod, use large value like 20.")
			cmd.Flags().Int("search-archive-max-query-threads", 10, "Number of end-user query parallel threads to query 5K-blocks indexes")
			cmd.Flags().Duration("search-archive-shutdown-delay", 0*time.Second, "On shutdown, time to wait before actually leaving, to try and drain connections")
			cmd.Flags().String("search-archive-warmup-filepath", "", "Optional filename containing queries to warm-up the search")
			cmd.Flags().String("search-archive-writable-path", "{dfuse-data-dir}/search/archiver", "Writable base path for storing index files")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			indexedTerms, err := eosSearch.NewIndexedTerms(viper.GetString("search-common-indexed-terms"))
			if err != nil {
				return nil, fmt.Errorf("unable to indexed terms: %w", err)
			}

			eosSearch.RegisterHandlers(indexedTerms)

			return archiveApp.New(&archiveApp.Config{
				BlockmetaAddr:           viper.GetString("common-blockmeta-addr"),
				MemcacheAddr:            viper.GetString("search-archive-memcache-addr"),
				EnableEmptyResultsCache: viper.GetBool("search-archive-enable-empty-results-cache"),
				ServiceVersion:          viper.GetString("search-common-mesh-service-version"),
				TierLevel:               viper.GetUint32("search-archive-tier-level"),
				GRPCListenAddr:          viper.GetString("search-archive-grpc-listen-addr"),
				HTTPListenAddr:          viper.GetString("search-archive-http-listen-addr"),
				PublishInterval:         viper.GetDuration("search-common-mesh-publish-interval"),
				EnableMovingTail:        viper.GetBool("search-archive-enable-moving-tail"),
				ShardSize:               viper.GetUint64("search-archive-shard-size"),
				StartBlock:              viper.GetInt64("search-archive-start-block"),
				StopBlock:               viper.GetUint64("search-archive-stop-block"),
				IndexPolling:            viper.GetBool("search-archive-index-polling"),
				SyncFromStore:           viper.GetBool("search-archive-sync-from-storage"),
				SyncMaxIndexes:          viper.GetInt("search-archive-sync-max-indexes"),
				IndicesDLThreads:        viper.GetInt("search-archive-indices-dl-threads"),
				NumQueryThreads:         viper.GetInt("search-archive-max-query-threads"),
				ShutdownDelay:           viper.GetDuration("search-archive-shutdown-delay"),
				WarmupFilepath:          viper.GetString("search-archive-warmup-filepath"),
				IndexesStoreURL:         mustReplaceDataDir(dfuseDataDir, viper.GetString("search-common-indices-store-url")),
				IndexesPath:             mustReplaceDataDir(dfuseDataDir, viper.GetString("search-archive-writable-path")),
			}, &archiveApp.Modules{
				Dmesh: runtime.SearchDmeshClient,
			}), nil
		},
	})
}
