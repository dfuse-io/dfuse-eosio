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
	"strings"
	"time"

	dblockmeta "github.com/dfuse-io/dfuse-eosio/blockmeta"
	"github.com/dfuse-io/dfuse-eosio/eosdb"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	_ "github.com/dfuse-io/dauth/null" // register plugin
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dfuse-eosio/dashboard"
	dgraphqlEosio "github.com/dfuse-io/dfuse-eosio/dgraphql"
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq"
	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	kvdbLoaderApp "github.com/dfuse-io/dfuse-eosio/kvdb-loader/app/kvdb-loader"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
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
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("manager-api-addr", EosManagerHTTPAddr, "eos-manager API address")
			cmd.Flags().String("manager-nodeos-api-addr", NodeosAPIAddr, "Target API address")
			cmd.Flags().Bool("manager-connection-watchdog", false, "Force-reconnect dead peers automatically")
			cmd.Flags().String("manager-config-dir", "manager/config", "Directory for config files")
			cmd.Flags().String("manager-nodeos-path", NodeosBinPath, "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
			cmd.Flags().String("manager-data-dir", "managernode/data", "Directory for data (blocks)")
			cmd.Flags().String("manager-producer-hostname", "", "Hostname that will produce block (other will be paused)")
			cmd.Flags().String("manager-trusted-producer", "", "The EOS account name of the Block Producer we trust all blocks from")
			cmd.Flags().Duration("manager-readiness-max-latency", 5*time.Second, "/healthz will return error until nodeos head block time is within that duration to now")
			cmd.Flags().String("manager-backup-store-url", PitreosPath, "Storage bucket with path prefix where backups should be done")
			cmd.Flags().String("manager-bootstrap-data-url", "", "The bootstrap data URL containing specific chain data used to initialized it.")
			cmd.Flags().String("manager-snapshot-store-url", SnapshotsPath, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")
			cmd.Flags().Bool("manager-debug-deep-mind", false, "Whether to print all Deepming log lines or not")
			cmd.Flags().Bool("manager-auto-restore", false, "Enables restore from the latest backup on boot if there is no block logs or if nodeos cannot start at all. Do not use on a single BP node")
			cmd.Flags().String("manager-restore-backup-name", "", "If non-empty, the node will be restored from that backup every time it starts.")
			cmd.Flags().String("manager-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
			cmd.Flags().Duration("manager-shutdown-delay", 0*time.Second, "Delay before shutting manager when sigterm received")
			cmd.Flags().String("manager-backup-tag", "default", "tag to identify the backup")
			cmd.Flags().Bool("manager-disable-profiler", true, "Disables the manageos profiler")
			cmd.Flags().StringSlice("manager-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().Bool("manager-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().Int("manager-auto-backup-modulo", 0, "If non-zero, a backup will be taken every {auto-backup-modulo} block.")
			cmd.Flags().Duration("manager-auto-backup-period", 0, "If non-zero, a backup will be taken every period of {auto-backup-period}. Specify 1h, 2h...")
			cmd.Flags().Int("manager-auto-snapshot-modulo", 0, "If non-zero, a snapshot will be taken every {auto-snapshot-modulo} block.")
			cmd.Flags().Duration("manager-auto-snapshot-period", 0, "If non-zero, a snapshot will be taken every period of {auto-snapshot-period}. Specify 1h, 2h...")
			cmd.Flags().String("manager-volume-snapshot-appver", "geth-v1", "[application]-v[version_number], used for persistentVolume snapshots")
			cmd.Flags().Duration("manager-auto-volume-snapshot-period", 0, "If non-zero, a volume snapshot will be taken every period of {auto-volume-snapshot-period}. Specify 1h, 2h...")
			cmd.Flags().Int("manager-auto-volume-snapshot-modulo", 0, "If non-zero, a volume snapshot will be taken every {auto-volume-snapshot-modulo} blocks. Ex: 500000")
			cmd.Flags().String("manager-target-volume-snapshot-specific", "", "Comma-separated list of block numbers where volume snapshots will be done automatically")
			cmd.Flags().Bool("manager-force-production", true, "Forces the production of blocks")
			return nil
		},
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			// TODO: check if `~/.dfuse/binaries/nodeos-{ProducerNodeVersion}` exists, if not download from:
			// curl https://abourget.keybase.pub/dfusebox/binaries/nodeos-{ProducerNodeVersion}
			if config.BoxConfig.RunProducer {
				managerConfigDir := buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("manager-config-dir"))
				if strings.HasPrefix(managerConfigDir, "/") {
					err := makeDirs([]string{
						managerConfigDir,
					})
					if err != nil {
						return err
					}
				}
				if config.BoxConfig.ProducerConfigIni == "" {
					return fmt.Errorf("producerConfigIni empty when runProducer is enabled")
				}

				if err := writeGenesisAndConfig(config.BoxConfig.ProducerConfigIni, config.BoxConfig.GenesisJSON, managerConfigDir, "producer"); err != nil {
					return err
				}
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			if config.BoxConfig.RunProducer {
				return nodeosManagerApp.New(&nodeosManagerApp.Config{
					ManagerAPIAddress:       viper.GetString("manager-api-addr"),
					NodeosAPIAddress:        viper.GetString("manager-nodeos-api-addr"),
					ConnectionWatchdog:      viper.GetBool("manager-connection-watchdog"),
					NodeosConfigDir:         buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("manager-config-dir")),
					NodeosBinPath:           viper.GetString("manager-nodeos-path"),
					NodeosDataDir:           buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("manager-data-dir")),
					ProducerHostname:        viper.GetString("manager-producer-hostname"),
					TrustedProducer:         viper.GetString("manager-trusted-producer"),
					ReadinessMaxLatency:     viper.GetDuration("manager-readiness-max-latency"),
					ForceProduction:         viper.GetBool("manager-force-production"),
					NodeosExtraArgs:         viper.GetStringSlice("manager-nodeos-args"),
					BackupStoreURL:          buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("manager-backup-store-url")),
					BootstrapDataURL:        viper.GetString("manager-bootstrap-data-url"),
					DebugDeepMind:           viper.GetBool("manager-debug-deep-mind"),
					LogToZap:                viper.GetBool("manager-log-to-zap"),
					AutoRestoreLatest:       viper.GetBool("manager-auto-restore"),
					RestoreBackupName:       viper.GetString("manager-restore-backup-name"),
					RestoreSnapshotName:     viper.GetString("manager-restore-snapshot-name"),
					SnapshotStoreURL:        buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("manager-snapshot-store-url")),
					ShutdownDelay:           viper.GetDuration("manager-shutdown-delay"),
					BackupTag:               viper.GetString("manager-backup-tag"),
					AutoBackupModulo:        viper.GetInt("manager-auto-backup-modulo"),
					AutoBackupPeriod:        viper.GetDuration("manager-auto-backup-period"),
					AutoSnapshotModulo:      viper.GetInt("manager-auto-snapshot-modulo"),
					AutoSnapshotPeriod:      viper.GetDuration("manager-auto-snapshot-period"),
					DisableProfiler:         viper.GetBool("manager-disable-profiler"),
					StartFailureHandlerFunc: nil,
				}), nil
			}
			// Can we detect a nil interface
			return nil, nil
		},
	})

	// Mindreader
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader",
		Title:       "Reader node",
		Description: "Blocks reading node",
		MetricsID:   "manager",
		Logger:      newLoggerDef("github.com/dfuse-io/manageos/(app/nodeos_mindreader|mindreader).*", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("mindreader-manager-api-addr", EosMindreaderHTTPAddr, "eos-manager API address")
			cmd.Flags().String("mindreader-api-addr", NodeosAPIAddr, "Target API address")
			cmd.Flags().Bool("mindreader-connection-watchdog", false, "Force-reconnect dead peers automatically")
			cmd.Flags().String("mindreader-config-dir", "mindreadernode/config", "Directory for config files. ")
			cmd.Flags().String("mindreader-nodeos-path", NodeosBinPath, "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
			cmd.Flags().String("mindreader-data-dir", "mindreadernode/data", "Directory for data (blocks)")
			cmd.Flags().String("mindreader-producer-hostname", "", "Hostname that will produce block (other will be paused)")
			cmd.Flags().String("mindreader-trusted-producer", "", "The EOS account name of the Block Producer we trust all blocks from")
			cmd.Flags().Duration("mindreader-readiness-max-latency", 5*time.Second, "/healthz will return error until nodeos head block time is within that duration to now")
			cmd.Flags().Bool("mindreader-disable-profiler", true, "Disables the manageos profiler")
			cmd.Flags().String("mindreader-backup-store-url", PitreosPath, "Storage bucket with path prefix where backups should be done")
			cmd.Flags().String("mindreader-snapshot-store-url", SnapshotsPath, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")
			cmd.Flags().String("mindreader-oneblock-store-url", OneBlockFilesPath, "Storage bucket with path prefix to write one-block file to")
			cmd.Flags().String("mindreader-working-dir", "mindreader", "Path where mindreader will stores its files")
			cmd.Flags().String("mindreader-backup-tag", "default", "tag to identify the backup")
			cmd.Flags().String("mindreader-grpc-listen-addr", MindreaderGRPCAddr, "gRPC listening address for stream of blocks and transactions")
			cmd.Flags().Uint("mindreader-start-block-num", 0, "Blocks that were produced with smaller block number then the given block num are skipped")
			cmd.Flags().Uint("mindreader-stop-block-num", 0, "Shutdown mindreader when we the following 'stop-block-num' has been reached, inclusively.")
			cmd.Flags().Int("mindreader-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the mindreader. Process will shutdown nodeos/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
			cmd.Flags().Bool("mindreader-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().StringSlice("mindreader-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().String("mindreader-bootstrap-data-url", "", "The bootstrap data URL containing specific chain data used to initialized it.")
			cmd.Flags().Bool("mindreader-debug-deep-mind", false, "Whether to print all Deepming log lines or not")
			cmd.Flags().Bool("mindreader-auto-restore", false, "Enables restore from the latest backup on boot if there is no block logs or if nodeos cannot start at all. Do not use on a single BP node")
			cmd.Flags().String("mindreader-restore-backup-name", "", "If non-empty, the node will be restored from that backup every time it starts.")
			cmd.Flags().String("mindreader-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
			cmd.Flags().Duration("mindreader-shutdown-delay", 0*time.Second, "Delay before shutting manager when sigterm received")
			cmd.Flags().String("mindreader-merged-blocks-store-url", MergedBlocksFilesPath, "USE FOR REPROCESSING ONLY. Storage bucket with path prefix to write merged blocks logs to (in conjunction with --merge-and-upload-directly)")
			cmd.Flags().Bool("mindreader-merge-and-upload-directly", false, "USE FOR REPROCESSING ONLY. When enabled, do not write one-block files, sidestep the merger and write the merged 100-blocks logs directly to --merged-blocks-store-url")
			cmd.Flags().Bool("mindreader-start-failure-handler", true, "Enables the startup function handler, that gets called if mindreader fails on startup")
			return nil
		},
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			nodeosConfigDir := buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-config-dir"))
			if strings.HasPrefix(nodeosConfigDir, "/") {
				err := makeDirs([]string{
					nodeosConfigDir,
				})
				if err != nil {
					return err
				}
			}

			if config.BoxConfig.ReaderConfigIni == "" {
				// TODO: considering this can eventually run the mindreader application solely, instead of returning
				// an error we may want to assume that config.ini file would already be at that place on disk
				return fmt.Errorf("readerConfigIni empty")
			}

			if err := writeGenesisAndConfig(config.BoxConfig.ReaderConfigIni, config.BoxConfig.GenesisJSON, nodeosConfigDir, "reader"); err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			archiveStoreURL := buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-oneblock-store-url"))
			if viper.GetBool("mindreader-merge-and-upload-directly") {
				archiveStoreURL = buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-merged-blocks-store-url"))
			}

			var startUpFunc func()
			if viper.GetBool("mindreader-start-failure-handler") {
				startUpFunc = func() {
					userLog.Error(`*********************************************************************************
									* Mindreader failed to start nodeos process
									* To see nodeos logs...
									* DEBUG=\"github.com/dfuse-io/manageos.*\" dfusebox start
									*********************************************************************************

									Make sure you have a dfuse instrumented 'nodeos' binary, follow instructions
									at https://github.com/dfuse-io/dfuse-eosio#dfuse-Instrumented-EOSIO-Prebuilt-Binaries
									to find how to install it.`)
					os.Exit(1)
				}
			}
			return nodeosMindreaderApp.New(&nodeosMindreaderApp.Config{
				ManagerAPIAddress:          viper.GetString("mindreader-manager-api-addr"),
				NodeosAPIAddress:           viper.GetString("mindreader-api-addr"),
				ConnectionWatchdog:         viper.GetBool("mindreader-connection-watchdog"),
				NodeosConfigDir:            buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-config-dir")),
				NodeosBinPath:              viper.GetString("mindreader-nodeos-path"),
				NodeosDataDir:              buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-data-dir")),
				ProducerHostname:           viper.GetString("mindreader-producer-hostname"),
				TrustedProducer:            viper.GetString("mindreader-trusted-producer"),
				ReadinessMaxLatency:        viper.GetDuration("mindreader-readiness-max-latency"),
				NodeosExtraArgs:            viper.GetStringSlice("mindreader-nodeos-args"),
				BackupStoreURL:             buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-backup-store-url")),
				BackupTag:                  viper.GetString("mindreader-backup-tag"),
				BootstrapDataURL:           viper.GetString("mindreader-bootstrap-data-url"),
				DebugDeepMind:              viper.GetBool("mindreader-debug-deep-mind"),
				LogToZap:                   viper.GetBool("mindreader-log-to-zap"),
				AutoRestoreLatest:          viper.GetBool("mindreader-auto-restore"),
				RestoreBackupName:          viper.GetString("mindreader-restore-backup-name"),
				RestoreSnapshotName:        viper.GetString("mindreader-restore-snapshot-name"),
				SnapshotStoreURL:           buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-snapshot-store-url")),
				ShutdownDelay:              viper.GetDuration("mindreader-shutdown-delay"),
				ArchiveStoreURL:            archiveStoreURL,
				MergeUploadDirectly:        viper.GetBool("mindreader-merge-and-upload-directly"),
				GRPCAddr:                   viper.GetString("mindreader-grpc-listen-addr"),
				StartBlockNum:              viper.GetUint64("mindreader-start-block-num"),
				StopBlockNum:               viper.GetUint64("mindreader-stop-block-num"),
				MindReadBlocksChanCapacity: viper.GetInt("mindreader-blocks-chan-capacity"),
				WorkingDir:                 buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("mindreader-working-dir")),
				DisableProfiler:            viper.GetBool("mindreader-disable-profiler"),
				StartFailureHandlerFunc:    startUpFunc,
			}), nil
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
			cmd.Flags().String("relayer-source-store", MergedBlocksFilesPath, "Store path url to read batch files from")
			cmd.Flags().Bool("relayer-enable-readiness-probe", true, "Enable relayer's app readiness probe")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
			}), nil
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
			cmd.Flags().String("merger-merged-block-path", MergedBlocksFilesPath, "URL of storage to write merged-block-files to")
			cmd.Flags().String("merger-one-block-path", OneBlockFilesPath, "URL of storage to read one-block-files from")
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
		// FIXME: Lots of config value construction is duplicated across InitFunc and FactoryFunc, how to streamline that
		//        and avoid the duplication? Note that this duplicate happens in many other apps, we might need to re-think our
		//        init flow and call init after the factory and giving it the instantiated app...
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			err := mkdirStorePathIfLocal(buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-merged-block-path")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-one-block-path")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("merger-seen-blocks-file")))
			if err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
			}), nil
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
			cmd.Flags().String("fluxdb-merger-blocks-files-path", MergedBlocksFilesPath, "Store path url to read batch files from")
			cmd.Flags().Bool("fluxdb-enable-dev-mode", false, "Enable dev mode")

			return nil
		},
		InitFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) error {
			return makeDirs([]string{filepath.Join(config.DataDir, "fluxdb")})
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
			}), nil
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
			cmd.Flags().String("kvdb-loader-merged-block-path", MergedBlocksFilesPath, "URL of storage to read one-block-files from")
			cmd.Flags().String("kvdb-loader-kvdb-dsn", KVBDDSN, "kvdb connection string")
			cmd.Flags().String("kvdb-loader-block-stream-addr", RelayerServingAddr, "grpc address of a block stream, usually the relayer grpc address")
			cmd.Flags().Uint64("kvdb-loader-batch-size", 1, "number of blocks batched together for database write")
			cmd.Flags().Uint64("kvdb-loader-start-block-num", 0, "[BATCH] Block number where we start processing")
			cmd.Flags().Uint64("kvdb-loader-stop-block-num", math.MaxUint32, "[BATCH] Block number where we stop processing")
			cmd.Flags().Uint64("kvdb-loader-num-blocks-before-start", 300, "[BATCH] Number of blocks to fetch before start block")
			cmd.Flags().String("kvdb-loader-http-listen-addr", KvdbHTTPServingAddr, "Listen address for /healthz endpoint")
			cmd.Flags().Bool("kvdb-loader-allow-live-on-empty-table", true, "[LIVE] force pipeline creation if live request and table is empty")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			absDataDir, err := filepath.Abs(viper.GetString("global-data-dir"))
			if err != nil {
				return nil, err
			}

			return kvdbLoaderApp.New(&kvdbLoaderApp.Config{
				ChainId:                   viper.GetString("chain-id"),
				ProcessingType:            viper.GetString("kvdb-loader-processing-type"),
				BlockStoreURL:             buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("kvdb-loader-merged-block-path")),
				KvdbDsn:                   fmt.Sprintf(viper.GetString("kvdb-loader-kvdb-dsn"), absDataDir),
				BlockStreamAddr:           viper.GetString("kvdb-loader-block-stream-addr"),
				BatchSize:                 viper.GetUint64("kvdb-loader-batch-size"),
				StartBlockNum:             viper.GetUint64("kvdb-loader-start-block-num"),
				StopBlockNum:              viper.GetUint64("kvdb-loader-stop-block-num"),
				NumBlocksBeforeStart:      viper.GetUint64("kvdb-loader-num-blocks-before-start"),
				AllowLiveOnEmptyTable:     viper.GetBool("kvdb-loader-allow-live-on-empty-table"),
				HTTPListenAddr:            viper.GetString("kvdb-loader-http-listen-addr"),
				Protocol:                  Protocol.String(),
				ParallelFileDownloadCount: 2,
			}), nil
		},
	})

	// Blockmeta
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "blockmeta",
		Title:       "Blockmeta",
		Description: "Serves information about blocks",
		MetricsID:   "blockmeta",
		Logger:      newLoggerDef("github.com/dfuse-io/blockmeta.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("blockmeta-grpc-listen-addr", BlockmetaServingAddr, "GRPC listen on this port")
			cmd.Flags().String("blockmeta-block-stream-addr", RelayerServingAddr, "Websocket endpoint to get a real-time blocks feed")
			cmd.Flags().String("blockmeta-blocks-store", MergedBlocksFilesPath, "URL to source store")
			cmd.Flags().Bool("blockmeta-live-source", true, "Whether we want to connect to a live block source or not, defaults to true")
			cmd.Flags().Bool("blockmeta-enable-readiness-probe", true, "Enable blockmeta's app readiness probe")
			cmd.Flags().StringSlice("blockmeta-eos-api-upstream-addr", []string{NodeosAPIAddr}, "EOS API address to fetch info from running chain, must be in-sync")
			cmd.Flags().StringSlice("blockmeta-eos-api-extra-addr", []string{MindreaderNodeosAPIAddr}, "Additional EOS API address for ID lookups (valid even if it is out of sync or read-only)")
			cmd.Flags().String("blockmeta-kvdb-dsn", BlockmetaDSN, "Kvdb database connection information")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			absDataDir, err := filepath.Abs(viper.GetString("global-data-dir"))
			if err != nil {
				return nil, err
			}

			eosDBClient, err := eosdb.New(fmt.Sprintf(viper.GetString("blockmeta-kvdb-dsn"), absDataDir))
			db := &dblockmeta.EOSBlockmetaDB{
				Driver: eosDBClient,
			}

			return blockmetaApp.New(&blockmetaApp.Config{
				Protocol:                Protocol,
				BlockStreamAddr:         viper.GetString("blockmeta-block-stream-addr"),
				GRPCListenAddr:          viper.GetString("blockmeta-grpc-listen-addr"),
				BlocksStoreURL:          buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("blockmeta-blocks-store")),
				LiveSource:              viper.GetBool("blockmeta-live-source"),
				EnableReadinessProbe:    viper.GetBool("blockmeta-enable-readiness-probe"),
				EOSAPIUpstreamAddresses: viper.GetStringSlice("blockmeta-eos-api-upstream-addr"),
				EOSAPIExtraAddresses:    viper.GetStringSlice("blockmeta-eos-api-extra-addr"),
				KVDBDSN:                 fmt.Sprintf(viper.GetString("blockmeta-kvdb-dsn"), absDataDir),
			}, db), nil
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
			cmd.Flags().String("abicodec-kvdb-dsn", KVBDDSN, "kvdb connection string")
			cmd.Flags().String("abicodec-cache-base-url", "storage/abicahe", "path where the cache store is state")
			cmd.Flags().String("abicodec-cache-file-name", "abicodec_cache.bin", "path where the cache store is state")
			cmd.Flags().Bool("abicodec-export-cache", false, "Export cache and exit")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			absDataDir, err := filepath.Abs(viper.GetString("global-data-dir"))
			if err != nil {
				return nil, err
			}

			return abicodecApp.New(&abicodecApp.Config{
				GRPCListenAddr:       viper.GetString("abicodec-grpc-listen-addr"),
				SearchAddr:           viper.GetString("abicodec-search-addr"),
				KvdbDSN:              fmt.Sprintf(viper.GetString("abicodec-kvdb-dsn"), absDataDir),
				ExportCache:          viper.GetBool("abicodec-export-cache"),
				CacheBaseURL:         buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("abicodec-cache-base-url")),
				CacheStateName:       viper.GetString("abicodec-cache-file-name"),
				EnableReadinessProbe: true,
			}), nil
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
			cmd.Flags().String("search-indexer-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-indexer-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			cmd.Flags().String("search-indexer-writable-path", "search/indexer", "Writable base path for storing index files")
			cmd.Flags().String("search-indexer-indices-store", IndicesFilePath, "Indices path to read or write index shards")
			cmd.Flags().String("search-indexer-blocks-store", MergedBlocksFilesPath, "Path to read blocks files")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return indexerApp.New(&indexerApp.Config{
				Protocol:                            Protocol,
				HTTPListenAddr:                      viper.GetString("search-indexer-http-listen-addr"),
				GRPCListenAddr:                      viper.GetString("search-indexer-grpc-listen-addr"),
				BlockstreamAddr:                     viper.GetString("search-indexer-block-stream-addr"),
				DfuseHooksActionName:                viper.GetString("search-indexer-dfuse-hooks-action-name"),
				IndexingRestrictionsJSON:            viper.GetString("search-indexer-indexing-restrictions-json"),
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
				WritablePath:                        buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-indexer-writable-path")),
				IndicesStoreURL:                     buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-indexer-indices-store")),
				BlocksStoreURL:                      buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-indexer-blocks-store")),
			}), nil
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
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return routerApp.New(&routerApp.Config{
				Dmesh:                modules.SearchDmeshClient,
				Protocol:             Protocol,
				BlockmetaAddr:        viper.GetString("search-router-blockmeta-addr"),
				GRPCListenAddr:       viper.GetString("search-router-listen-addr"),
				HeadDelayTolerance:   viper.GetUint64("search-router-head-delay-tolerance"),
				LibDelayTolerance:    viper.GetUint64("search-router-lib-delay-tolerance"),
				EnableReadinessProbe: viper.GetBool("search-router-enable-readiness-probe"),
				EnableRetry:          viper.GetBool("search-router-enable-retry"),
			}), nil
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
			cmd.Flags().String(p+"indices-store", IndicesFilePath, "GS path to read or write index shards")
			cmd.Flags().String(p+"writable-path", "search/archiver", "Writable base path for storing index files")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
				IndexesStoreURL:         buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-archive-indices-store")),
				IndexesPath:             buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-archive-writable-path")),
			}), nil
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
			cmd.Flags().String("search-live-blocks-store", MergedBlocksFilesPath, "Path to read blocks files")
			cmd.Flags().Duration("search-live-mesh-publish-polling-duration", 0*time.Second, "How often does search live poll dmesh")
			cmd.Flags().Uint64("search-live-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().String("search-live-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-live-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return liveApp.New(&liveApp.Config{
				Dmesh:                    modules.SearchDmeshClient,
				Protocol:                 Protocol,
				ServiceVersion:           viper.GetString("search-mesh-service-version"),
				TierLevel:                viper.GetUint32("search-live-tier-level"),
				GRPCListenAddr:           viper.GetString("search-live-grpc-listen-addr"),
				BlockmetaAddr:            viper.GetString("search-live-blockmeta-addr"),
				LiveIndexesPath:          buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-live-live-indices-path")),
				BlocksStoreURL:           buildStoreURL(viper.GetString("global-data-dir"), viper.GetString("search-live-blocks-store")),
				BlockstreamAddr:          viper.GetString("search-live-block-stream-addr"),
				StartBlockDriftTolerance: viper.GetUint64("search-live-start-block-drift-tolerance"),
				ShutdownDelay:            viper.GetDuration("search-live-shutdown-delay"),
				TruncationThreshold:      viper.GetInt("search-live-truncation-threshold"),
				RealtimeTolerance:        viper.GetDuration("search-live-realtime-tolerance"),
				EnableReadinessProbe:     viper.GetBool("search-live-enable-readiness-probe"),
				PublishDuration:          viper.GetDuration("search-live-mesh-publish-polling-duration"),
				HeadDelayTolerance:       viper.GetUint64("search-live-head-delay-tolerance"),
				IndexingRestrictionsJSON: viper.GetString("search-live-indexing-restrictions-json"),
				DfuseHooksActionName:     viper.GetString("search-live-dfuse-hooks-action-name"),
			}), nil
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
			cmd.Flags().String("search-forkresolver-blocks-store", MergedBlocksFilesPath, "Path to read blocks files")
			cmd.Flags().String("search-forkresolver-indexing-restrictions-json", "", "json-formatted array of items to skip from indexing")
			cmd.Flags().String("search-forkresolver-dfuse-hooks-action-name", "", "The dfuse Hooks event action name to intercept")
			cmd.Flags().Bool("search-forkresolver-enable-readiness-probe", true, "Enable search forlresolver's app readiness probe")

			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
			}), nil
		},
	})

	// eosWS (deprecated app, scheduled to be dismantled and features migrated to dgraphql)
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosws",
		Title:       "EOSWS",
		Description: "Serves websocket and http queries to clients",
		MetricsID:   "eosws",
		Logger:      newLoggerDef("github.com/dfuse-io/dfuse-eosio/eosws.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosws-http-serving-addreosws", EoswsHTTPServingAddr, "Interface to listen on, with main application")
			cmd.Flags().Duration("eosws-graceful-shutdown-delay", time.Second*1, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().String("eosws-block-meta-addr", BlockmetaServingAddr, "Address of the Blockmeta service")
			cmd.Flags().String("eosws-nodeos-rpc-addr", NodeosAPIAddr, "RPC endpoint of the nodeos instance")
			cmd.Flags().String("eosws-kvdb-dsn", KVBDDSN, "kvdb connection string")
			cmd.Flags().Duration("eosws-realtime-tolerance", 15*time.Second, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Int("eosws-blocks-buffer-size", 10, "Number of blocks to keep in memory when initializing")
			cmd.Flags().String("eosws-merged-block-files-path", MergedBlocksFilesPath, "path to merged blocks files")
			cmd.Flags().String("eosws-block-stream-addr", RelayerServingAddr, "gRPC endpoint to get streams of blocks (relayer)")
			cmd.Flags().String("eosws-fluxdb-addr", FluxDBServingAddr, "FluxDB server address")
			cmd.Flags().Bool("eosws-fetch-price", false, "Enable regularly fetching token price from a known source")
			cmd.Flags().Bool("eosws-fetch-vote-tally", false, "Enable regularly fetching vote tally")
			cmd.Flags().String("eosws-search-addr", RouterServingAddr, "search grpc endpoin")
			cmd.Flags().String("eosws-search-addr-secondary", "", "search grpc endpoin")
			cmd.Flags().Duration("eosws-filesource-ratelimit", 2*time.Millisecond, "time to sleep between blocks coming from filesource to control replay speed")
			cmd.Flags().String("eosws-auth-plugin", "null://", "authenticator plugin URI configuration")
			cmd.Flags().String("eosws-metering-plugin", "null://", "metering plugin URI configuration")
			cmd.Flags().Bool("eosws-authenticate-nodeos-api", false, "Gate access to native nodeos APIs with authentication")
			return nil
		},
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return eoswsApp.New(&eoswsApp.Config{
				HTTPListenAddr:              viper.GetString("eosws-http-serving-addreosws"),
				SearchAddr:                  viper.GetString("eosws-search-addr"),
				KVDBDSN:                     fmt.Sprintf(viper.GetString("eosws-kvdb-dsn"), filepath.Join(config.DataDir, "fluxdb")),
				AuthPlugin:                  viper.GetString("eosws-auth-plugin"),
				MeteringPlugin:              viper.GetString("eosws-metering-plugin"),
				NodeosRPCEndpoint:           viper.GetString("eosws-nodeos-rpc-addr"),
				BlockmetaAddr:               viper.GetString("eosws-block-meta-addr"),
				BlockStreamAddr:             viper.GetString("eosws-block-stream-addr"),
				SourceStoreURL:              buildStoreURL(config.DataDir, viper.GetString("eosws-merged-block-files-path")),
				FluxHTTPAddr:                viper.GetString("eosws-fluxdb-addr"),
				UseOpencensusStackdriver:    false,
				FetchPrice:                  viper.GetBool("eosws-fetch-price"),
				FetchVoteTally:              viper.GetBool("eosws-fetch-vote-tally"),
				FilesourceRateLimitPerBlock: viper.GetDuration("eosws-filesource-ratelimit"),
				BlocksBufferSize:            viper.GetInt("eosws-blocks-buffer-size"),
				RealtimeTolerance:           viper.GetDuration("eosws-realtime-tolerance"),
				DataIntegrityProofSecret:    "boo",
			}), nil
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
			cmd.Flags().String("dgraphql-kvdb-dsn", "bigtable://dev.dev/test", "Bigtable database connection information") // Used on EOSIO right now, eventually becomes the reference.
			cmd.Flags().String("dgraphql-auth-plugin", "null://", "Auth plugin, ese dauth repository")
			cmd.Flags().String("dgraphql-metering-plugin", "null://", "Metering plugin, see dmetering repository")
			cmd.Flags().String("dgraphql-network-id", NetworkID, "Network ID, for billing (usually maps namespaces on deployments)")
			cmd.Flags().Duration("dgraphql-graceful-shutdown-delay", 0*time.Millisecond, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().Bool("dgraphql-disable-authentication", false, "disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "flag to override trace id or not")
			cmd.Flags().String("dgraphql-protocol", "eos", "name of the protocol")
			return nil
		},
		InitFunc: nil,
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return dgraphqlEosio.NewApp(&dgraphqlEosio.Config{
				// eos specifc configs
				SearchAddr:    viper.GetString("dgraphql-search-addr"),
				ABICodecAddr:  viper.GetString("dgraphql-abi-addr"),
				BlockMetaAddr: viper.GetString("dgraphql-blockmeta-addr"),
				TokenmetaAddr: viper.GetString("dgraphql-tokenmeta-addr"),
				KVDBDSN:       viper.GetString("dgraphql-kvdb-dsn"),
				Config: dgraphqlApp.Config{
					// base dgraphql configs
					// need to be passed this way because promoted fields
					HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
					GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
					NetworkID:       viper.GetString("dgraphql-network-id"),
					AuthPlugin:      viper.GetString("dgraphql-auth-plugin"),
					MeteringPlugin:  viper.GetString("dgraphql-metering-plugin"),
					OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
					Protocol:        viper.GetString("dgraphql-protocol"),
				},
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
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
			return eosqApp.New(&eosqApp.Config{
				DashboardHTTPListenAddr: config.DashboardHTTPListenAddr,
				HttpListenAddr:          config.EosqHTTPServingAddr,
			}), nil
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
		FactoryFunc: func(config *launcher.RuntimeConfig, modules *launcher.RuntimeModules) (launcher.App, error) {
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
			}), nil
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
