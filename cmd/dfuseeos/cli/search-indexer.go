package cli

import (
	"fmt"

	"github.com/streamingfast/bstream"
	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	indexerApp "github.com/streamingfast/search/app/indexer"
)

func init() {
	// Search Indexer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-indexer",
		Title:       "Search indexer",
		Description: "Indexes transactions for search",
		MetricsID:   "indexer",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/search/(indexer|app/indexer).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("search-indexer-grpc-listen-addr", IndexerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-indexer-http-listen-addr", IndexerHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().Bool("search-indexer-enable-upload", true, "Upload merged indexes to the --indexes-store")
			cmd.Flags().Bool("search-indexer-delete-after-upload", true, "Delete local indexes after uploading them")
			cmd.Flags().Int("search-indexer-start-block", 0, "Start indexing from block num")
			cmd.Flags().Uint("search-indexer-stop-block", 0, "Stop indexing at block num")
			cmd.Flags().Bool("search-indexer-enable-batch-mode", false, "Enabled the indexer in batch mode with a start & stop block")
			cmd.Flags().Bool("search-indexer-verbose", false, "Verbose logging")
			cmd.Flags().Bool("search-indexer-enable-index-truncation", false, "Enable index truncation, requires a relative --start-block (negative number)")
			cmd.Flags().Uint64("search-indexer-shard-size", 200, "Number of blocks to store in a given Bleve index")
			cmd.Flags().String("search-indexer-writable-path", "{dfuse-data-dir}/search/indexer", "Writable base path for storing index files")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {

			mapper, err := eosSearch.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create block mapper: %w", err)
			}

			eosSearch.RegisterHandlers(mapper.IndexedTerms())

			dfuseDataDir := runtime.AbsDataDir
			blocksStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))
			blockstreamAddr := viper.GetString("common-blockstream-addr")
			tracker := runtime.Tracker.Clone()
			tracker.AddGetter(bstream.NetworkLIBTarget, bstream.NetworkLIBBlockRefGetter(blockstreamAddr))

			dataDir := runtime.AbsDataDir
			return indexerApp.New(&indexerApp.Config{
				HTTPListenAddr:        viper.GetString("search-indexer-http-listen-addr"),
				GRPCListenAddr:        viper.GetString("search-indexer-grpc-listen-addr"),
				BlockstreamAddr:       blockstreamAddr,
				ShardSize:             viper.GetUint64("search-indexer-shard-size"),
				StartBlock:            int64(viper.GetInt("search-indexer-start-block")),
				StopBlock:             viper.GetUint64("search-indexer-stop-block"),
				IsVerbose:             viper.GetBool("search-indexer-verbose"),
				EnableBatchMode:       viper.GetBool("search-indexer-enable-batch-mode"),
				EnableUpload:          viper.GetBool("search-indexer-enable-upload"),
				DeleteAfterUpload:     viper.GetBool("search-indexer-delete-after-upload"),
				EnableIndexTruncation: viper.GetBool("search-indexer-enable-index-truncation"),
				WritablePath:          mustReplaceDataDir(dataDir, viper.GetString("search-indexer-writable-path")),
				IndicesStoreURL:       mustReplaceDataDir(dataDir, viper.GetString("search-common-indices-store-url")),
				BlocksStoreURL:        blocksStoreURL,
			}, &indexerApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
				BlockMapper: mapper,
				Tracker:     tracker,
			}), nil
		},
	})
}
