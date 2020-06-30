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
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	"github.com/dfuse-io/bstream"
	_ "github.com/dfuse-io/dauth/authenticator/null" // register authenticator plugin
	_ "github.com/dfuse-io/dauth/ratelimiter/null"   // register ratelimiter plugin
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dfuse-eosio/apiproxy"
	dblockmeta "github.com/dfuse-io/dfuse-eosio/blockmeta"
	boot "github.com/dfuse-io/dfuse-eosio/booter"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/dashboard"
	dgraphqlEosio "github.com/dfuse-io/dfuse-eosio/dgraphql"
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq/app/eosq"
	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	filteringRelayerApp "github.com/dfuse-io/dfuse-eosio/filtering/app/filtering"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	localSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	trxdbLoaderApp "github.com/dfuse-io/dfuse-eosio/trxdb-loader/app/trxdb-loader"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	nodeosManagerApp "github.com/dfuse-io/manageos/app/nodeos_manager"
	nodeosMindreaderApp "github.com/dfuse-io/manageos/app/nodeos_mindreader"
	nodeosMindreaderStdinApp "github.com/dfuse-io/manageos/app/nodeos_mindreader_stdin"
	"github.com/dfuse-io/manageos/mindreader"
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
	"go.uber.org/zap/zapcore"
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
		cmd.Flags().String("search-common-indexed-terms", filtering.DefaultIndexedTerms, "Comma separated list of terms available for indexing. These include: receiver, account, action, auth, scheduled, status, notif, input, event, ram.consumed, ram.released, db.table, db.key, data.[freeform]. Ex: 'data.from', 'data.to', they are those fields dynamically specified by smart contracts as part of their action invocations.")
		cmd.Flags().String("search-common-dfuse-events-action-name", "", "[COMMON] The dfuse Events action name to intercept")
		cmd.Flags().Bool("search-common-dfuse-events-unrestricted", false, "[COMMON] Flag to disable all restrictions of dfuse Events specialize indexing, for example for a private deployment")
		cmd.Flags().String("search-common-indices-store-url", IndicesStoreURL, "[COMMON] Indices path to read or write index shards Used by: search-indexer, search-archiver.")

		return nil
	}

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "node-manager",
		Title:       "Node manager",
		Description: "Block producing node",
		MetricsID:   "producer",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/manageos/app/nodeos_manager", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("node-manager-http-listen-addr", EosManagerAPIAddr, "nodeos manager API address")
			cmd.Flags().String("node-manager-nodeos-api-addr", NodeosAPIAddr, "Target API address to communicate with underlying nodeos")
			cmd.Flags().Bool("node-manager-connection-watchdog", false, "Force-reconnect dead peers automatically")
			cmd.Flags().String("node-manager-config-dir", "./producer", "Directory for config files")
			cmd.Flags().String("node-manager-nodeos-path", NodeosBinPath, "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
			cmd.Flags().String("node-manager-data-dir", "{dfuse-data-dir}/node-manager/data", "Directory for data (nodeos blocks and state)")
			cmd.Flags().String("node-manager-producer-hostname", "", "Hostname that will produce block (other will be paused)")
			cmd.Flags().String("node-manager-trusted-producer", "", "The EOS account name of the Block Producer we trust all blocks from")
			cmd.Flags().Duration("node-manager-readiness-max-latency", 5*time.Second, "/healthz will return error until nodeos head block time is within that duration to now")
			cmd.Flags().String("node-manager-bootstrap-data-url", "", "The bootstrap data URL containing specific chain data used to initialized it.")
			cmd.Flags().String("node-manager-snapshot-store-url", SnapshotsURL, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")
			cmd.Flags().Bool("node-manager-debug-deep-mind", false, "Whether to print all Deepming log lines or not")
			cmd.Flags().String("node-manager-auto-restore-source", "snapshot", "Enables restore from the latest source. Can be either, 'snapshot' or 'backup'. Do not use 'backup' on single block producing node")
			cmd.Flags().String("node-manager-restore-backup-name", "", "If non-empty, the node will be restored from that backup every time it starts.")
			cmd.Flags().String("node-manager-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
			cmd.Flags().Duration("node-manager-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
			cmd.Flags().String("node-manager-backup-tag", "default", "tag to identify the backup")
			cmd.Flags().Bool("node-manager-disable-profiler", true, "Disables the manageos profiler")
			cmd.Flags().StringSlice("node-manager-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().Bool("node-manager-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().String("node-manager-auto-backup-hostname-match", "", "If non-empty, auto-backups will only trigger if os.Hostname() return this value")
			cmd.Flags().String("node-manager-auto-snapshot-hostname-match", "", "If non-empty, auto-snapshots will only trigger if os.Hostname() return this value")
			cmd.Flags().Int("node-manager-auto-backup-modulo", 0, "If non-zero, a backup will be taken every {auto-backup-modulo} block.")
			cmd.Flags().Duration("node-manager-auto-backup-period", 0, "If non-zero, a backup will be taken every period of {auto-backup-period}. Specify 1h, 2h...")
			cmd.Flags().Int("node-manager-auto-snapshot-modulo", 0, "If non-zero, a snapshot will be taken every {auto-snapshot-modulo} block.")
			cmd.Flags().Duration("node-manager-auto-snapshot-period", 0, "If non-zero, a snapshot will be taken every period of {auto-snapshot-period}. Specify 1h, 2h...")
			cmd.Flags().Int("node-manager-number-of-snapshots-to-keep", 0, "if non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
			cmd.Flags().String("node-manager-volume-snapshot-appver", "geth-v1", "[application]-v[version_number], used for persistentVolume snapshots")
			cmd.Flags().Duration("node-manager-auto-volume-snapshot-period", 0, "If non-zero, a volume snapshot will be taken every period of {auto-volume-snapshot-period}. Specify 1h, 2h...")
			cmd.Flags().Int("node-manager-auto-volume-snapshot-modulo", 0, "If non-zero, a volume snapshot will be taken every {auto-volume-snapshot-modulo} blocks. Ex: 500000")
			cmd.Flags().String("node-manager-target-volume-snapshot-specific", "", "Comma-separated list of block numbers where volume snapshots will be done automatically")
			cmd.Flags().Bool("node-manager-force-production", true, "Forces the production of blocks")
			return nil
		},
		InitFunc: func(modules *launcher.RuntimeModules) error {
			// TODO: check if `~/.dfuse/binaries/nodeos-{ProducerNodeVersion}` exists, if not download from:
			// curl https://abourget.keybase.pub/dfusebox/binaries/nodeos-{ProducerNodeVersion}
			if err := CheckNodeosInstallation(viper.GetString("node-manager-nodeos-path")); err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			return nodeosManagerApp.New(&nodeosManagerApp.Config{
				MetricID:                  "producer",
				ManagerAPIAddress:         viper.GetString("node-manager-http-listen-addr"),
				NodeosAPIAddress:          viper.GetString("node-manager-nodeos-api-addr"),
				ConnectionWatchdog:        viper.GetBool("node-manager-connection-watchdog"),
				NodeosConfigDir:           viper.GetString("node-manager-config-dir"),
				NodeosBinPath:             viper.GetString("node-manager-nodeos-path"),
				NodeosDataDir:             MustReplaceDataDir(dfuseDataDir, viper.GetString("node-manager-data-dir")),
				ProducerHostname:          viper.GetString("node-manager-producer-hostname"),
				TrustedProducer:           viper.GetString("node-manager-trusted-producer"),
				ReadinessMaxLatency:       viper.GetDuration("node-manager-readiness-max-latency"),
				ForceProduction:           viper.GetBool("node-manager-force-production"),
				NodeosExtraArgs:           viper.GetStringSlice("node-manager-nodeos-args"),
				BackupStoreURL:            MustReplaceDataDir(dfuseDataDir, viper.GetString("common-backup-store-url")),
				BootstrapDataURL:          viper.GetString("node-manager-bootstrap-data-url"),
				DebugDeepMind:             viper.GetBool("node-manager-debug-deep-mind"),
				LogToZap:                  viper.GetBool("node-manager-log-to-zap"),
				AutoRestoreSource:         viper.GetString("node-manager-auto-restore-source"),
				RestoreBackupName:         viper.GetString("node-manager-restore-backup-name"),
				RestoreSnapshotName:       viper.GetString("node-manager-restore-snapshot-name"),
				SnapshotStoreURL:          MustReplaceDataDir(dfuseDataDir, viper.GetString("node-manager-snapshot-store-url")),
				ShutdownDelay:             viper.GetDuration("node-manager-shutdown-delay"),
				BackupTag:                 viper.GetString("node-manager-backup-tag"),
				AutoBackupModulo:          viper.GetInt("node-manager-auto-backup-modulo"),
				AutoBackupPeriod:          viper.GetDuration("node-manager-auto-backup-period"),
				AutoBackupHostnameMatch:   viper.GetString("node-manager-auto-backup-hostname-match"),
				AutoSnapshotModulo:        viper.GetInt("node-manager-auto-snapshot-modulo"),
				AutoSnapshotPeriod:        viper.GetDuration("node-manager-auto-snapshot-period"),
				AutoSnapshotHostnameMatch: viper.GetString("node-manager-auto-snapshot-hostname-match"),
				NumberOfSnapshotsToKeep:   viper.GetInt("node-manager-number-of-snapshots-to-keep"),
				DisableProfiler:           viper.GetBool("node-manager-disable-profiler"),
				StartFailureHandlerFunc:   nil,
			}), nil

			// Can we detect a nil interface
			return nil, nil
		},
	})
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader-stdin",
		Title:       "deep-mind reader from stdin",
		Description: "deep-mind reader from stdin, does not start nodeos itself",
		MetricsID:   "mindreader-stdin",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/manageos/(app/nodeos_mindreader_stdin|mindreader).*", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			archiveStoreURL := MustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))

			consoleReaderFactory := func(reader io.Reader) (mindreader.ConsolerReader, error) {
				return codec.NewConsoleReader(reader)
			}
			//
			consoleReaderBlockTransformer := func(obj interface{}) (*bstream.Block, error) {
				blk, ok := obj.(*pbcodec.Block)
				if !ok {
					return nil, fmt.Errorf("expected *pbcodec.Block, got %T", obj)
				}

				return codec.BlockFromProto(blk)
			}

			return nodeosMindreaderStdinApp.New(&nodeosMindreaderStdinApp.Config{
				ArchiveStoreURL:            archiveStoreURL,
				MergeArchiveStoreURL:       mergeArchiveStoreURL,
				MergeUploadDirectly:        viper.GetBool("mindreader-merge-and-store-directly"),
				GRPCAddr:                   viper.GetString("mindreader-grpc-listen-addr"),
				MindReadBlocksChanCapacity: viper.GetInt("mindreader-blocks-chan-capacity"),
				WorkingDir:                 MustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-working-dir")),
				DisableProfiler:            viper.GetBool("mindreader-disable-profiler"),
			}, &nodeosMindreaderStdinApp.Modules{
				ConsoleReaderFactory:     consoleReaderFactory,
				ConsoleReaderTransformer: consoleReaderBlockTransformer,
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader",
		Title:       "deep-mind reader node",
		Description: "Blocks reading node",
		MetricsID:   "mindreader",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/manageos/(app/nodeos_mindreader|mindreader).*", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("mindreader-manager-api-addr", EosMindreaderHTTPAddr, "eos-manager API address")
			cmd.Flags().String("mindreader-nodeos-api-addr", MindreaderNodeosAPIAddr, "Target API address to communicate with underlying nodeos")
			cmd.Flags().Bool("mindreader-connection-watchdog", false, "Force-reconnect dead peers automatically")
			cmd.Flags().String("mindreader-config-dir", "./mindreader", "Directory for config files. ")
			cmd.Flags().String("mindreader-nodeos-path", NodeosBinPath, "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
			cmd.Flags().String("mindreader-data-dir", "{dfuse-data-dir}/mindreader/data", "Directory for data (nodeos blocks and state)")
			cmd.Flags().String("mindreader-producer-hostname", "", "Hostname that will produce block (other will be paused)")
			cmd.Flags().String("mindreader-trusted-producer", "", "The EOS account name of the Block Producer we trust all blocks from")
			cmd.Flags().Duration("mindreader-readiness-max-latency", 5*time.Second, "/healthz will return error until nodeos head block time is within that duration to now")
			cmd.Flags().Bool("mindreader-disable-profiler", true, "Disables the manageos profiler")
			cmd.Flags().String("mindreader-snapshot-store-url", SnapshotsURL, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")
			cmd.Flags().String("mindreader-working-dir", "{dfuse-data-dir}/mindreader/work", "Path where mindreader will stores its files")
			cmd.Flags().String("mindreader-backup-tag", "default", "tag to identify the backup")
			cmd.Flags().Bool("mindreader-no-blocks-log", true, "always DELETE blocks.log before running (run without any archive)")
			cmd.Flags().String("mindreader-grpc-listen-addr", MindreaderGRPCAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Uint("mindreader-start-block-num", 0, "Blocks that were produced with smaller block number then the given block num are skipped")
			cmd.Flags().Uint("mindreader-stop-block-num", 0, "Shutdown mindreader when we the following 'stop-block-num' has been reached, inclusively.")
			cmd.Flags().Bool("mindreader-discard-after-stop-num", false, "ignore remaining blocks being processed after stop num (only useful if we discard the mindreader data after reprocessing a chunk of blocks)")
			cmd.Flags().Int("mindreader-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the mindreader. Process will shutdown nodeos/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
			cmd.Flags().Bool("mindreader-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().StringSlice("mindreader-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().String("mindreader-bootstrap-data-url", "", "The bootstrap data URL containing specific chain data used to initialized it.")
			cmd.Flags().Bool("mindreader-debug-deep-mind", false, "Whether to print all Deepming log lines or not")
			cmd.Flags().String("mindreader-auto-restore-source", "snapshot", "Enables restore from the latest source. Can be either, 'snapshot' or 'backup'.")
			cmd.Flags().Duration("mindreader-auto-snapshot-period", 15*time.Minute, "If non-zero, takes state snapshots at this interval")
			cmd.Flags().Duration("mindreader-auto-backup-period", 0, "If non-zero, takes pitreos backups at this interval")
			cmd.Flags().String("mindreader-auto-snapshot-hostname-match", "", "If non-empty, auto-snapshots will only trigger if os.Hostname() return this value")
			cmd.Flags().String("mindreader-auto-backup-hostname-match", "", "If non-empty, auto-backups will only trigger if os.Hostname() return this value")
			cmd.Flags().Int("mindreader-number-of-snapshots-to-keep", 0, "if non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
			cmd.Flags().String("mindreader-restore-backup-name", "", "If non-empty, the node will be restored from that backup every time it starts.")
			cmd.Flags().String("mindreader-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
			cmd.Flags().Duration("mindreader-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
			cmd.Flags().Bool("mindreader-merge-and-store-directly", false, "[BATCH] When enabled, do not write oneblock files, sidestep the merger and write the merged 100-blocks logs directly to --common-blocks-store-url")
			cmd.Flags().Bool("mindreader-start-failure-handler", true, "Enables the startup function handler, that gets called if mindreader fails on startup")
			cmd.Flags().Bool("mindreader-fail-on-non-contiguous-block", false, "Enables the Continuity Checker that stops (or refuses to start) the nodeos if a block was missed. It has a significant performance cost on reprocessing large segments of blocks")
			return nil
		},
		InitFunc: func(modules *launcher.RuntimeModules) error {
			if err := CheckNodeosInstallation(viper.GetString("mindreader-nodeos-path")); err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			archiveStoreURL := MustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))

			var startUpFunc func()
			if viper.GetBool("mindreader-start-failure-handler") {
				startUpFunc = func() {
					userLog.Error(`*********************************************************************************
* Mindreader failed to start nodeos process
* To see nodeos logs...
* DEBUG="mindreader" dfuseeos start
*********************************************************************************`)
					os.Exit(1)
				}

			}
			consoleReaderFactory := func(reader io.Reader) (mindreader.ConsolerReader, error) {
				return codec.NewConsoleReader(reader)
			}
			//
			consoleReaderBlockTransformer := func(obj interface{}) (*bstream.Block, error) {
				blk, ok := obj.(*pbcodec.Block)
				if !ok {
					return nil, fmt.Errorf("expected *pbcodec.Block, got %T", obj)
				}

				return codec.BlockFromProto(blk)
			}

			return nodeosMindreaderApp.New(&nodeosMindreaderApp.Config{
				MetricID:                  "mindreader",
				ManagerAPIAddress:         viper.GetString("mindreader-manager-api-addr"),
				NodeosAPIAddress:          viper.GetString("mindreader-nodeos-api-addr"),
				ConnectionWatchdog:        viper.GetBool("mindreader-connection-watchdog"),
				NodeosConfigDir:           viper.GetString("mindreader-config-dir"),
				NodeosBinPath:             viper.GetString("mindreader-nodeos-path"),
				NodeosDataDir:             MustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-data-dir")),
				ProducerHostname:          viper.GetString("mindreader-producer-hostname"),
				TrustedProducer:           viper.GetString("mindreader-trusted-producer"),
				ReadinessMaxLatency:       viper.GetDuration("mindreader-readiness-max-latency"),
				NodeosExtraArgs:           viper.GetStringSlice("mindreader-nodeos-args"),
				BackupStoreURL:            MustReplaceDataDir(dfuseDataDir, viper.GetString("common-backup-store-url")),
				BackupTag:                 viper.GetString("mindreader-backup-tag"),
				NoBlocksLog:               viper.GetBool("mindreader-no-blocks-log"),
				BootstrapDataURL:          viper.GetString("mindreader-bootstrap-data-url"),
				DebugDeepMind:             viper.GetBool("mindreader-debug-deep-mind"),
				LogToZap:                  viper.GetBool("mindreader-log-to-zap"),
				FailOnNonContinuousBlocks: viper.GetBool("mindreader-fail-on-non-contiguous-block"),

				AutoRestoreSource:          viper.GetString("mindreader-auto-restore-source"),
				AutoSnapshotPeriod:         viper.GetDuration("mindreader-auto-snapshot-period"),
				AutoSnapshotHostnameMatch:  viper.GetString("mindreader-auto-snapshot-hostname-match"),
				AutoBackupPeriod:           viper.GetDuration("mindreader-auto-backup-period"),
				AutoBackupHostnameMatch:    viper.GetString("mindreader-auto-backup-hostname-match"),
				NumberOfSnapshotsToKeep:    viper.GetInt("mindreader-number-of-snapshots-to-keep"),
				RestoreBackupName:          viper.GetString("mindreader-restore-backup-name"),
				RestoreSnapshotName:        viper.GetString("mindreader-restore-snapshot-name"),
				SnapshotStoreURL:           MustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-snapshot-store-url")),
				ShutdownDelay:              viper.GetDuration("mindreader-shutdown-delay"),
				ArchiveStoreURL:            archiveStoreURL,
				MergeArchiveStoreURL:       mergeArchiveStoreURL,
				MergeUploadDirectly:        viper.GetBool("mindreader-merge-and-store-directly"),
				GRPCAddr:                   viper.GetString("mindreader-grpc-listen-addr"),
				StartBlockNum:              viper.GetUint64("mindreader-start-block-num"),
				StopBlockNum:               viper.GetUint64("mindreader-stop-block-num"),
				DiscardAfterStopBlock:      viper.GetBool("mindreader-discard-after-stop-num"),
				MindReadBlocksChanCapacity: viper.GetInt("mindreader-blocks-chan-capacity"),
				WorkingDir:                 MustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-working-dir")),
				DisableProfiler:            viper.GetBool("mindreader-disable-profiler"),
				StartFailureHandlerFunc:    startUpFunc,
			}, &nodeosMindreaderApp.Modules{
				ConsoleReaderFactory:     consoleReaderFactory,
				ConsoleReaderTransformer: consoleReaderBlockTransformer,
			}), nil
		},
	})

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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
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
				SourceStoreURL:   MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
			}), nil
		},
	})

	// Filtering Relayer
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "filtering-relayer",
		Title:       "filtering-relayer",
		Description: "Serves blocks as a filtered stream, from a global deloyed relayer",
		MetricsID:   "filtering-relayer",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/filtering.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("filtering-relayer-grpc-listen-addr", FilteringRelayerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("filtering-relayer-global-relayer-addr", RelayerServingAddr, "Address for global relayer service to connect to")
			cmd.Flags().String("filtering-relayer-filter-in", "", "The CEL filter in expression to use when filtering blocks from the global relayer")
			cmd.Flags().String("filtering-relayer-filter-out", "", "The CEL filter out expression to use when filtering blocks from the global relayer")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return filteringRelayerApp.New(&filteringRelayerApp.Config{
				GRPCListenAddr: viper.GetString("filtering-relayer-grpc-listen-addr"),
				RelayerAddr:    viper.GetString("filtering-relayer-global-relayer-addr"),
				FilterIn:       viper.GetString("filtering-relayer-filter-in"),
				FilterOut:      viper.GetString("filtering-relayer-filter-out"),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return err
			}
			err = mkdirStorePathIfLocal(MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(MustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(MustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")))
			if err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				StorageOneBlockFilesPath:     MustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")),
				TimeBetweenStoreLookups:      viper.GetDuration("merger-time-between-store-lookups"),
				GRPCListenAddr:               viper.GetString("merger-grpc-listen-addr"),
				Live:                         viper.GetBool("merger-process-live-blocks"),
				StartBlockNum:                viper.GetUint64("merger-start-block-num"),
				StopBlockNum:                 viper.GetUint64("merger-stop-block-num"),
				ProgressFilename:             viper.GetString("merger-progress-filename"),
				MinimalBlockNum:              viper.GetUint64("merger-minimal-block-num"),
				WritersLeewayDuration:        viper.GetDuration("merger-writers-leeway"),
				SeenBlocksFile:               MustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
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
				StoreDSN:                   MustReplaceDataDir(absDataDir, viper.GetString("fluxdb-statedb-dsn")),
				BlockStreamAddr:            viper.GetString("common-blockstream-addr"),
				BlockStoreURL:              MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				ThreadsNum:                 viper.GetInt("fluxdb-max-threads"),
				HTTPListenAddr:             viper.GetString("fluxdb-http-listen-addr"),
				ReprocShardStoreURL:        MustReplaceDataDir(dfuseDataDir, viper.GetString("fluxdb-reproc-shard-store-url")),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}

			// FIXME: these names, as they become
			mapper, err := filtering.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create block mapper: %w", err)
			}

			return trxdbLoaderApp.New(&trxdbLoaderApp.Config{
				ChainID:                   viper.GetString("common-chain-id"),
				ProcessingType:            viper.GetString("trxdb-loader-processing-type"),
				BlockStoreURL:             MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				KvdbDSN:                   MustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:           viper.GetString("common-blockstream-addr"),
				BatchSize:                 viper.GetUint64("trxdb-loader-batch-size"),
				StartBlockNum:             viper.GetUint64("trxdb-loader-start-block-num"),
				StopBlockNum:              viper.GetUint64("trxdb-loader-stop-block-num"),
				NumBlocksBeforeStart:      viper.GetUint64("trxdb-loader-num-blocks-before-start"),
				AllowLiveOnEmptyTable:     viper.GetBool("trxdb-loader-allow-live-on-empty-table"),
				HTTPListenAddr:            viper.GetString("trxdb-loader-http-listen-addr"),
				ParallelFileDownloadCount: viper.GetInt("trxdb-loader-parallel-file-download-count"),
			}, &trxdbLoaderApp.Modules{
				BlockMapper: mapper,
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}

			trxdbClient, err := trxdb.New(MustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")))
			if err != nil {
				return nil, err
			}

			//todo: add db to a modules struct in blockmeta
			db := &dblockmeta.EOSBlockmetaDB{
				Driver: trxdbClient,
			}

			return blockmetaApp.New(&blockmetaApp.Config{
				Protocol:                Protocol,
				BlockStreamAddr:         viper.GetString("common-blockstream-addr"),
				GRPCListenAddr:          viper.GetString("blockmeta-grpc-listen-addr"),
				BlocksStoreURL:          MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				LiveSource:              viper.GetBool("blockmeta-live-source"),
				EOSAPIUpstreamAddresses: viper.GetStringSlice("blockmeta-eos-api-upstream-addr"),
				EOSAPIExtraAddresses:    viper.GetStringSlice("blockmeta-eos-api-extra-addr"),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
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
				KvdbDSN:        MustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				CacheBaseURL:   MustReplaceDataDir(dfuseDataDir, viper.GetString("abicodec-cache-base-url")),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			mapper, err := filtering.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create block mapper: %w", err)
			}

			// FIXME: PUT AT THE RIGHT PLACE..
			localSearch.RegisterHandlers(mapper.IndexedTerms())

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

			blocksStoreURL := MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))
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
				WritablePath:          MustReplaceDataDir(dfuseDataDir, viper.GetString("search-indexer-writable-path")),
				IndicesStoreURL:       MustReplaceDataDir(dfuseDataDir, viper.GetString("search-common-indices-store-url")),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
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
				IndexesStoreURL:         MustReplaceDataDir(dfuseDataDir, viper.GetString("search-common-indices-store-url")),
				IndexesPath:             MustReplaceDataDir(dfuseDataDir, viper.GetString("search-archive-writable-path")),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			mapper, err := filtering.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
				viper.GetString("search-common-indexed-terms"),
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create EOS block mapper: %w", err)
			}
			return liveApp.New(&liveApp.Config{
				ServiceVersion:           viper.GetString("search-common-mesh-service-version"),
				TierLevel:                viper.GetUint32("search-live-tier-level"),
				GRPCListenAddr:           viper.GetString("search-live-grpc-listen-addr"),
				BlockmetaAddr:            viper.GetString("common-blockmeta-addr"),
				LiveIndexesPath:          MustReplaceDataDir(dfuseDataDir, viper.GetString("search-live-live-indices-path")),
				BlocksStoreURL:           MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
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
			mapper, err := filtering.NewBlockMapper(
				viper.GetString("search-common-dfuse-events-action-name"),
				viper.GetBool("search-common-dfuse-events-unrestricted"),
				viper.GetString("search-common-action-filter-on-expr"),
				viper.GetString("search-common-action-filter-out-expr"),
				viper.GetString("search-common-indexed-terms"),
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
			cmd.Flags().String("eosws-nodeos-rpc-addr", NodeosAPIAddr, "RPC endpoint of the nodeos instance")
			cmd.Flags().Duration("eosws-realtime-tolerance", 15*time.Second, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Int("eosws-blocks-buffer-size", 10, "Number of blocks to keep in memory when initializing")
			cmd.Flags().String("eosws-fluxdb-addr", FluxDBServingAddr, "FluxDB server address")
			cmd.Flags().Bool("eosws-fetch-price", false, "Enable regularly fetching token price from a known source")
			cmd.Flags().Bool("eosws-fetch-vote-tally", false, "Enable regularly fetching vote tally")
			cmd.Flags().String("eosws-search-addr-secondary", "", "secondary search grpc endpoint")
			cmd.Flags().Duration("eosws-filesource-ratelimit", 2*time.Millisecond, "time to sleep between blocks coming from filesource to control replay speed")
			cmd.Flags().String("eosws-healthz-secret", "", "Secret to access healthz")
			cmd.Flags().String("eosws-data-integrity-proof-secret", "boo", "Data integrity secret for DIPP middleware")
			cmd.Flags().Bool("eosws-authenticate-nodeos-api", false, "Gate access to native nodeos APIs with authentication")
			cmd.Flags().Bool("eosws-use-opencensus-stack-driver", false, "Enables stack driver tracing")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			return eoswsApp.New(&eoswsApp.Config{
				HTTPListenAddr:              viper.GetString("eosws-http-listen-addr"),
				NodeosRPCEndpoint:           viper.GetString("eosws-nodeos-rpc-addr"),
				BlockmetaAddr:               viper.GetString("common-blockmeta-addr"),
				KVDBDSN:                     MustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:             viper.GetString("common-blockstream-addr"),
				SourceStoreURL:              MustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				SearchAddr:                  viper.GetString("common-search-addr"),
				SearchAddrSecondary:         viper.GetString("eosws-search-addr-secondary"),
				FluxHTTPAddr:                viper.GetString("eosws-fluxdb-addr"),
				AuthenticateNodeosAPI:       viper.GetBool("eosws-authenticate-nodeos-api"),
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
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/(dgraphql.*|dfuse-eosio/dgraphql.*)", nil),
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
			dfuseDataDir, err := AbsoluteDfuseDataDir()
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
				KVDBDSN:           MustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
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
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/dashboard.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dashboard-grpc-listen-addr", DashboardGrpcServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dashboard-http-listen-addr", DashboardHTTPListenAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dashboard-eos-node-manager-api-addr", EosManagerAPIAddr, "Address of the nodeos manager api")
			// FIXME: we can re-add when the app actually makes use of it.
			//cmd.Flags().String("dashboard-mindreader-manager-api-addr", MindreaderNodeosAPIAddr, "Address of the mindreader nodeos manager api")
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
			cmd.Flags().String("apiproxy-nodeos-http-addr", NodeosAPIAddr, "Address of a queriable nodeos instance")
			cmd.Flags().String("apiproxy-root-http-addr", EosqHTTPServingAddr, "What to serve at the root of the proxy (defaults to eosq)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			autocertDomains := strings.Split(viper.GetString("apiproxy-autocert-domains"), ",")
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}
			return apiproxy.New(&apiproxy.Config{
				HTTPListenAddr:   viper.GetString("apiproxy-http-listen-addr"),
				HTTPSListenAddr:  viper.GetString("apiproxy-https-listen-addr"),
				AutocertDomains:  autocertDomains,
				AutocertCacheDir: MustReplaceDataDir(dfuseDataDir, viper.GetString("apiproxy-autocert-cache-dir")),
				EoswsHTTPAddr:    viper.GetString("apiproxy-eosws-http-addr"),
				DgraphqlHTTPAddr: viper.GetString("apiproxy-dgraphql-http-addr"),
				NodeosHTTPAddr:   viper.GetString("apiproxy-nodeos-http-addr"),
				RootHTTPAddr:     viper.GetString("apiproxy-root-http-addr"),
			}), nil
		},
	})

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "booter",
		Title:       "Booter",
		Description: "Boots chain baed on provided bootseq",
		MetricsID:   "booter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/booter.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("booter-bootseq", "./bootseq.yaml", "File path tp the desired boot sequence")
			cmd.Flags().String("booter-nodeos-api-addr", fmt.Sprintf("http://localhost%s/", NodeosAPIAddr), "Target API address to communicate with underlying nodeos")
			cmd.Flags().String("booter-data-dir", "{dfuse-data-dir}/booter", "Booter's working directory")
			cmd.Flags().String("booter-vault-file", "", "Wallet file that contains encrypted key material")
			cmd.Flags().String("booter-private-key", "", "Genesis private key")

			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := AbsoluteDfuseDataDir()
			if err != nil {
				return nil, err
			}

			return boot.New(&boot.Config{
				NodeosAPIAddress: viper.GetString("booter-nodeos-api-addr"),
				BootSeqFile:      viper.GetString("booter-bootseq"),
				Datadir:          MustReplaceDataDir(dfuseDataDir, viper.GetString("booter-data-dir")),
				VaultPath:        viper.GetString("booter-vault-file"),
				PrivateKey:       viper.GetString("booter-private-key"),
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

func AbsoluteDfuseDataDir() (string, error) {
	return filepath.Abs(viper.GetString("global-data-dir"))
}
