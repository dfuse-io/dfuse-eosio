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
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"time"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	_ "github.com/dfuse-io/dauth/null" // register plugin
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dfuse-eosio/dashboard"
	dgraphqlApp "github.com/dfuse-io/dfuse-eosio/dgraphql/app/dgraphql"
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq"
	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	kvdbLoaderApp "github.com/dfuse-io/dfuse-eosio/kvdb-loader/app/kvdb-loader"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	nodeosManagerApp "github.com/dfuse-io/manageos/app/nodeos_manager"
	nodeosMindreaderApp "github.com/dfuse-io/manageos/app/nodeos_mindreader"
	mergerApp "github.com/dfuse-io/merger/app/merger"
	relayerApp "github.com/dfuse-io/relayer/app/relayer"
	archiveApp "github.com/dfuse-io/search/app/archive"
	forkresolverApp "github.com/dfuse-io/search/app/forkresolver"
	indexerApp "github.com/dfuse-io/search/app/indexer"
	liveApp "github.com/dfuse-io/search/app/live"
	routerApp "github.com/dfuse-io/search/app/router"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	// Manager
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "manager",
		Title:       "Producer node",
		Description: "Block producing node",
		MetricsID:   "manager",
		Logger:      newLoggerDef("github.com/dfuse-io/manageos/app/nodeos_manager", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			// TODO: check if `~/.dfuse/binaries/nodeos-{ProducerNodeVersion}` exists, if not download from:
			// curl https://abourget.keybase.pub/dfusebox/binaries/nodeos-{ProducerNodeVersion}

			err := makeDirs([]string{
				filepath.Join(config.DataDir, "managernode", "config"),
				filepath.Join(config.DataDir, "storage", "snapshots"),
				filepath.Join(config.DataDir, "managernode", "data"),
				filepath.Join(config.DataDir, "storage", "pitreos"),
			})
			if err != nil {
				return err
			}

			if config.BoxConfig.RunProducer {
				if config.BoxConfig.ProducerConfigIni == "" {
					return fmt.Errorf("producerConfigIni empty when runProducer is enabled")
				}

				if err := writeGenesisAndConfig(config.BoxConfig.ProducerConfigIni, config.BoxConfig.GenesisJSON, filepath.Join(config.DataDir, "managernode", "config"), "producer"); err != nil {
					return err
				}
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			if config.BoxConfig.RunProducer {
				return nodeosManagerApp.New(&nodeosManagerApp.Config{
					ManagerAPIAddress:   config.EosManagerHTTPAddr,
					NodeosAPIAddress:    config.NodeosAPIAddr,
					ConnectionWatchdog:  false,
					NodeosConfigDir:     filepath.Join(config.DataDir, "managernode", "config"),
					NodeosBinPath:       config.NodeExecutable,
					NodeosDataDir:       filepath.Join(config.DataDir, "managernode", "data"),
					TrustedProducer:     config.NodeosTrustedProducer,
					ReadinessMaxLatency: 5 * time.Second,
					BackupStoreURL:      filepath.Join(config.DataDir, "storage", "pitreos"),
					BootstrapDataURL:    config.BootstrapDataURL,
					DebugDeepMind:       false,
					AutoRestoreLatest:   false,
					RestoreBackupName:   "",
					RestoreSnapshotName: "",
					SnapshotStoreURL:    filepath.Join(config.DataDir, "storage", "snapshots"),
					ShutdownDelay:       config.NodeosShutdownDelay,
					BackupTag:           "default",
					AutoBackupModulo:    0,
					AutoBackupPeriod:    0,
					AutoSnapshotModulo:  0,
					AutoSnapshotPeriod:  0,
					DisableProfiler:     true,
					NodeosExtraArgs:     config.NodeosExtraArgs,
					LogToZap:            true,
					ForceProduction:     true,
				})
			}
			// Can we detect a nil interface
			return nil
		},
	})

	// Mindreader
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader",
		Title:       "Reader node",
		Description: "Blocks reading node",
		MetricsID:   "manager",
		Logger:      newLoggerDef("github.com/dfuse-io/manageos/(app/nodeos_mindreader|mindreader).*", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "mindreadernode", "config"),
				filepath.Join(config.DataDir, "mindreadernode", "data"),
				filepath.Join(config.DataDir, "storage", "pitreos"),
				filepath.Join(config.DataDir, "storage", "snapshots"),
				filepath.Join(config.DataDir, "storage", "one-blocks"),
				filepath.Join(config.DataDir, "mindreader"),
			})
			if err != nil {
				return err
			}

			if config.BoxConfig.ReaderConfigIni == "" {
				return fmt.Errorf("readerConfigIni empty")
			}

			if err := writeGenesisAndConfig(config.BoxConfig.ReaderConfigIni, config.BoxConfig.GenesisJSON, filepath.Join(config.DataDir, "mindreadernode", "config"), "reader"); err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return nodeosMindreaderApp.New(&nodeosMindreaderApp.Config{
				ManagerAPIAddress:          config.EosMindreaderHTTPAddr,
				NodeosAPIAddress:           config.NodeosAPIAddr,
				NodeosConfigDir:            filepath.Join(config.DataDir, "mindreadernode", "config"),
				NodeosDataDir:              filepath.Join(config.DataDir, "mindreadernode", "data"),
				BackupStoreURL:             filepath.Join(config.DataDir, "storage", "pitreos"),
				SnapshotStoreURL:           filepath.Join(config.DataDir, "storage", "snapshots"),
				ArchiveStoreURL:            filepath.Join(config.DataDir, "storage", "one-blocks"),
				WorkingDir:                 filepath.Join(config.DataDir, "mindreader"),
				ConnectionWatchdog:         false,
				NodeosBinPath:              config.NodeExecutable,
				ReadinessMaxLatency:        5 * time.Second,
				BackupTag:                  "default",
				GRPCAddr:                   config.MindreaderGRPCAddr,
				StartBlockNum:              config.StartBlock,
				StopBlockNum:               config.StopBlock,
				MindReadBlocksChanCapacity: 100,
				LogToZap:                   true,
				DisableProfiler:            true,
				StartFailureHandlerFunc: func() {
					userLog.Error(`*********************************************************************************
* Mindreader failed to start nodeos process
* To see nodeos logs...
* DEBUG=\"github.com/dfuse-io/manageos.*\" dfusebox start
*********************************************************************************

Make sure you have a dfuse instrumented 'nodeos' binary, follow instructions
at https://github.com/dfuse-io/dfuse-eosio#dfuse-Instrumented-EOSIO-Prebuilt-Binaries
to find how to install it.`)
					os.Exit(1)
				},
			})
		},
	})

	// Relayer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "relayer",
		Title:       "Relayer",
		Description: "Serves blocks as a stream, with a buffer",
		MetricsID:   "relayer",
		Logger:      newLoggerDef("github.com/dfuse-io/relayer.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("relayer-grpc-listen-addr", RelayerServingAddr, "Listening address for gRPC service serving blocks")
			cmd.Flags().StringSlice("relayer-source", []string{MindreaderGRPCAddr}, "List of blockstream sources to connect to for live block feeds (repeat flag as needed)")
			cmd.Flags().String("relayer-merger-addr", MergerServingAddr, "Address for grpc merger service")
			cmd.Flags().Int("relayer-buffer-size", 300, "number of blocks that will be kept and sent immediately on connection")
			cmd.Flags().Duration("relayer-max-drift", 300*time.Second, "max delay between live blocks before we die in hope of a better world")
			cmd.Flags().Uint64("relayer-min-start-offset", 120, "number of blocks before HEAD where we want to start for faster buffer filling (missing blocks come from files/merger)")
			cmd.Flags().Duration("relayer-max-source-latency", 1*time.Minute, "max latency tolerated to connect to a source")
			cmd.Flags().Duration("relayer-init-time", 1*time.Minute, "time before we start looking for max drift")
			cmd.Flags().String("relayer-source-store", "storage/merged-blocks", "Store path url to read batch files from")
			cmd.Flags().Bool("relayer-enable-readiness-probe", true, "Enable relayer's app readiness probe")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return relayerApp.New(&relayerApp.Config{
				SourcesAddr:          viper.GetStringSlice("relayer-source"),
				GRPCListenAddr:       viper.GetString("relayer-grpc-listen-addr"),
				MergerAddr:           viper.GetString("relayer-merger-addr"),
				BufferSize:           viper.GetInt("relayer-buffer-size"),
				MaxDrift:             viper.GetDuration("relayer-max-drift"),
				MaxSourceLatency:     viper.GetDuration("relayer-max-source-latency"),
				InitTime:             viper.GetDuration("relayer-init-time"),
				MinStartOffset:       viper.GetUint64("relayer-min-start-offset"),
				Protocol:             Protocol,
				EnableReadinessProbe: viper.GetBool("relayer-enable-readiness-probe"),
				SourceStoreURL:       buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("relayer-source-store")),
			})
		},
	})

	// Merger
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "merger",
		Title:       "Merger",
		Description: "Produces merged block files from single-block files",
		MetricsID:   "merger",
		Logger:      newLoggerDef("github.com/dfuse-io/merger.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("merger-merged-block-path", "storage/merged-blocks", "URL of storage to write merged-block-files to")
			cmd.Flags().String("merger-one-block-path", "storage/one-blocks", "URL of storage to read one-block-files from")
			cmd.Flags().Duration("merger-store-timeout", 2*time.Minute, "max time to to allow for each store operation")
			cmd.Flags().Duration("merger-time-between-store-lookups", 10*time.Second, "delay between polling source store (higher for remote storage)")
			cmd.Flags().String("merger-grpc-serving-addr", MergerServingAddr, "gRPC listen address to serve merger endpoints")
			cmd.Flags().Bool("merger-process-live-blocks", true, "Ignore --start-.. and --stop-.. blocks, and process only live blocks")
			cmd.Flags().Uint64("merger-start-block-num", 0, "FOR REPROCESSING: if >= 0, Set the block number where we should start processing")
			cmd.Flags().Uint64("merger-stop-block-num", 0, "FOR REPROCESSING: if > 0, Set the block number where we should stop processing (and stop the process)")
			cmd.Flags().String("merger-progress-filename", "", "FOR REPROCESSING: If non-empty, will update progress in this file and start right there on restart")
			cmd.Flags().Uint64("merger-minimal-block-num", 0, "FOR LIVE: Set the minimal block number where we should start looking at the destination storage to figure out where to start")
			cmd.Flags().Duration("merger-writers-leeway", 10*time.Second, "how long we wait after seeing the upper boundary, to ensure that we get as many blocks as possible in a bundle")
			cmd.Flags().String("merger-seen-blocks-file", "merger/merger.seen.gob", "file to save to / load from the map of 'seen blocks'")
			cmd.Flags().Uint64("merger-max-fixable-fork", 10000, "after that number of blocks, a block belonging to another fork will be discarded (DELETED depending on flagDeleteBlocksBefore) instead of being inserted in last bundle")
			cmd.Flags().Bool("merger-delete-blocks-before", true, "Enable deletion of one-block files when prior to the currently processed bundle (to avoid long file listings)")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-merged-block-path")),
				StorageOneBlockFilesPath:     buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-one-block-path")),
				StoreOperationTimeout:        viper.GetDuration("merger-store-timeout"),
				TimeBetweenStoreLookups:      viper.GetDuration("merger-time-between-store-lookups"),
				GRPCListenAddr:               viper.GetString("merger-grpc-serving-addr"),
				Live:                         viper.GetBool("merger-process-live-blocks"),
				StartBlockNum:                viper.GetUint64("merger-start-block-num"),
				StopBlockNum:                 viper.GetUint64("merger-stop-block-num"),
				ProgressFilename:             viper.GetString("merger-progress-filename"),
				MinimalBlockNum:              viper.GetUint64("merger-minimal-block-num"),
				WritersLeewayDuration:        viper.GetDuration("merger-writers-leeway"),
				SeenBlocksFile:               buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-seen-blocks-file")),
				MaxFixableFork:               viper.GetUint64("merger-max-fixable-fork"),
				DeleteBlocksBefore:           viper.GetBool("merger-delete-blocks-before"),
				Protocol:                     Protocol,
				EnableReadinessProbe:         true,
			})
		},
	})

	// fluxdb
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "fluxdb",
		Title:       "FluxDB",
		Description: "Temporal chain state store",
		MetricsID:   "fluxdb",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/fluxdb.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("fluxdb-enable-server-mode", true, "Enable dev mode")
			cmd.Flags().Bool("fluxdb-enable-inject-mode", true, "Enable dev mode")
			cmd.Flags().String("fluxdb-kvdb-store-dsn", "badger://%s/flux.db", "Storage connection string")
			cmd.Flags().String("fluxdb-kvdb-grpc-serving-addr", FluxDBServingAddr, "Storage connection string")
			cmd.Flags().Duration("fluxdb-db-graceful-shutdown-delay", 0, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().String("fluxdb-db-blocks-store", "gs://example/blocks", "dbin blocks store")
			cmd.Flags().String("fluxdb-db-block-stream-addr", "localhost:9001", "gRPC endpoint to get real-time blocks")
			cmd.Flags().Int("fluxdb-db-threads", 2, "Number of threads of parallel processing")
			cmd.Flags().Bool("fluxdb-db-live", true, "Also connect to a live source, can be turn off when doing re-processing")
			cmd.Flags().String("fluxdb-block-stream-addr", RelayerServingAddr, "grpc address of a block stream, usually the relayer grpc address")
			cmd.Flags().String("fluxdb-merger-blocks-files-path", "storage/merged-blocks", "Store path url to read batch files from")
			cmd.Flags().Bool("fluxdb-enable-dev-mode", false, "Enable dev mode")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return fluxdbApp.New(&fluxdbApp.Config{
				EnableServerMode:   viper.GetBool("luxdb-enable-server-mode"),
				EnableInjectMode:   viper.GetBool("fluxdb-enable-inject-mode"),
				StoreDSN:           fmt.Sprintf(viper.GetString("fluxdb-kvdb-store-dsn"), filepath.Join(config.DataDir, "fluxdb")),
				EnableLivePipeline: viper.GetBool("fluxdb-db-live"),
				BlockStreamAddr:    viper.GetString("fluxdb-block-stream-addr"),
				BlockStoreURL:      buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("fluxdb-merger-blocks-files-path")),
				HTTPListenAddr:     viper.GetString("fluxdb-kvdb-grpc-serving-addr"),
				EnableDevMode:      viper.GetBool("fluxdb-enable-dev-mode"),
				ThreadsNum:         2,
				NetworkID:          NetworkID,
			})
		},
	})

	// KVDB Loader
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "kvdb-loader",
		Title:       "DB loader",
		Description: "Main blocks and transactions database",
		MetricsID:   "kvdb-loader",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/kvdb-loader.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("kvdb-loader-chain-id", "", "Chain ID")
			cmd.Flags().String("kvdb-loader-processing-type", "live", "The actual processing type to perform, either `live`, `batch` or `patch`")
			cmd.Flags().String("kvdb-loader-merged-block-path", "storage/merged-blocks", "URL of storage to read one-block-files from")
			cmd.Flags().String("kvdb-loader-kvdb-dsn", "badger://%s/kvdb_badger.db?compression=zstd", "kvdb connection string")
			cmd.Flags().String("kvdb-loader-block-stream-addr", RelayerServingAddr, "grpc address of a block stream, usually the relayer grpc address")
			cmd.Flags().Uint64("kvdb-loader-batch-size", 1, "number of blocks batched together for database write")
			cmd.Flags().Uint64("kvdb-loader-start-block-num", 0, "[BATCH] Block number where we start processing")
			cmd.Flags().Uint64("kvdb-loader-stop-block-num", math.MaxUint32, "[BATCH] Block number where we stop processing")
			cmd.Flags().Uint64("kvdb-loader-num-blocks-before-start", 300, "[BATCH] Number of blocks to fetch before start block")
			cmd.Flags().String("kvdb-loader-http-listen-addr", KvdbHTTPServingAddr, "Listen address for /healthz endpoint")
			cmd.Flags().Bool("kvdb-loader-allow-live-on-empty-table", true, "[LIVE] force pipeline creation if live request and table is empty")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return kvdbLoaderApp.New(&kvdbLoaderApp.Config{
				ChainId:                   viper.GetString("chain-id"),
				ProcessingType:            viper.GetString("kvdb-loader-processing-type"),
				BlockStoreURL:             buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("kvdb-loader-merged-block-path")),
				KvdbDsn:                   fmt.Sprintf(viper.GetString("kvdb-loader-kvdb-dsn"), viper.GetString("global-data-dir")),
				BlockStreamAddr:           viper.GetString("kvdb-loader-block-stream-addr"),
				BatchSize:                 viper.GetUint64("kvdb-loader-batch-size"),
				StartBlockNum:             viper.GetUint64("kvdb-loader-start-block-num"),
				StopBlockNum:              viper.GetUint64("kvdb-loader-stop-block-num"),
				NumBlocksBeforeStart:      viper.GetUint64("kvdb-loader-num-blocks-before-start"),
				AllowLiveOnEmptyTable:     viper.GetBool("kvdb-loader-allow-live-on-empty-table"),
				HTTPListenAddr:            viper.GetString("kvdb-loader-http-listen-addr"),
				Protocol:                  Protocol.String(),
				ParallelFileDownloadCount: 2,
			})
		},
	})

	// Blockmeta
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "blockmeta",
		Title:       "Blockmeta",
		Description: "Serves information about blocks",
		MetricsID:   "blockmeta",
		Logger:      newLoggerDef("github.com/dfuse-io/blockmeta.*", nil),
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return blockmetaApp.New(&blockmetaApp.Config{
				KvdbDSN:                 config.KvdbDSN,
				BlocksStore:             filepath.Join(config.DataDir, "storage", "merged-blocks"),
				BlockStreamAddr:         config.RelayerServingAddr,
				ListenAddr:              config.BlockmetaServingAddr,
				Protocol:                config.Protocol,
				LiveSource:              true,
				EnableReadinessProbe:    true,
				EOSAPIUpstreamAddresses: []string{config.BoxConfig.NodeosAPIAddr},
				EOSAPIExtraAddresses:    []string{config.MindreaderNodeosAPIAddr},
			})
		},
	})

	// Abicodec
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "abicodec",
		Title:       "ABI codec",
		Description: "Decodes binary data against ABIs for different contracts",
		MetricsID:   "abicodec",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/abicodec.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("abicodec-grpc-listen-addr", ":9000", "TCP Listener addr for gRPC")
			cmd.Flags().String("abicodec-search-addr", ":7004", "Base URL for search service")
			cmd.Flags().String("abicodec-kvdb-dsn", "badger://%s/kvdb_badger.db?compression=zstd", "kvdb connection string")
			cmd.Flags().String("abicodec-cache-base-url", "storage/abicahe", "path where the cache store is state")
			cmd.Flags().String("abicodec-cache-file-name", "abicodec_cache.bin", "path where the cache store is state")
			cmd.Flags().Bool("abicodec-export-cache", false, "Export cache and exit")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return abicodecApp.New(&abicodecApp.Config{
				GRPCListenAddr:       viper.GetString("abicodec-grpc-listen-addr"),
				SearchAddr:           viper.GetString("abicodec-search-addr"),
				KvdbDSN:              fmt.Sprintf(viper.GetString("abicodec-kvdb-dsn"), viper.GetString("global-data-dir")),
				ExportCache:          viper.GetBool("abicodec-export-cache"),
				CacheBaseURL:         buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("abicodec-cache-base-url")),
				CacheStateName:       viper.GetString("abicodec-cache-file-name"),
				EnableReadinessProbe: true,
			})
		},
	})

	// Search Indexer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "indexer",
		Title:       "Search indexer",
		Description: "Indexes transactions for search",
		MetricsID:   "indexer",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(indexer|app/indexer).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("search-indexer-grpc-listen-addr", IndexerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-indexer-http-listen-addr", IndexerHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().Bool("search-indexer-enable-upload", true, "Upload merged indexes to the --indexes-store")
			cmd.Flags().Bool("search-indexer-delete-after-upload", true, "Delete local indexes after uploading them")
			cmd.Flags().String("search-indexer-block-stream-addr", RelayerServingAddr, "gRPC URL to reach a stream of blocks")
			cmd.Flags().String("search-indexer-blockmeta-addr", BlockmetaServingAddr, "Blockmeta endpoint is queried to find the last irreversible block on the network being indexed")
			cmd.Flags().Int("search-indexer-start-block", 0, "Start indexing from block num")
			cmd.Flags().Uint("search-indexer-stop-block", 0, "Stop indexing at block num")
			cmd.Flags().Bool("search-indexer-enable-batch-mode", false, "Enabled the indexer in batch mode with a start & stoip block")
			cmd.Flags().Uint("search-indexer-num-blocks-before-start", 0, "Number of blocks to fetch before start block")
			cmd.Flags().Bool("search-indexer-verbose", false, "Verbose logging")
			cmd.Flags().Bool("search-indexer-enable-index-truncation", false, "Enable index truncation, requires a relative --start-block (negative number)")
			cmd.Flags().Bool("search-indexer-enable-readiness-probe", true, "Enable search indexer's app readiness probe")
			cmd.Flags().Uint64("search-indexer-shard-size", 200, "Number of blocks to store in a given Bleve index")
			cmd.Flags().String("search-indexer-writable-path", "search/indexer", "Writable base path for storing index files")
			cmd.Flags().String("search-indexer-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-indexer-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return indexerApp.New(&indexerApp.Config{
				Protocol:                            Protocol,
				HTTPListenAddr:                      viper.GetString("search-indexer-http-listen-addr"),
				GRPCListenAddr:                      viper.GetString("search-indexer-grpc-listen-addr"),
				IndexesStoreURL:                     filepath.Join(config.DataDir, "storage", "indexes"),
				BlocksStoreURL:                      filepath.Join(config.DataDir, "storage", "merged-blocks"),
				BlockstreamAddr:                     viper.GetString("search-indexer-block-stream-addr"),
				DfuseHooksActionName:                viper.GetString("search-indexer-dfuse-hooks-action-name"),
				IndexingRestrictionsJSON:            viper.GetString("search-indexer-indexing-restrictions-json"),
				WritablePath:                        viper.GetString("search-indexer-writable-path"),
				ShardSize:                           viper.GetUint64("search-indexer-shard-size"),
				StartBlock:                          int64(viper.GetInt("search-indexer-start-block")),
				StopBlock:                           viper.GetUint64("search-indexer-stop-block"),
				IsVerbose:                           viper.GetBool("search-indexer-verbose"),
				EnableBatchMode:                     viper.GetBool("search-indexer-enable-batch-mode"),
				BlockmetaAddr:                       viper.GetString("search-indexer-blockmeta-addr"),
				NumberOfBlocksToFetchBeforeStarting: viper.GetUint64("search-indexer-num-blocks-before-start"),
				EnableUpload:                        viper.GetBool("search-indexer-enable-upload"),
				DeleteAfterUpload:                   viper.GetBool("search-indexer-delete-after-upload"),
				EnableIndexTruncation:               viper.GetBool("search-indexer-enable-index-truncation"),
				EnableReadinessProbe:                viper.GetBool("search-indexer-enable-readiness-probe"),
			})
		},
	})

	// Search Router
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "router",
		Title:       "Search router",
		Description: "Routes search queries to archiver, live",
		MetricsID:   "router",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(router|app/router).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("search-router-listen-addr", RouterServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-router-blockmeta-addr", BlockmetaServingAddr, "Blockmeta endpoint is queried to validate cursors that are passed LIB and forked out")
			cmd.Flags().Bool("search-router-enable-retry", false, "Enables the router's attempt to retry a backend search if there is an error. This could have adverse consequences when search through the live")
			cmd.Flags().Uint64("search-router-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().Uint64("search-router-lib-delay-tolerance", 0, "Number of blocks above a backend's lib we allow a request query to be served (Live & Router)")
			cmd.Flags().Bool("search-router-enable-readiness-probe", true, "Enable search router's app readiness probe")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return routerApp.New(&routerApp.Config{
				Dmesh:                modules.SearchDmeshClient,
				Protocol:             Protocol,
				BlockmetaAddr:        viper.GetString("search-router-blockmeta-addr"),
				GRPCListenAddr:       viper.GetString("search-router-listen-addr"),
				HeadDelayTolerance:   viper.GetUint64("search-router-head-delay-tolerance"),
				LibDelayTolerance:    viper.GetUint64("search-router-lib-delay-tolerance"),
				EnableReadinessProbe: viper.GetBool("search-router-enable-readiness-probe"),
				EnableRetry:          viper.GetBool("search-router-enable-retry"),
			})
		},
	})

	// Search Archive
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "archive",
		Title:       "Search archive",
		Description: "Serves historical search queries",
		MetricsID:   "archive",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(archive|app/archive).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			// These flags are scoped to search, since they are shared betwween search-router, search-live, search-archive, etc....
			if cmd.Flag("search-mesh-store-addr") == nil {
				cmd.Flags().String("search-mesh-store-addr", "", "address of the backing etcd cluster for mesh service discovery")
			}
			if cmd.Flag("search-mesh-namespace") == nil {
				cmd.Flags().String("search-mesh-namespace", DmeshNamespace, "dmesh namespace where services reside (eos-mainnet)")
			}
			if cmd.Flag("search-mesh-service-version") == nil {
				cmd.PersistentFlags().String("search-mesh-service-version", DmeshServiceVersion, "dmesh service version (v1)")
			}
			p := "search-archive-"
			cmd.Flags().String(p+"grpc-listen-addr", ArchiveServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String(p+"http-listen-addr", ArchiveHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String(p+"memcache-addr", "", "Empty results cache's memcache server address")
			cmd.Flags().Bool(p+"enable-empty-results-cache", false, "Enable roaring-bitmap-based empty results caching")
			cmd.Flags().Uint32(p+"tier-level", 50, "Level of the search tier")
			cmd.Flags().Duration(p+"mesh-publish-polling-duration", 0*time.Second, "How often does search archive poll dmesh")
			cmd.Flags().Bool(p+"enable-moving-tail", false, "Enable moving tail, requires a relative --start-block (negative number)")
			cmd.Flags().Uint64(p+"shard-size", 200, "Number of blocks to store in a given Bleve index")
			cmd.Flags().Int(p+"start-block", 0, "Start at given block num, the initial sync and polling")
			cmd.Flags().Uint(p+"stop-block", 0, "Stop before given block num, the initial sync and polling")
			cmd.Flags().Bool(p+"index-polling", true, "Populate local indexes using indexes store polling.")
			cmd.Flags().Bool(p+"sync-from-storage", false, "Download missing indexes from --indexes-store before starting")
			cmd.Flags().Int(p+"sync-max-indexes", 100000, "Maximum number of indexes to sync. On production, use a very large number.")
			cmd.Flags().Int(p+"indices-dl-threads", 1, "Number of indices files to download from the GS input store and decompress in parallel. In prod, use large value like 20.")
			cmd.Flags().Int(p+"max-query-threads", 10, "Number of end-user query parallel threads to query 5K-blocks indexes")
			cmd.Flags().Duration(p+"shutdown-delay", 0*time.Second, "On shutdown, time to wait before actually leaving, to try and drain connections")
			cmd.Flags().String(p+"warmup-filepath", "", "Optional filename containing queries to warm-up the search")
			cmd.Flags().Bool(p+"enable-readiness-probe", true, "Enable search archive's app readiness probe")
			cmd.Flags().String(p+"indexes-store", "storage/indexes", "GS path to read or write index shards")
			cmd.Flags().String(p+"writable-path", "search/archiver", "Writable base path for storing index files")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return archiveApp.New(&archiveApp.Config{
				Dmesh:                   modules.SearchDmeshClient,
				Protocol:                Protocol,
				MemcacheAddr:            viper.GetString("search-archive-memcache-addr"),
				EnableEmptyResultsCache: viper.GetBool("search-archive-enable-empty-results-cache"),
				ServiceVersion:          viper.GetString("search-mesh-service-version"),
				TierLevel:               viper.GetUint32("search-archive-tier-level"),
				GRPCListenAddr:          viper.GetString("search-archive-grpc-listen-addr"),
				HTTPListenAddr:          viper.GetString("search-archive-http-listen-addr"),
				PublishDuration:         viper.GetDuration("search-archive-mesh-publish-polling-duration"),
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
				EnableReadinessProbe:    viper.GetBool("search-archive-enable-readiness-probe"),
				IndexesStoreURL:         buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-archive-indexes-store")),
				IndexesPath:             buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-archive-writable-path")),
			})
		},
	})
	// Search Live
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "live",
		Title:       "Search live",
		Description: "Serves live search queries",
		MetricsID:   "live",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(live|app/live).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			if cmd.Flag("search-mesh-store-addr") == nil {
				cmd.Flags().String("search-mesh-store-addr", "", "address of the backing etcd cluster for mesh service discovery")
			}
			if cmd.Flag("search-mesh-namespace") == nil {
				cmd.Flags().String("search-mesh-namespace", DmeshNamespace, "dmesh namespace where services reside (eos-mainnet)")
			}
			if cmd.Flag("search-mesh-service-version") == nil {
				cmd.PersistentFlags().String("search-mesh-service-version", DmeshServiceVersion, "dmesh service version (v1)")
			}
			cmd.Flags().Uint32("search-live-tier-level", 100, "Level of the search tier")
			cmd.Flags().String("search-live-grpc-listen-addr", LiveServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-live-block-stream-addr", RelayerServingAddr, "gRPC Address to reach a stream of blocks")
			cmd.Flags().String("search-live-live-indices-path", "search/live", "Location for live indexes (ideally a ramdisk)")
			cmd.Flags().Int("search-live-truncation-threshold", 1, "number of available dmesh peers that should serve irreversible blocks before we truncate them from this backend's memory")
			cmd.Flags().Duration("search-live-realtime-tolerance", 1*time.Minute, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Duration("search-live-shutdown-delay", 0*time.Second, "On shutdown, time to wait before actually leaving, to try and drain connections")
			cmd.Flags().String("search-live-blockmeta-addr", BlockmetaServingAddr, "Blockmeta endpoint is queried for its headinfo service")
			cmd.Flags().Uint64("search-live-start-block-drift-tolerance", 500, "allowed number of blocks between search archive and network head to get start block from the search archive")
			cmd.Flags().Bool("search-live-enable-readiness-probe", true, "Enable search live's app readiness probe")
			cmd.Flags().String("search-live-blocks-store", "storage/merged-blocks", "Path to read blocks files")
			cmd.Flags().Duration("search-live-mesh-publish-polling-duration", 0*time.Second, "How often does search live poll dmesh")
			cmd.Flags().Uint64("search-live-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().String("search-live-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-live-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return liveApp.New(&liveApp.Config{
				Dmesh:                    modules.SearchDmeshClient,
				Protocol:                 Protocol,
				ServiceVersion:           viper.GetString("search-mesh-service-version"),
				TierLevel:                viper.GetUint32("search-live-tier-level"),
				GRPCListenAddr:           viper.GetString("search-live-grpc-listen-addr"),
				BlockmetaAddr:            viper.GetString("search-live-blockmeta-addr"),
				BlocksStoreURL:           buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-live-blocks-store")),
				BlockstreamAddr:          viper.GetString("search-live-block-stream-addr"),
				StartBlockDriftTolerance: viper.GetUint64("search-live-start-block-drift-tolerance"),
				ShutdownDelay:            viper.GetDuration("search-live-shutdown-delay"),
				LiveIndexesPath:          viper.GetString("search-live-live-indices-path"),
				TruncationThreshold:      viper.GetInt("search-live-truncation-threshold"),
				RealtimeTolerance:        viper.GetDuration("search-live-realtime-tolerance"),
				EnableReadinessProbe:     viper.GetBool("search-live-enable-readiness-probe"),
				PublishDuration:          viper.GetDuration("search-live-mesh-publish-polling-duration"),
				HeadDelayTolerance:       viper.GetUint64("search-live-head-delay-tolerance"),
				IndexingRestrictionsJSON: viper.GetString("search-live-indexing-restrictions-json"),
				DfuseHooksActionName:     viper.GetString("search-live-dfuse-hooks-action-name"),
			})
		},
	})

	// Search Fork Resolver
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "forkresolver",
		Title:       "Search fork resolver",
		Description: "Search forks",
		MetricsID:   "forkresolver",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(forkresolver|app/forkresolver).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			if cmd.Flag("search-mesh-store-addr") == nil {
				cmd.Flags().String("search-mesh-store-addr", "", "address of the backing etcd cluster for mesh service discovery")
			}
			if cmd.Flag("search-mesh-namespace") == nil {
				cmd.Flags().String("search-mesh-namespace", DmeshNamespace, "dmesh namespace where services reside (eos-mainnet)")
			}
			if cmd.Flag("search-mesh-service-version") == nil {
				cmd.PersistentFlags().String("search-mesh-service-version", DmeshServiceVersion, "dmesh service version (v1)")
			}

			cmd.Flags().String("search-forkresolver-grpc-listen-addr", ForkresolverServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("search-forkresolver-http-listen-addr", ForkresolverHTTPServingAddr, "Address to listen for incoming HTTP requests")
			cmd.Flags().String("search-forkresolver-indices-path", "search/forkresolver", "Location for inflight indices")
			cmd.Flags().Duration("search-forkresolver-mesh-publish-polling-duration", 0*time.Second, "How often does search forkresolver poll dmesh")
			cmd.Flags().String("search-forkresolver-blocks-store", "storage/merged-blocks", "Path to read blocks files")
			cmd.Flags().String("search-forkresolver-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-forkresolver-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			cmd.Flags().Bool("search-forkresolver-enable-readiness-probe", true, "Enable search forlresolver's app readiness probe")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return forkresolverApp.New(&forkresolverApp.Config{
				Dmesh:                    modules.SearchDmeshClient,
				Protocol:                 Protocol,
				ServiceVersion:           viper.GetString("search-mesh-service-version"),
				GRPCListenAddr:           viper.GetString("search-forkresolver-grpc-listen-addr"),
				HttpListenAddr:           viper.GetString("search-forkresolver-http-listen-addr"),
				PublishDuration:          viper.GetDuration("search-forkresolver-mesh-publish-polling-duration"),
				IndicesPath:              viper.GetString("search-forkresolver-indices-path"),
				BlocksStoreURL:           viper.GetString("search-forkresolver-blocks-store"),
				DfuseHooksActionName:     viper.GetString("search-forkresolver-dfuse-hooks-action-name"),
				IndexingRestrictionsJSON: viper.GetString("search-forkresolver-indexing-restrictions-json"),
				EnableReadinessProbe:     viper.GetBool("search-forkresolver-enable-readiness-probe"),
			})
		},
	})

	// eosWS (deprecated app, scheduled to be dismantled and features migrated to dgraphql)
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosws",
		Title:       "EOSWS",
		Description: "Serves websocket and http queries to clients",
		MetricsID:   "eosws",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/eosws.*", nil),
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return eoswsApp.New(&eoswsApp.Config{
				HTTPListenAddr:              config.EoswsHTTPServingAddr,
				SearchAddr:                  config.RouterServingAddr,
				KVDBDSN:                     config.KvdbDSN,
				AuthPlugin:                  "null://",
				MeteringPlugin:              "null://",
				NodeosRPCEndpoint:           config.NodeosAPIAddr,
				BlockmetaAddr:               config.BlockmetaServingAddr,
				BlockStreamAddr:             config.RelayerServingAddr,
				SourceStoreURL:              filepath.Join(config.DataDir, "storage", "merged-blocks"),
				FluxHTTPAddr:                config.FluxDBServingAddr,
				UseOpencensusStackdriver:    false,
				FetchPrice:                  false,
				FetchVoteTally:              false,
				FilesourceRateLimitPerBlock: 1 * time.Millisecond,
				BlocksBufferSize:            10,
				RealtimeTolerance:           15 * time.Second,
				DataIntegrityProofSecret:    "boo",
				//NetworkID:       "eos-local",
			})
		},
	})

	// dGraphql
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dgraphql",
		Title:       "GraphQL",
		Description: "Serves GraphQL queries to clients",
		MetricsID:   "dgraphql",
		Logger:      newLoggerDef("github.com/dfuse-io/dgraphql.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dgraphql-http-addr", DgraphqlHTTPServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dgraphql-grpc-addr", DgraphqlGrpcServingAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dgraphql-search-addr", RouterServingAddr, "Base URL for search service")
			cmd.Flags().String("dgraphql-abi-addr", AbiServingAddr, "Base URL for abicodec service")
			cmd.Flags().String("dgraphql-block-meta-addr", BlockmetaServingAddr, "Base URL for blockmeta service")
			cmd.Flags().String("dgraphql-tokenmeta-addr", TokenmetaServingAddr, "Base URL tokenmeta service")
			cmd.Flags().String("dgraphql-search-addr-v2", "", "Base URL for search service")
			cmd.Flags().String("dgraphql-kvdb-dsn", "bigtable://dev.dev/test", "Bigtable database connection information") // Used on EOSIO right now, eventually becomes the reference.
			cmd.Flags().String("dgraphql-auth-plugin", "null://", "Auth plugin, ese dauth repository")
			cmd.Flags().String("dgraphql-metering-plugin", "null://", "Metering plugin, see dmetering repository")
			cmd.Flags().String("dgraphql-network-id", NetworkID, "Network ID, for billing (usually maps namespaces on deployments)")
			cmd.Flags().Duration("dgraphql-graceful-shutdown-delay", 0*time.Millisecond, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().Bool("dgraphql-disable-authentication", false, "disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "flag to override trace id or not")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return dgraphqlApp.New(&dgraphqlApp.Config{
				HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
				GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
				SearchAddr:      viper.GetString("dgraphql-search-addr"),
				SearchAddrV2:    viper.GetString("dgraphql-search-addr-v2"),
				KVDBDSN:         viper.GetString("dgraphql-kvdb-dsn"),
				NetworkID:       viper.GetString("dgraphql-network-id"),
				AuthPlugin:      viper.GetString("dgraphql-auth-plugin"),
				MeteringPlugin:  viper.GetString("dgraphql-metering-plugin"),
				ABICodecAddr:    viper.GetString("dgraphql-abi-addr"),
				BlockMetaAddr:   viper.GetString("dgraphql-block-meta-addr"),
				TokenmetaAddr:   viper.GetString("dgraphql-tokenmeta-addr"),
				OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
			})
		},
	})

	// eosq
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosq",
		Title:       "Eosq",
		Description: "EOSIO Block Explorer",
		MetricsID:   "eosq",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/eosq.*", nil),
		InitFunc:    nil,
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return eosqApp.New(&eosqApp.Config{
				DashboardHTTPListenAddr: config.DashboardHTTPListenAddr,
				HttpListenAddr:          config.EosqHTTPServingAddr,
			})
		},
	})

	// dashboard
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dashboard",
		Title:       "Dashboard",
		Description: "Main dfusebox dashboard",
		MetricsID:   "dashboard",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/dashboard.*", nil),
		InitFunc:    nil,
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return dashboard.New(&dashboard.Config{
				DmeshClient:              modules.SearchDmeshClient,
				ManagerCommandURL:        config.EosManagerHTTPAddr,
				GRPCListenAddr:           config.DashboardGrpcServingAddr,
				HTTPListenAddr:           config.DashboardHTTPListenAddr,
				EoswsHTTPServingAddr:     config.EoswsHTTPServingAddr,
				DgraphqlHTTPServingAddr:  config.DgraphqlHTTPServingAddr,
				NodeosAPIHTTPServingAddr: config.MindreaderNodeosAPIAddr,
				Launcher:                 modules.Launcher,
				MetricManager:            modules.MetricManager,
			})
		},
	})

}

func writeGenesisAndConfig(configIni string, genesisJSON string, destDir string, destNode string) error {
	managerGenesisFile := path.Join(destDir, "genesis.json")
	if err := ioutil.WriteFile(managerGenesisFile, []byte(genesisJSON), 0755); err != nil {
		return fmt.Errorf("failed to create %s's genesis file %q: %w", destNode, managerGenesisFile, err)
	}

	// TODO: NOTIFY THE USER when its overwritten: COMPARE the config.ini previously there, read & compare & notify
	managerConfigFile := path.Join(destDir, "config.ini")
	if err := ioutil.WriteFile(managerConfigFile, []byte(configIni), 0755); err != nil {
		return fmt.Errorf("failed to create %s's config file %q: %w", destNode, managerConfigFile, err)
	}

	return nil
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

func newLoggerDef(regex string, levels []zapcore.Level) *launcher.LoggingDef {
	if len(levels) == 0 {
		levels = []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}
	}

	return &launcher.LoggingDef{
		Levels: levels,
		Regex:  regex,
	}
}
