package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/node-manager/superviser"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	nodeManager "github.com/dfuse-io/node-manager"
	nodeMindreaderApp "github.com/dfuse-io/node-manager/app/node_mindreader"
	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/mindreader"
	"github.com/dfuse-io/node-manager/operator"
	"github.com/dfuse-io/node-manager/profiler"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()
	logging.Register("github.com/dfuse-io/dfuse-eosio/mindreader", &appLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader",
		Title:       "deep-mind reader node",
		Description: "Blocks reading node",
		MetricsID:   "mindreader",
		// Now that we also have a `mindreader_stdin` registered logger, we need to pay attention to the actual regexp to ensure we match only our packages!
		Logger: launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/mindreader$", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("mindreader-manager-api-addr", EosMindreaderHTTPAddr, "The dfuse Node Manager API address")
			cmd.Flags().String("mindreader-nodeos-api-addr", MindreaderNodeosAPIAddr, "Target API address to communicate with underlying nodeos")
			cmd.Flags().Bool("mindreader-connection-watchdog", false, "Force-reconnect dead peers automatically")
			cmd.Flags().String("mindreader-config-dir", "./mindreader", "Directory for config files. ")
			cmd.Flags().String("mindreader-nodeos-path", NodeosBinPath, "Path to the nodeos binary. Defaults to the 'nodeos' found in your PATH")
			cmd.Flags().String("mindreader-data-dir", "{dfuse-data-dir}/mindreader/data", "Directory for data (nodeos blocks and state)")
			cmd.Flags().String("mindreader-producer-hostname", "", "Hostname that will produce block (other will be paused)")
			cmd.Flags().String("mindreader-trusted-producer", "", "The EOS account name of the Block Producer we trust all blocks from")
			cmd.Flags().Duration("mindreader-readiness-max-latency", 5*time.Second, "/healthz will return error until nodeos head block time is within that duration to now")
			cmd.Flags().Bool("mindreader-disable-profiler", true, "Disables the node-manager profiler")
			cmd.Flags().String("mindreader-snapshot-store-url", SnapshotsURL, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")
			cmd.Flags().String("mindreader-working-dir", "{dfuse-data-dir}/mindreader/work", "Path where mindreader will stores its files")
			cmd.Flags().String("mindreader-backup-tag", "default", "tag to identify the backup")
			cmd.Flags().Bool("mindreader-no-blocks-log", true, "always DELETE blocks.log before running (run without any archive)")
			cmd.Flags().String("mindreader-grpc-listen-addr", MindreaderGRPCAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Uint("mindreader-start-block-num", 0, "Blocks that were produced with smaller block number then the given block num are skipped")
			cmd.Flags().Uint("mindreader-stop-block-num", 0, "Shutdown mindreader when we the following 'stop-block-num' has been reached, inclusively.")
			cmd.Flags().Bool("mindreader-discard-after-stop-num", false, "ignore remaining blocks being processed after stop num (only useful if we discard the mindreader data after reprocessing a chunk of blocks)")
			cmd.Flags().Int("mindreader-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the mindreader. Process will shutdown superviser/nodeos if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
			cmd.Flags().Bool("mindreader-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().StringSlice("mindreader-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().String("mindreader-bootstrap-data-url", "", "The bootstrap data URL containing specific chain data used to initialized it.")
			cmd.Flags().Bool("mindreader-debug-deep-mind", false, "Whether to print all Deepming log lines or not")
			cmd.Flags().String("mindreader-auto-restore-source", "snapshot", "Enables restore from the latest source. Can be either, 'snapshot' or 'backup'.")
			cmd.Flags().Duration("mindreader-auto-snapshot-period", 15*time.Minute, "If non-zero, takes state snapshots at this interval")
			cmd.Flags().Duration("mindreader-auto-snapshot-modulo", 0, "If non-zero, takes state snapshots at each interval of <modulo> blocks")
			cmd.Flags().Duration("mindreader-auto-backup-period", 0, "If non-zero, takes pitreos backups at this interval")
			cmd.Flags().Duration("mindreader-auto-backup-modulo", 0, "If non-zero, takes pitreos backups at each interval of <modulo> blocks")
			cmd.Flags().String("mindreader-auto-snapshot-hostname-match", "", "If non-empty, auto-snapshots will only trigger if os.Hostname() return this value")
			cmd.Flags().String("mindreader-auto-backup-hostname-match", "", "If non-empty, auto-backups will only trigger if os.Hostname() return this value")
			cmd.Flags().Int("mindreader-number-of-snapshots-to-keep", 0, "If non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
			cmd.Flags().String("mindreader-restore-backup-name", "", "If non-empty, the node will be restored from that backup every time it starts.")
			cmd.Flags().String("mindreader-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
			cmd.Flags().Duration("mindreader-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
			cmd.Flags().Bool("mindreader-merge-and-store-directly", false, "[BATCH] When enabled, do not write oneblock files, sidestep the merger and write the merged 100-blocks logs directly to --common-blocks-store-url")
			cmd.Flags().Bool("mindreader-start-failure-handler", true, "Enables the startup function handler, that gets called if mindreader fails on startup")
			cmd.Flags().Bool("mindreader-fail-on-non-contiguous-block", false, "Enables the Continuity Checker that stops (or refuses to start) the superviser if a block was missed. It has a significant performance cost on reprocessing large segments of blocks")
			return nil
		},
		InitFunc: func(modules *launcher.RuntimeModules) error {
			if err := CheckNodeosInstallation(viper.GetString("mindreader-nodeos-path")); err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			archiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))

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

			var p *profiler.Profiler
			if !viper.GetBool("mindreader-disable-profiler") {
				p = profiler.GetInstance(appLogger)
			}

			metricID := "mindreader"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("mindreader-readiness-max-latency"))

			hostname, _ := os.Hostname()
			chainSuperviser, err := superviser.NewSuperviser(
				viper.GetBool("mindreader-debug-deep-mind"),
				metricsAndReadinessManager.UpdateHeadBlock,
				&superviser.SuperviserOptions{
					LocalNodeEndpoint: viper.GetString("mindreader-manager-api-addr"),
					ConfigDir:         viper.GetString("mindreader-config-dir"),
					BinPath:           viper.GetString("mindreader-nodeos-path"),
					DataDir:           mustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-data-dir")),
					Hostname:          hostname,
					ProducerHostname:  viper.GetString("mindreader-producer-hostname"),
					TrustedProducer:   viper.GetString("mindreader-trusted-producer"),
					AdditionalArgs:    viper.GetStringSlice("mindreader-nodeos-args"),
					LogToZap:          viper.GetBool("mindreader-log-to-zap"),
				},
				appLogger,
			)

			if err != nil {
				return nil, fmt.Errorf("unable to create nodeos chain superviser: %w", err)
			}

			chainOperator, err := operator.New(
				appLogger,
				chainSuperviser,
				metricsAndReadinessManager,
				&operator.Options{
					BootstrapDataURL:           viper.GetString("mindreader-bootstrap-data-url"),
					BackupTag:                  viper.GetString("mindreader-backup-tag"),
					BackupStoreURL:             mustReplaceDataDir(dfuseDataDir, viper.GetString("common-backup-store-url")),
					AutoRestoreSource:          viper.GetString("mindreader-auto-restore-source"),
					ShutdownDelay:              viper.GetDuration("mindreader-shutdown-delay"),
					RestoreBackupName:          viper.GetString("mindreader-restore-snapshot-name"),
					RestoreSnapshotName:        viper.GetString("mindreader-restore-backup-name"),
					SnapshotStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-snapshot-store-url")),
					NumberOfSnapshotsToKeep:    viper.GetInt("mindreader-number-of-snapshots-to-keep"),
					EnableSupervisorMonitoring: false,
					Profiler:                   p,
				})

			if err != nil {
				return nil, fmt.Errorf("unable to create chain operator: %w", err)
			}

			mindreaderPlugin, err := mindreader.NewMindReaderPlugin(
				archiveStoreURL,
				mergeArchiveStoreURL,
				viper.GetBool("mindreader-merge-and-store-directly"),
				mustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-working-dir")),
				consoleReaderFactory,
				consoleReaderBlockTransformer,
				viper.GetUint64("mindreader-start-block-num"),
				viper.GetUint64("mindreader-stop-block-num"),
				viper.GetBool("mindreader-discard-after-stop-num"),
				viper.GetInt("mindreader-blocks-chan-capacity"),
				metricsAndReadinessManager.UpdateHeadBlock,
				chainOperator.SetMaintenance,
				func() {
					chainOperator.Shutdown(nil)
				},
				viper.GetBool("mindreader-fail-on-non-contiguous-block"),
				appLogger,
			)
			if err != nil {
				return nil, err
			}

			chainSuperviser.RegisterPostRestoreHandler(mindreaderPlugin.ResetContinuityChecker)
			chainSuperviser.RegisterLogPlugin(mindreaderPlugin)

			return nodeMindreaderApp.New(&nodeMindreaderApp.Config{
				ManagerAPIAddress:         viper.GetString("mindreader-manager-api-addr"),
				ConnectionWatchdog:        viper.GetBool("mindreader-connection-watchdog"),
				AutoBackupHostnameMatch:   viper.GetString("mindreader-auto-backup-hostname-match"),
				AutoBackupPeriod:          viper.GetDuration("mindreader-auto-backup-period"),
				AutoBackupModulo:          viper.GetInt("mindreader-auto-backup-modulo"),
				AutoSnapshotHostnameMatch: viper.GetString("mindreader-auto-snapshot-hostname-match"),
				AutoSnapshotPeriod:        viper.GetDuration("mindreader-auto-snapshot-period"),
				AutoSnapshotModulo:        viper.GetInt("mindreader-auto-snapshot-modulo"),
				GRPCAddr:                  viper.GetString("mindreader-grpc-listen-addr"),
			}, &nodeMindreaderApp.Modules{
				Operator:                     chainOperator,
				MetricsAndReadinessManager:   metricsAndReadinessManager,
				MindreaderPlugin:             mindreaderPlugin,
				LaunchConnectionWatchdogFunc: chainSuperviser.LaunchConnectionWatchdog,
				StartFailureHandlerFunc:      startUpFunc,
			}, appLogger), nil
		},
	})

}
