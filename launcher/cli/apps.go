// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eoscanada/eos-go"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	"github.com/dfuse-io/bstream"
	_ "github.com/dfuse-io/dauth/authenticator/null" // register authenticator plugin
	_ "github.com/dfuse-io/dauth/ratelimiter/null"   // register ratelimiter plugin
	"github.com/dfuse-io/dfuse-box/dashboard"
	"github.com/dfuse-io/dfuse-box/launcher"
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dfuse-eosio/apiproxy"
	dblockmeta "github.com/dfuse-io/dfuse-eosio/blockmeta"
	"github.com/dfuse-io/dfuse-eosio/codec"
	dgraphqlEosio "github.com/dfuse-io/dfuse-eosio/dgraphql"
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq/app/eosq"
	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	trxdbLoaderApp "github.com/dfuse-io/dfuse-eosio/trxdb-loader/app/trxdb-loader"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	mergerApp "github.com/dfuse-io/merger/app/merger"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	relayerApp "github.com/dfuse-io/relayer/app/relayer"
	archiveApp "github.com/dfuse-io/search/app/archive"
	forkresolverApp "github.com/dfuse-io/search/app/forkresolver"
	indexerApp "github.com/dfuse-io/search/app/indexer"
	liveApp "github.com/dfuse-io/search/app/live"
	routerApp "github.com/dfuse-io/search/app/router"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {

	launcher.RegisterCommonFlags = func(cmd *cobra.Command) error {
		// Common stores configuration flags
		cmd.Flags().String("common-backup-store-url", PitreosURL, "[COMMON] Store URL (with prefix) where to read or write backups.")
		cmd.Flags().String("common-blocks-store-url", MergedBlocksStoreURL, "[COMMON] Store URL (with prefix) where to read/write. Used by: relayer, fluxdb, trxdb-loader, blockmeta, search-indexer, search-live, search-forkresolver, eosws")
		cmd.Flags().String("common-oneblock-store-url", OneBlockStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
		cmd.Flags().String("common-blockstream-addr", RelayerServingAddr, "gRPC endpoint to get real-time blocks. Used by: fluxdb, trxdb-loader, blockmeta, search-indexer, search-live, eosws (relayer uses its own --relayer-blockstream-addr)")

		// Network config
		cmd.Flags().String("common-network-id", NetworkID, "Short network identifier, for billing purposes (usually maps namespaces on deployments). Used by: dgraphql")
		cmd.Flags().String("common-chain-id", "", "Chain ID in hex. Used by: trxdb-loader (to reverse the signatures and extract public keys)") // TODO: eventually, pluck that from somewhere instead of asking for it here (!). You risk noticing its missing very late, and it'll require reprocessing if you want the pubkeys.

		// Authentication, metering and rate limiter plugins
		cmd.Flags().String("common-auth-plugin", "null://", "Auth plugin URI, see dfuse-io/dauth repository")
		cmd.Flags().String("common-metering-plugin", "null://", "Metering plugin URI, see dfuse-io/dmetering repository")
		cmd.Flags().String("common-ratelimiter-plugin", "null://", "Rate Limiter plugin URI, see dfuse-io/dauth repository")

		// Database connection strings
		cmd.Flags().String("common-trxdb-dsn", TrxdbDSN, "kvdb connection string to trxdb database. Used by: trxdb-loader, abicodec, eosws, dgraphql")

		// Service addresses
		cmd.Flags().String("common-search-addr", RouterServingAddr, "gRPC endpoint to reach the Search Router. Used by: abicodec, eosws, dgraphql")
		cmd.Flags().String("common-blockmeta-addr", BlockmetaServingAddr, "gRPC endpoint to reach the Blockmeta. Used by: search-indexer, search-router, search-live, eosws, dgraphql")

		// Search flags
		// Register common search flags once for all the services
		cmd.Flags().String("search-common-mesh-store-addr", "", "[COMMON] Address of the backing etcd cluster for mesh service discovery.")
		cmd.Flags().String("search-common-mesh-dsn", DmeshDSN, "[COMMON] Dmesh DSN, supports local & etcd")
		cmd.Flags().String("search-common-mesh-service-version", DmeshServiceVersion, "[COMMON] Dmesh service version (v1)")
		cmd.Flags().Duration("search-common-mesh-publish-interval", 0*time.Second, "[COMMON] How often does search archive poll dmesh")
		cmd.Flags().String("search-common-action-filter-on-expr", "", "[COMMON] CEL program to whitelist actions to index. See https://github.com/dfuse-io/dfuse-eosio/blob/develop/search/README.md")
		cmd.Flags().String("search-common-action-filter-out-expr", "account == 'eidosonecoin' || receiver == 'eidosonecoin' || (account == 'eosio.token' && (data.to == 'eidosonecoin' || data.from == 'eidosonecoin'))", "[COMMON] CEL program to blacklist actions to index. These 2 options are used by search indexer, live and forkresolver.")
		cmd.Flags().String("search-common-dfuse-events-action-name", "", "[COMMON] The dfuse Events action name to intercept")
		cmd.Flags().Bool("search-common-dfuse-events-unrestricted", false, "[COMMON] Flag to disable all restrictions of dfuse Events specialize indexing, for example for a private deployment")
		cmd.Flags().String("search-common-indices-store-url", IndicesStoreURL, "[COMMON] Indices path to read or write index shards Used by: search-indexer, search-archiver.")

		return nil
	}

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
			cmd.Flags().Int("relayer-buffer-size", 300, "number of blocks that will be kept and sent immediately on connection")
			cmd.Flags().Duration("relayer-max-drift", 300*time.Second, "max delay between live blocks before we die in hope of a better world")
			cmd.Flags().Uint64("relayer-min-start-offset", 120, "number of blocks before HEAD where we want to start for faster buffer filling (missing blocks come from files/merger)")
			cmd.Flags().Duration("relayer-max-source-latency", 1*time.Minute, "max latency tolerated to connect to a source")
			cmd.Flags().Duration("relayer-init-time", 1*time.Minute, "time before we start looking for max drift")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
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
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "merger",
		Title:       "Merger",
		Description: "Produces merged block files from single-block files",
		MetricsID:   "merger",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/merger.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Duration("merger-time-between-store-lookups", 10*time.Second, "delay between polling source store (higher for remote storage)")
			cmd.Flags().String("merger-grpc-listen-addr", MergerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("merger-process-live-blocks", true, "Ignore --start-.. and --stop-.. blocks, and process only live blocks")
			cmd.Flags().Uint64("merger-start-block-num", 0, "FOR REPROCESSING: if >= 0, Set the block number where we should start processing")
			cmd.Flags().Uint64("merger-stop-block-num", 0, "FOR REPROCESSING: if > 0, Set the block number where we should stop processing (and stop the process)")
			cmd.Flags().String("merger-progress-filename", "", "FOR REPROCESSING: If non-empty, will update progress in this file and start right there on restart")
			cmd.Flags().Uint64("merger-minimal-block-num", 0, "FOR LIVE: Set the minimal block number where we should start looking at the destination storage to figure out where to start")
			cmd.Flags().Duration("merger-writers-leeway", 10*time.Second, "how long we wait after seeing the upper boundary, to ensure that we get as many blocks as possible in a bundle")
			cmd.Flags().String("merger-seen-blocks-file", "{dfuse-data-dir}/merger/merger.seen.gob", "file to save to / load from the map of 'seen blocks'")
			cmd.Flags().Uint64("merger-max-fixable-fork", 10000, "after that number of blocks, a block belonging to another fork will be discarded (DELETED depending on flagDeleteBlocksBefore) instead of being inserted in last bundle")
			cmd.Flags().Bool("merger-delete-blocks-before", true, "Enable deletion of oneblock files when prior to the currently processed bundle (to avoid long file listings)")
			return nil
		},
		// FIXME: Lots of config value construction is duplicated across InitFunc and FactoryFunc, how to streamline that
		//        and avoid the duplication? Note that this duplicate happens in many other apps, we might need to re-think our
		//        init flow and call init after the factory and giving it the instantiated app...
		InitFunc: func(modules *launcher.RuntimeModules) error {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return err
			}
			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")))
			if err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				StorageOneBlockFilesPath:     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")),
				TimeBetweenStoreLookups:      viper.GetDuration("merger-time-between-store-lookups"),
				GRPCListenAddr:               viper.GetString("merger-grpc-listen-addr"),
				Live:                         viper.GetBool("merger-process-live-blocks"),
				StartBlockNum:                viper.GetUint64("merger-start-block-num"),
				StopBlockNum:                 viper.GetUint64("merger-stop-block-num"),
				ProgressFilename:             viper.GetString("merger-progress-filename"),
				MinimalBlockNum:              viper.GetUint64("merger-minimal-block-num"),
				WritersLeewayDuration:        viper.GetDuration("merger-writers-leeway"),
				SeenBlocksFile:               mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")),
				MaxFixableFork:               viper.GetUint64("merger-max-fixable-fork"),
				DeleteBlocksBefore:           viper.GetBool("merger-delete-blocks-before"),
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "fluxdb",
		Title:       "FluxDB",
		Description: "Temporal chain state store",
		MetricsID:   "fluxdb",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/fluxdb.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("fluxdb-enable-server-mode", true, "Enables flux server mode, launch a server")
			cmd.Flags().Bool("fluxdb-enable-inject-mode", true, "Enables flux inject mode, writes into its database")
			cmd.Flags().Bool("fluxdb-enable-reproc-sharder-mode", false, "[BATCH] Enables flux reproc shard mode, exclusive option, cannot be set if either server, injector or reproc-injector mode is set")
			cmd.Flags().Bool("fluxdb-enable-reproc-injector-mode", false, "[BATCH] Enables flux reproc injector mode, exclusive option, cannot be set if either server, injector or reproc-shard mode is set")
			cmd.Flags().Bool("fluxdb-enable-pipeline", true, "Enables fluxdb without a blocks pipeline, useful for running a development server (**do not** use this in prod)")
			cmd.Flags().String("fluxdb-statedb-dsn", FluxDSN, "kvdb connection string to State database")
			cmd.Flags().Int("fluxdb-max-threads", 2, "Number of threads of parallel processing")
			cmd.Flags().String("fluxdb-http-listen-addr", FluxDBServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("fluxdb-reproc-shard-store-url", "file://{dfuse-data-dir}/statedb/reproc-shards", "[BATCH] Storage url where all reproc shard write requests should be written to")
			cmd.Flags().Uint64("fluxdb-reproc-shard-count", 0, "[BATCH] Number of shards to split in (in 'reproc-sharder' mode), or join (in 'reproc-injector' mode)")
			cmd.Flags().Uint64("fluxdb-reproc-shard-start-block-num", 0, "[BATCH] Start processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("fluxdb-reproc-shard-stop-block-num", 0, "[BATCH] Stop processing block logs at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("fluxdb-reproc-injector-shard-index", 0, "[BATCH] Index of the shard to perform injection for, should be lower than shard-count")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}
			return fluxdbApp.New(&fluxdbApp.Config{
				EnableServerMode:           viper.GetBool("fluxdb-enable-server-mode"),
				EnableInjectMode:           viper.GetBool("fluxdb-enable-inject-mode"),
				EnableReprocSharderMode:    viper.GetBool("fluxdb-enable-reproc-sharder-mode"),
				EnableReprocInjectorMode:   viper.GetBool("fluxdb-enable-reproc-injector-mode"),
				EnablePipeline:             viper.GetBool("fluxdb-enable-pipeline"),
				StoreDSN:                   mustReplaceDataDir(absDataDir, viper.GetString("fluxdb-statedb-dsn")),
				BlockStreamAddr:            viper.GetString("common-blockstream-addr"),
				BlockStoreURL:              mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				ThreadsNum:                 viper.GetInt("fluxdb-max-threads"),
				HTTPListenAddr:             viper.GetString("fluxdb-http-listen-addr"),
				ReprocShardStoreURL:        mustReplaceDataDir(dfuseDataDir, viper.GetString("fluxdb-reproc-shard-store-url")),
				ReprocShardCount:           viper.GetUint64("fluxdb-reproc-shard-count"),
				ReprocSharderStartBlockNum: viper.GetUint64("fluxdb-reproc-shard-start-block-num"),
				ReprocSharderStopBlockNum:  viper.GetUint64("fluxdb-reproc-shard-stop-block-num"),
				ReprocInjectorShardIndex:   viper.GetUint64("fluxdb-reproc-injector-shard-index"),
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "trxdb-loader",
		Title:       "DB loader",
		Description: "Main blocks and transactions database",
		MetricsID:   "trxdb-loader",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/trxdb-loader.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("trxdb-loader-processing-type", "live", "The actual processing type to perform, either `live`, `batch` or `patch`")
			cmd.Flags().Uint64("trxdb-loader-batch-size", 1, "number of blocks batched together for database write")
			cmd.Flags().Uint64("trxdb-loader-start-block-num", 0, "[BATCH] Block number where we start processing")
			cmd.Flags().Uint64("trxdb-loader-stop-block-num", math.MaxUint32, "[BATCH] Block number where we stop processing")
			cmd.Flags().Uint64("trxdb-loader-num-blocks-before-start", 300, "[BATCH] Number of blocks to fetch before start block")
			cmd.Flags().String("trxdb-loader-http-listen-addr", KvdbHTTPServingAddr, "Listen address for /healthz endpoint")
			cmd.Flags().Int("trxdb-loader-parallel-file-download-count", 2, "Maximum number of files to download in parallel")
			cmd.Flags().Bool("trxdb-loader-allow-live-on-empty-table", true, "[LIVE] force pipeline creation if live request and table is empty")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}

			return trxdbLoaderApp.New(&trxdbLoaderApp.Config{
				ChainId:                   viper.GetString("common-chain-id"),
				ProcessingType:            viper.GetString("trxdb-loader-processing-type"),
				BlockStoreURL:             mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				KvdbDsn:                   mustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:           viper.GetString("common-blockstream-addr"),
				BatchSize:                 viper.GetUint64("trxdb-loader-batch-size"),
				StartBlockNum:             viper.GetUint64("trxdb-loader-start-block-num"),
				StopBlockNum:              viper.GetUint64("trxdb-loader-stop-block-num"),
				NumBlocksBeforeStart:      viper.GetUint64("trxdb-loader-num-blocks-before-start"),
				AllowLiveOnEmptyTable:     viper.GetBool("trxdb-loader-allow-live-on-empty-table"),
				HTTPListenAddr:            viper.GetString("trxdb-loader-http-listen-addr"),
				ParallelFileDownloadCount: viper.GetInt("trxdb-loader-parallel-file-download-count"),
			}), nil
		},
	})

	// Blockmeta
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "blockmeta",
		Title:       "Blockmeta",
		Description: "Serves information about blocks",
		MetricsID:   "blockmeta",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/blockmeta.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("blockmeta-grpc-listen-addr", BlockmetaServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("blockmeta-live-source", true, "Whether we want to connect to a live block source or not.")
			cmd.Flags().Bool("blockmeta-enable-readiness-probe", true, "Enable blockmeta's app readiness probe")
			cmd.Flags().StringSlice("blockmeta-eos-api-upstream-addr", []string{NodeosAPIAddr}, "EOS API address to fetch info from running chain, must be in-sync")
			cmd.Flags().StringSlice("blockmeta-eos-api-extra-addr", []string{MindreaderNodeosAPIAddr}, "Additional EOS API address for ID lookups (valid even if it is out of sync or read-only)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			for _, addr := range viper.GetStringSlice("blockmeta-eos-api-upstream-addr") {
				if !strings.HasPrefix(addr, "http") {
					addr = "http://" + addr
				}
				dblockmeta.APIs = append(dblockmeta.APIs, eos.New(addr))
			}
			for _, addr := range viper.GetStringSlice("blockmeta-eos-api-extra-addr") {
				if !strings.HasPrefix(addr, "http") {
					addr = "http://" + addr
				}
				dblockmeta.ExtraAPIs = append(dblockmeta.ExtraAPIs, eos.New(addr))
			}

			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}

			trxdbClient, err := trxdb.New(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")))
			if err != nil {
				return nil, err
			}

			//todo: add db to a modules struct in blockmeta
			db := &dblockmeta.EOSBlockmetaDB{
				Driver: trxdbClient,
			}

			return blockmetaApp.New(&blockmetaApp.Config{
				Protocol:        Protocol,
				BlockStreamAddr: viper.GetString("common-blockstream-addr"),
				GRPCListenAddr:  viper.GetString("blockmeta-grpc-listen-addr"),
				BlocksStoreURL:  mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				LiveSource:      viper.GetBool("blockmeta-live-source"),
			}, db), nil
		},
	})

	// Abicodec
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "abicodec",
		Title:       "ABI codec",
		Description: "Decodes binary data against ABIs for different contracts",
		MetricsID:   "abicodec",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/abicodec.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("abicodec-grpc-listen-addr", AbiServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("abicodec-cache-base-url", "{dfuse-data-dir}/storage/abicache", "path where the cache store is state")
			cmd.Flags().String("abicodec-cache-file-name", "abicodec_cache.bin", "path where the cache store is state")
			cmd.Flags().Bool("abicodec-export-cache", false, "Export cache and exit")
			cmd.Flags().String("abicodec-export-cache-url", "{dfuse-data-dir}/storage/abicache", "path where the exported cache will reside")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}

			return abicodecApp.New(&abicodecApp.Config{
				GRPCListenAddr: viper.GetString("abicodec-grpc-listen-addr"),
				SearchAddr:     viper.GetString("common-search-addr"),
				KvdbDSN:        mustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				CacheBaseURL:   mustReplaceDataDir(dfuseDataDir, viper.GetString("abicodec-cache-base-url")),
				CacheStateName: viper.GetString("abicodec-cache-file-name"),
				ExportCache:    viper.GetBool("abicodec-export-cache"),
				ExportCacheURL: viper.GetString("abicodec-export-cache-url"),
			}), nil
		},
	})

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
			mapper, err := eosSearch.NewEOSBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create EOS block mapper: %w", err)
			}

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
				BlockMapper:        mapper,
				StartBlockResolver: bstream.ParallelStartResolver(startBlockResolvers, -1),
			}), nil
		},
	})

	// Search Router
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-router",
		Title:       "Search router",
		Description: "Routes search queries to archiver, live",
		MetricsID:   "router",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(router|app/router).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			// Router-specific flags
			cmd.Flags().String("search-router-grpc-listen-addr", RouterServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("search-router-enable-retry", false, "Enables the router's attempt to retry a backend search if there is an error. This could have adverse consequences when search through the live")
			cmd.Flags().Uint64("search-router-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().Uint64("search-router-lib-delay-tolerance", 0, "Number of blocks above a backend's lib we allow a request query to be served (Live & Router)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return routerApp.New(&routerApp.Config{
				ServiceVersion:     viper.GetString("search-common-mesh-service-version"),
				BlockmetaAddr:      viper.GetString("common-blockmeta-addr"),
				GRPCListenAddr:     viper.GetString("search-router-grpc-listen-addr"),
				HeadDelayTolerance: viper.GetUint64("search-router-head-delay-tolerance"),
				LibDelayTolerance:  viper.GetUint64("search-router-lib-delay-tolerance"),
				EnableRetry:        viper.GetBool("search-router-enable-retry"),
			}, &routerApp.Modules{
				Dmesh: modules.SearchDmeshClient,
			}), nil
		},
	})

	// Search Archive
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-archive",
		Title:       "Search archive",
		Description: "Serves historical search queries",
		MetricsID:   "archive",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(archive|app/archive).*", nil),
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
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			return archiveApp.New(&archiveApp.Config{
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
				Dmesh: modules.SearchDmeshClient,
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-live",
		Title:       "Search live",
		Description: "Serves live search queries",
		MetricsID:   "live",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(live|app/live).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Uint32("search-live-tier-level", 100, "Level of the search tier")
			cmd.Flags().String("search-live-grpc-listen-addr", LiveServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-live-live-indices-path", "{dfuse-data-dir}/search/live", "Location for live indexes (ideally a ramdisk)")
			cmd.Flags().Int("search-live-truncation-threshold", 1, "number of available dmesh peers that should serve irreversible blocks before we truncate them from this backend's memory")
			cmd.Flags().Duration("search-live-realtime-tolerance", 1*time.Minute, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Duration("search-live-shutdown-delay", 0*time.Second, "On shutdown, time to wait before actually leaving, to try and drain connections")
			cmd.Flags().Uint64("search-live-start-block-drift-tolerance", 500, "allowed number of blocks between search archive and network head to get start block from the search archive")
			cmd.Flags().Uint64("search-live-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			mapper, err := eosSearch.NewEOSBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create EOS block mapper: %w", err)
			}
			return liveApp.New(&liveApp.Config{
				ServiceVersion:           viper.GetString("search-common-mesh-service-version"),
				TierLevel:                viper.GetUint32("search-live-tier-level"),
				GRPCListenAddr:           viper.GetString("search-live-grpc-listen-addr"),
				BlockmetaAddr:            viper.GetString("common-blockmeta-addr"),
				LiveIndexesPath:          mustReplaceDataDir(dfuseDataDir, viper.GetString("search-live-live-indices-path")),
				BlocksStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				BlockstreamAddr:          viper.GetString("common-blockstream-addr"),
				StartBlockDriftTolerance: viper.GetUint64("search-live-start-block-drift-tolerance"),
				ShutdownDelay:            viper.GetDuration("search-live-shutdown-delay"),
				TruncationThreshold:      viper.GetInt("search-live-truncation-threshold"),
				RealtimeTolerance:        viper.GetDuration("search-live-realtime-tolerance"),
				PublishInterval:          viper.GetDuration("search-common-mesh-publish-interval"),
				HeadDelayTolerance:       viper.GetUint64("search-live-head-delay-tolerance"),
			}, &liveApp.Modules{
				BlockMapper: mapper,
				Dmesh:       modules.SearchDmeshClient,
			}), nil
		},
	})

	// Search Fork Resolver
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-forkresolver",
		Title:       "Search fork resolver",
		Description: "Search forks",
		MetricsID:   "forkresolver",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(forkresolver|app/forkresolver).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("search-forkresolver-grpc-listen-addr", ForkresolverServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-forkresolver-http-listen-addr", ForkresolverHTTPServingAddr, "Address to listen for incoming HTTP requests")
			cmd.Flags().String("search-forkresolver-indices-path", "{dfuse-data-dir}/search/forkresolver", "Location for inflight indices")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			mapper, err := eosSearch.NewEOSBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create EOS block mapper: %w", err)
			}

			return forkresolverApp.New(&forkresolverApp.Config{
				ServiceVersion:  viper.GetString("search-common-mesh-service-version"),
				GRPCListenAddr:  viper.GetString("search-forkresolver-grpc-listen-addr"),
				HttpListenAddr:  viper.GetString("search-forkresolver-http-listen-addr"),
				PublishInterval: viper.GetDuration("search-common-mesh-publish-interval"),
				IndicesPath:     viper.GetString("search-forkresolver-indices-path"),
				BlocksStoreURL:  viper.GetString("common-blocks-store-url"),
			}, &forkresolverApp.Modules{
				Dmesh:       modules.SearchDmeshClient,
				BlockMapper: mapper,
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosws",
		Title:       "EOSWS",
		Description: "Serves websocket and http queries to clients",
		MetricsID:   "eosws",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/eosws.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosws-http-listen-addr", EoswsHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("eosws-superviser-rpc-addr", NodeosAPIAddr, "RPC endpoint of the superviser instance")
			cmd.Flags().Duration("eosws-realtime-tolerance", 15*time.Second, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Int("eosws-blocks-buffer-size", 10, "Number of blocks to keep in memory when initializing")
			cmd.Flags().String("eosws-fluxdb-addr", FluxDBServingAddr, "FluxDB server address")
			cmd.Flags().Bool("eosws-fetch-price", false, "Enable regularly fetching token price from a known source")
			cmd.Flags().Bool("eosws-fetch-vote-tally", false, "Enable regularly fetching vote tally")
			cmd.Flags().String("eosws-search-addr-secondary", "", "secondary search grpc endpoint")
			cmd.Flags().Duration("eosws-filesource-ratelimit", 2*time.Millisecond, "time to sleep between blocks coming from filesource to control replay speed")
			cmd.Flags().String("eosws-healthz-secret", "", "Secret to access healthz")
			cmd.Flags().String("eosws-data-integrity-proof-secret", "boo", "Data integrity secret for DIPP middleware")
			cmd.Flags().Bool("eosws-authenticate-superviser-api", false, "Gate access to native superviser APIs with authentication")
			cmd.Flags().Bool("eosws-use-opencensus-stack-driver", false, "Enables stack driver tracing")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			return eoswsApp.New(&eoswsApp.Config{
				HTTPListenAddr:              viper.GetString("eosws-http-listen-addr"),
				NodeosRPCEndpoint:           viper.GetString("eosws-superviser-rpc-addr"),
				BlockmetaAddr:               viper.GetString("common-blockmeta-addr"),
				KVDBDSN:                     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:             viper.GetString("common-blockstream-addr"),
				SourceStoreURL:              mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				SearchAddr:                  viper.GetString("common-search-addr"),
				SearchAddrSecondary:         viper.GetString("eosws-search-addr-secondary"),
				FluxHTTPAddr:                viper.GetString("eosws-fluxdb-addr"),
				AuthenticateNodeosAPI:       viper.GetBool("eosws-authenticate-superviser-api"),
				MeteringPlugin:              viper.GetString("common-metering-plugin"),
				AuthPlugin:                  viper.GetString("common-auth-plugin"),
				UseOpencensusStackdriver:    viper.GetBool("eosws-use-opencensus-stack-driver"),
				FetchPrice:                  viper.GetBool("eosws-fetch-price"),
				FetchVoteTally:              viper.GetBool("eosws-fetch-vote-tally"),
				FilesourceRateLimitPerBlock: viper.GetDuration("eosws-filesource-ratelimit"),
				BlocksBufferSize:            viper.GetInt("eosws-blocks-buffer-size"),
				RealtimeTolerance:           viper.GetDuration("eosws-realtime-tolerance"),
				DataIntegrityProofSecret:    viper.GetString("eosws-data-integrity-proof-secret"),
				HealthzSecret:               viper.GetString("eosws-healthz-secret"),
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dgraphql",
		Title:       "GraphQL",
		Description: "Serves GraphQL queries to clients",
		MetricsID:   "dgraphql",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dgraphql.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dgraphql-http-addr", DgraphqlHTTPServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dgraphql-grpc-addr", DgraphqlGrpcServingAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dgraphql-abi-addr", AbiServingAddr, "Base URL for abicodec service")
			cmd.Flags().Duration("dgraphql-graceful-shutdown-delay", 0, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().Bool("dgraphql-disable-authentication", false, "disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "flag to override trace id or not")
			cmd.Flags().String("dgraphql-protocol", "eos", "name of the protocol")
			cmd.Flags().String("dgraphql-auth-url", JWTIssuerURL, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("dgraphql-api-key", DgraphqlAPIKey, "API key used in graphiql")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}

			return dgraphqlEosio.NewApp(&dgraphqlEosio.Config{
				// eos specifc configs
				SearchAddr:        viper.GetString("common-search-addr"),
				ABICodecAddr:      viper.GetString("dgraphql-abi-addr"),
				BlockMetaAddr:     viper.GetString("common-blockmeta-addr"),
				KVDBDSN:           mustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				RatelimiterPlugin: viper.GetString("common-ratelimiter-plugin"),
				Config: dgraphqlApp.Config{
					// base dgraphql configs
					// need to be passed this way because promoted fields
					HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
					GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
					AuthPlugin:      viper.GetString("common-auth-plugin"),
					MeteringPlugin:  viper.GetString("common-metering-plugin"),
					NetworkID:       viper.GetString("common-network-id"),
					OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
					Protocol:        viper.GetString("dgraphql-protocol"),
					JwtIssuerURL:    viper.GetString("dgraphql-auth-url"),
					ApiKey:          viper.GetString("dgraphql-api-key"),
				},
			})
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosq",
		Title:       "Eosq",
		Description: "EOSIO Block Explorer",
		MetricsID:   "eosq",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/eosq.*", nil),
		InitFunc:    nil,
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosq-http-listen-addr", EosqHTTPServingAddr, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("eosq-api-endpoint-url", APIProxyHTTPListenAddr, "API key used in eosq")
			cmd.Flags().String("eosq-auth-url", JWTIssuerURL, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("eosq-api-key", EosqAPIKey, "API key used in eosq")
			cmd.Flags().String("eosq-environment", "dev", "Environment where eosq will run (dev, dev, production)")
			cmd.Flags().String("eosq-available-networks", "", "json string to configure the networks section of eosq.")
			cmd.Flags().String("eosq-default-network", "local", "Default network that is displayed. It should correspond to an `id` in the available networks")
			cmd.Flags().Bool("eosq-disable-analytics", true, "Disables sentry and segment")
			cmd.Flags().Bool("eosq-display-price", false, "Should display prices via our price API")
			cmd.Flags().String("eosq-price-ticker-name", "EOS", "The price ticker")
			cmd.Flags().Bool("eosq-on-demand", false, "Is eosq deployed for an on-demand network")
			cmd.Flags().Bool("eosq-disable-tokenmeta", true, "Disables tokenmeta calls from eosq")
			return nil
		},

		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return eosqApp.New(&eosqApp.Config{
				HTTPListenAddr:    viper.GetString("eosq-http-listen-addr"),
				Environement:      viper.GetString("eosq-environment"),
				APIEndpointURL:    viper.GetString("eosq-api-endpoint-url"),
				ApiKey:            viper.GetString("eosq-api-key"),
				AuthEndpointURL:   viper.GetString("eosq-auth-url"),
				AvailableNetworks: viper.GetString("eosq-available-networks"),
				DisableAnalytics:  viper.GetBool("eosq-disable-analytics"),
				DefaultNetwork:    viper.GetString("eosq-default-network"),
				DisplayPrice:      viper.GetBool("eosq-display-price"),
				PriceTickerName:   viper.GetString("eosq-price-ticker-name"),
				OnDemand:          viper.GetBool("eosq-on-demand"),
				DisableTokenmeta:  viper.GetBool("eosq-disable-tokenmeta"),
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dashboard",
		Title:       "Dashboard",
		Description: "dfuse for EOSIO - dashboard",
		MetricsID:   "dashboard",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-box/dashboard.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dashboard-grpc-listen-addr", DashboardGrpcServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dashboard-http-listen-addr", DashboardHTTPListenAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dashboard-eos-node-manager-api-addr", EosManagerAPIAddr, "Address of the superviser manager api")
			// FIXME: we can re-add when the app actually makes use of it.
			//cmd.Flags().String("dashboard-mindreader-manager-api-addr", MindreaderNodeosAPIAddr, "Address of the mindreader superviser manager api")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return dashboard.New(&dashboard.Config{
				GRPCListenAddr:        viper.GetString("dashboard-grpc-listen-addr"),
				HTTPListenAddr:        viper.GetString("dashboard-http-listen-addr"),
				EosNodeManagerAPIAddr: viper.GetString("dashboard-eos-node-manager-api-addr"),
				//NodeosAPIHTTPServingAddr: viper.GetString("dashboard-mindreader-manager-api-addr"),
			}, &dashboard.Modules{
				Launcher:    modules.Launcher,
				DmeshClient: modules.SearchDmeshClient,
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "apiproxy",
		Title:       "API Proxy",
		Description: "Reverse proxies all API services under one port",
		MetricsID:   "apiproxy",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/apiproxy.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("apiproxy-http-listen-addr", APIProxyHTTPListenAddr, "HTTP Listener address")
			cmd.Flags().String("apiproxy-https-listen-addr", "", "If non-empty, will listen for HTTPS connections on this address")
			cmd.Flags().String("apiproxy-autocert-domains", "", "If non-empty, requests certificates from Let's Encrypt for this comma-separated list of domains")
			cmd.Flags().String("apiproxy-autocert-cache-dir", "{dfuse-data-dir}/api-proxy", "Path to directory where certificates will be saved to disk")
			cmd.Flags().String("apiproxy-eosws-http-addr", EoswsHTTPServingAddr, "Target address of the eosws API endpoint")
			cmd.Flags().String("apiproxy-dgraphql-http-addr", DgraphqlHTTPServingAddr, "Target address of the dgraphql API endpoint")
			cmd.Flags().String("apiproxy-superviser-http-addr", NodeosAPIAddr, "Address of a queriable superviser instance")
			cmd.Flags().String("apiproxy-root-http-addr", EosqHTTPServingAddr, "What to serve at the root of the proxy (defaults to eosq)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			autocertDomains := strings.Split(viper.GetString("apiproxy-autocert-domains"), ",")
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			return apiproxy.New(&apiproxy.Config{
				HTTPListenAddr:   viper.GetString("apiproxy-http-listen-addr"),
				HTTPSListenAddr:  viper.GetString("apiproxy-https-listen-addr"),
				AutocertDomains:  autocertDomains,
				AutocertCacheDir: mustReplaceDataDir(dfuseDataDir, viper.GetString("apiproxy-autocert-cache-dir")),
				EoswsHTTPAddr:    viper.GetString("apiproxy-eosws-http-addr"),
				DgraphqlHTTPAddr: viper.GetString("apiproxy-dgraphql-http-addr"),
				NodeosHTTPAddr:   viper.GetString("apiproxy-superviser-http-addr"),
				RootHTTPAddr:     viper.GetString("apiproxy-root-http-addr"),
			}), nil
		},
	})

}

func makeDirs(directories []string) error {
	for _, directory := range directories {
		err := os.MkdirAll(directory, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %q: %w", directory, err)
		}
	}

	return nil
}

func dfuseAbsoluteDataDir() (string, error) {
	return filepath.Abs(viper.GetString("global-data-dir"))
}
