package cli

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/dstore"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	indexerApp "github.com/dfuse-io/search/app/indexer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	// Search Indexer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-indexer",
		Title:       "Search indexer",
		Description: "Indexes transactions for search",
		MetricsID:   "indexer",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(indexer|app/indexer).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("search-indexer-grpc-listen-addr", IndexerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-indexer-http-listen-addr", IndexerHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().Bool("search-indexer-enable-upload", true, "Upload merged indexes to the --indexes-store")
			cmd.Flags().Bool("search-indexer-delete-after-upload", true, "Delete local indexes after uploading them")
			cmd.Flags().Int("search-indexer-start-block", 0, "Start indexing from block num")
			cmd.Flags().Uint("search-indexer-stop-block", 0, "Stop indexing at block num")
			cmd.Flags().Bool("search-indexer-enable-batch-mode", false, "Enabled the indexer in batch mode with a start & stoip block")
			cmd.Flags().Bool("search-indexer-verbose", false, "Verbose logging")
			cmd.Flags().Bool("search-indexer-enable-index-truncation", false, "Enable index truncation, requires a relative --start-block (negative number)")
			cmd.Flags().Uint64("search-indexer-shard-size", 200, "Number of blocks to store in a given Bleve index")
			cmd.Flags().String("search-indexer-writable-path", "{dfuse-data-dir}/search/indexer", "Writable base path for storing index files")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			mapper, err := eosSearch.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create block mapper: %w", err)
			}

			filter, err := filtering.NewBlockFilter(viper.GetString("common-include-filter-expr"), viper.GetString("common-exclude-filter-expr"))
			if err != nil {
				return nil, fmt.Errorf("unable to create block filter: %w", err)
			}

			eosSearch.RegisterHandlers(mapper.IndexedTerms())

			var startBlockResolvers []bstream.StartBlockResolver
			blockmetaAddr := viper.GetString("common-blockmeta-addr")
			if blockmetaAddr != "" {
				conn, err := dgrpc.NewInternalClient(blockmetaAddr)
				if err != nil {
					userLog.Warn("cannot get grpc connection to blockmeta, disabling this startBlockResolver for search indexer", zap.Error(err), zap.String("blockmeta_addr", blockmetaAddr))
				} else {
					blockmetaCli := pbblockmeta.NewBlockIDClient(conn)
					startBlockResolvers = append(startBlockResolvers, bstream.StartBlockResolverFunc(pbblockmeta.StartBlockResolver(blockmetaCli)))
				}
			}

			blocksStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))
			blocksStore, err := dstore.NewDBinStore(blocksStoreURL)
			if err != nil {
				userLog.Warn("cannot get setup blockstore, disabling this startBlockResolver for search indexer", zap.Error(err), zap.String("blocksStoreURL", blocksStoreURL))
			} else {
				startBlockResolvers = append(startBlockResolvers, codec.BlockstoreStartBlockResolver(blocksStore))
			}
			if len(startBlockResolvers) == 0 {
				return nil, fmt.Errorf("no StartBlockResolver could be set for search indexer")
			}

			return indexerApp.New(&indexerApp.Config{
				HTTPListenAddr:        viper.GetString("search-indexer-http-listen-addr"),
				GRPCListenAddr:        viper.GetString("search-indexer-grpc-listen-addr"),
				BlockstreamAddr:       viper.GetString("common-blockstream-addr"),
				ShardSize:             viper.GetUint64("search-indexer-shard-size"),
				StartBlock:            int64(viper.GetInt("search-indexer-start-block")),
				StopBlock:             viper.GetUint64("search-indexer-stop-block"),
				IsVerbose:             viper.GetBool("search-indexer-verbose"),
				EnableBatchMode:       viper.GetBool("search-indexer-enable-batch-mode"),
				EnableUpload:          viper.GetBool("search-indexer-enable-upload"),
				DeleteAfterUpload:     viper.GetBool("search-indexer-delete-after-upload"),
				EnableIndexTruncation: viper.GetBool("search-indexer-enable-index-truncation"),
				WritablePath:          mustReplaceDataDir(dfuseDataDir, viper.GetString("search-indexer-writable-path")),
				IndicesStoreURL:       mustReplaceDataDir(dfuseDataDir, viper.GetString("search-common-indices-store-url")),
				BlocksStoreURL:        blocksStoreURL,
			}, &indexerApp.Modules{
				BlockFilter:        filter.TransformInPlace,
				BlockMapper:        mapper,
				StartBlockResolver: bstream.ParallelStartResolver(startBlockResolvers, -1),
			}), nil
		},
	})
}
