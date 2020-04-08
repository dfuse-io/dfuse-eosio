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
	"os"
	"path"
	"path/filepath"
	"time"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	_ "github.com/dfuse-io/dauth/null" // register plugin
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dfuse-eosio/dashboard"
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq"
	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	kvdbLoaderApp "github.com/dfuse-io/dfuse-eosio/kvdb-loader/app/kvdb-loader"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	dgraphqlEosioApp "github.com/dfuse-io/dgraphql/app/eosio"
	nodeosManagerApp "github.com/dfuse-io/manageos/app/nodeos_manager"
	nodeosMindreaderApp "github.com/dfuse-io/manageos/app/nodeos_mindreader"
	mergerApp "github.com/dfuse-io/merger/app/merger"
	relayerApp "github.com/dfuse-io/relayer/app/relayer"
	archiveApp "github.com/dfuse-io/search/app/archive"
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
				EnableReadinessProbe: true,
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
				filepath.Join(config.DataDir, "storage", "one-blocks"),
				filepath.Join(config.DataDir, "merger"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return mergerApp.New(&mergerApp.Config{
				StartBlockNum:           config.StartBlock,
				Live:                    true,
				GRPCListenAddr:          config.MergerServingAddr,
				Protocol:                config.Protocol,
				MinimalBlockNum:         0,
				StopBlockNum:            0,
				StoragePathDest:         filepath.Join(config.DataDir, "storage", "merged-blocks"),
				StoragePathSource:       filepath.Join(config.DataDir, "storage", "one-blocks"),
				TimeBetweenStoreLookups: 200 * time.Millisecond,
				WritersLeewayDuration:   10 * time.Second,
				SeenBlocksFile:          filepath.Join(config.DataDir, "merger", "merger.seen.gob"),
				MaxFixableFork:          10000,
				DeleteBlocksBefore:      true,
				EnableReadinessProbe:    true,
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "fluxdb"),
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return fluxdbApp.New(&fluxdbApp.Config{
				EnableServerMode: true,
				EnableInjectMode: true,
				StoreDSN:         fmt.Sprintf("badger://%s/flux.db", filepath.Join(config.DataDir, "fluxdb")),
				//StoreDSN:           "tikv://pd0:2379?keyPrefix=02000001",
				//StoreDSN:           "bigkv://dev.dev/flux?createTable=true",
				NetworkID:          config.NetworkID,
				EnableLivePipeline: true,
				BlockStreamAddr:    config.RelayerServingAddr,
				ThreadsNum:         2,
				HTTPListenAddr:     config.FluxDBServingAddr,
				EnableDevMode:      false,
				BlockStoreURL:      filepath.Join(config.DataDir, "storage", "merged-blocks"),
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "kvdb"),
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return kvdbLoaderApp.New(&kvdbLoaderApp.Config{
				ChainId:               "",
				ProcessingType:        "live",
				BlockStoreURL:         filepath.Join(config.DataDir, "storage", "merged-blocks"),
				BlockStreamAddr:       config.RelayerServingAddr,
				KvdbDsn:               config.KvdbDSN,
				BatchSize:             1,
				AllowLiveOnEmptyTable: true,
				Protocol:              config.Protocol.String(),
				HTTPListenAddr:        config.KvdbHTTPServingAddr,
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "indexes"),
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
				filepath.Join(config.DataDir, "search", "indexer"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return indexerApp.New(&indexerApp.Config{
				IndexesStoreURL:       filepath.Join(config.DataDir, "storage", "indexes"),
				GRPCListenAddr:        config.IndexerServingAddr,
				HTTPListenAddr:        config.IndexerHTTPServingAddr,
				BlocksStoreURL:        filepath.Join(config.DataDir, "storage", "merged-blocks"),
				Protocol:              config.Protocol,
				BlockstreamAddr:       config.RelayerServingAddr,
				WritablePath:          filepath.Join(config.DataDir, "search", "indexer"),
				ShardSize:             config.ShardSize,
				StartBlock:            int64(config.StartBlock),
				StopBlock:             config.StopBlock,
				BlockmetaAddr:         config.BlockmetaServingAddr,
				EnableUpload:          true,
				DeleteAfterUpload:     true,
				EnableIndexTruncation: false,
				EnableReadinessProbe:  true,
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "abicache"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return abicodecApp.New(&abicodecApp.Config{
				GRPCListenAddr:       config.AbiServingAddr,
				SearchAddr:           config.RouterServingAddr,
				KvdbDSN:              config.KvdbDSN,
				ExportCache:          false,
				CacheBaseURL:         "file://" + filepath.Join(config.DataDir, "storage", "abicache"),
				CacheStateName:       "abicodec_cache.bin",
				EnableReadinessProbe: true,
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
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return routerApp.New(&routerApp.Config{
				Dmesh:                modules.SearchDmeshClient,
				Protocol:             config.Protocol,
				BlockmetaAddr:        config.BlockmetaServingAddr,
				GRPCListenAddr:       config.RouterServingAddr,
				HeadDelayTolerance:   0,
				LibDelayTolerance:    0,
				EnableReadinessProbe: true,
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
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "indexes"),
				filepath.Join(config.DataDir, "search", "archiver"),
			})
			if err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return archiveApp.New(&archiveApp.Config{
				Dmesh:                modules.SearchDmeshClient,
				Protocol:             config.Protocol,
				ServiceVersion:       config.DmeshServiceVersion,
				TierLevel:            50,
				GRPCListenAddr:       config.ArchiveServingAddr,
				HTTPListenAddr:       config.ArchiveHTTPServingAddr,
				EnableMovingTail:     false,
				IndexesStoreURL:      filepath.Join(config.DataDir, "storage", "indexes"),
				IndexesPath:          filepath.Join(config.DataDir, "search", "archiver"),
				ShardSize:            config.ShardSize,
				StartBlock:           int64(config.StartBlock),
				StopBlock:            config.StopBlock,
				SyncFromStore:        true,
				SyncMaxIndexes:       100000,
				IndicesDLThreads:     1,
				NumQueryThreads:      10,
				IndexPolling:         true,
				EnableReadinessProbe: true,
			})
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "live",
		Title:       "Search live",
		Description: "Serves live search queries",
		MetricsID:   "live",
		Logger:      newLoggerDef("github.com/dfuse-io/search/(live|app/live).*", nil),
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := os.RemoveAll(filepath.Join(config.DataDir, "search", "live"))
			if err != nil {
				return err
			}

			err = makeDirs([]string{
				filepath.Join(config.DataDir, "storage", "merged-blocks"),
				filepath.Join(config.DataDir, "search", "live"),
			})
			if err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return liveApp.New(&liveApp.Config{
				Dmesh:                    modules.SearchDmeshClient,
				Protocol:                 config.Protocol,
				ServiceVersion:           config.DmeshServiceVersion,
				TierLevel:                100,
				GRPCListenAddr:           config.LiveServingAddr,
				BlocksStoreURL:           filepath.Join(config.DataDir, "storage", "merged-blocks"),
				BlockstreamAddr:          config.RelayerServingAddr,
				BlockmetaAddr:            config.BlockmetaServingAddr,
				LiveIndexesPath:          filepath.Join(config.DataDir, "search", "live"),
				TruncationThreshold:      1,
				RealtimeTolerance:        1 * time.Minute,
				EnableReadinessProbe:     true,
				StartBlockDriftTolerance: 500,
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
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) launcher.App {
			return dgraphqlEosioApp.New(&dgraphqlEosioApp.Config{
				HTTPListenAddr:  config.DgraphqlHTTPServingAddr,
				GRPCListenAddr:  config.DgraphqlGrpcServingAddr,
				SearchAddr:      config.RouterServingAddr,
				ABICodecAddr:    config.AbiServingAddr,
				BlockMetaAddr:   config.BlockmetaServingAddr,
				TokenmetaAddr:   config.TokenmetaServingAddr,
				KVDBDSN:         config.KvdbDSN,
				NetworkID:       config.NetworkID,
				AuthPlugin:      "null://",
				MeteringPlugin:  "null://",
				OverrideTraceID: false,
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
				HttpListenAddr:          config.EosqHTTPServingAddress,
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
