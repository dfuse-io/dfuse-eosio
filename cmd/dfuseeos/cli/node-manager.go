package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/dfuse-io/dfuse-eosio/node-manager/superviser"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	nodeManager "github.com/dfuse-io/node-manager"
	nodeManagerApp "github.com/dfuse-io/node-manager/app/node_manager"
	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/operator"
	"github.com/dfuse-io/node-manager/profiler"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()
	logging.Register("github.com/dfuse-io/dfuse-eosio/node-manager", &appLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "node-manager",
		Title:       "Node manager",
		Description: "Block producing node",
		MetricsID:   "producer",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/node-manager.*", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("node-manager-http-listen-addr", EosManagerAPIAddr, "The dfuse Node Manager API address")
			cmd.Flags().String("node-manager-nodeos-api-addr", NodeosAPIAddr, "Target API address to communicate with underlying superviser")
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
			cmd.Flags().Bool("node-manager-disable-profiler", true, "Disables the node-manager profiler")
			cmd.Flags().StringSlice("node-manager-nodeos-args", []string{}, "Extra arguments to be passed when executing nodeos binary")
			cmd.Flags().Bool("node-manager-log-to-zap", true, "Enables the deepmind logs to be outputted as debug in the zap logger")
			cmd.Flags().String("node-manager-auto-backup-hostname-match", "", "If non-empty, auto-backups will only trigger if os.Hostname() return this value")
			cmd.Flags().String("node-manager-auto-snapshot-hostname-match", "", "If non-empty, auto-snapshots will only trigger if os.Hostname() return this value")
			cmd.Flags().Int("node-manager-auto-backup-modulo", 0, "If non-zero, a backup will be taken every {auto-backup-modulo} block.")
			cmd.Flags().Duration("node-manager-auto-backup-period", 0, "If non-zero, a backup will be taken every period of {auto-backup-period}. Specify 1h, 2h...")
			cmd.Flags().Int("node-manager-auto-snapshot-modulo", 0, "If non-zero, a snapshot will be taken every {auto-snapshot-modulo} block.")
			cmd.Flags().Duration("node-manager-auto-snapshot-period", 0, "If non-zero, a snapshot will be taken every period of {auto-snapshot-period}. Specify 1h, 2h...")
			cmd.Flags().Int("node-manager-number-of-snapshots-to-keep", 0, "if non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
			cmd.Flags().Bool("node-manager-force-production", true, "Forces the production of blocks")
			return nil
		},
		InitFunc: func(modules *launcher.Runtime) error {
			// TODO: check if `~/.dfuse/binaries/nodeos-{ProducerNodeVersion}` exists, if not download from:
			// curl https://abourget.keybase.pub/dfusebox/binaries/nodeos-{ProducerNodeVersion}
			if err := CheckNodeosInstallation(viper.GetString("node-manager-nodeos-path")); err != nil {
				return err
			}
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			hostname, _ := os.Hostname()
			metricID := "producer"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)

			var p *profiler.Profiler
			if !viper.GetBool("node-manager-disable-profiler") {
				p = profiler.GetInstance(appLogger)
			}

			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("node-manager-readiness-max-latency"))
			chainSuperviser, err := superviser.NewSuperviser(
				viper.GetBool("node-manager-debug-deep-mind"),
				metricsAndReadinessManager.UpdateHeadBlock,
				&superviser.SuperviserOptions{
					LocalNodeEndpoint: viper.GetString("node-manager-nodeos-api-addr"),
					ConfigDir:         viper.GetString("node-manager-config-dir"),
					BinPath:           viper.GetString("node-manager-nodeos-path"),
					DataDir:           mustReplaceDataDir(dfuseDataDir, viper.GetString("node-manager-data-dir")),
					Hostname:          hostname,
					ProducerHostname:  viper.GetString("node-manager-producer-hostname"),
					TrustedProducer:   viper.GetString("node-manager-trusted-producer"),
					AdditionalArgs:    viper.GetStringSlice("node-manager-nodeos-args"),
					ForceProduction:   viper.GetBool("node-manager-force-production"),
					LogToZap:          viper.GetBool("node-manager-log-to-zap"),
				}, appLogger)
			if err != nil {
				return nil, fmt.Errorf("unable to create nodeos chain superviser: %w", err)
			}

			chainOperator, err := operator.New(
				appLogger,
				chainSuperviser,
				metricsAndReadinessManager,
				&operator.Options{
					BootstrapDataURL:           viper.GetString("node-manager-bootstrap-data-url"),
					BackupTag:                  viper.GetString("node-manager-backup-tag"),
					BackupStoreURL:             mustReplaceDataDir(dfuseDataDir, viper.GetString("common-backup-store-url")),
					AutoRestoreSource:          viper.GetString("node-manager-auto-restore-source"),
					ShutdownDelay:              viper.GetDuration("node-manager-shutdown-delay"),
					RestoreBackupName:          viper.GetString("node-manager-restore-backup-name"),
					RestoreSnapshotName:        viper.GetString("node-manager-restore-snapshot-name"),
					SnapshotStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("node-manager-snapshot-store-url")),
					StartFailureHandlerFunc:    nil,
					NumberOfSnapshotsToKeep:    viper.GetInt("node-manager-number-of-snapshots-to-keep"),
					EnableSupervisorMonitoring: true,
					Profiler:                   p,
				})

			if err != nil {
				return nil, fmt.Errorf("unable to create chain operator: %w", err)
			}

			return nodeManagerApp.New(&nodeManagerApp.Config{
				ManagerAPIAddress:         viper.GetString("node-manager-http-listen-addr"),
				ConnectionWatchdog:        viper.GetBool("node-manager-connection-watchdog"),
				AutoBackupModulo:          viper.GetInt("node-manager-auto-backup-modulo"),
				AutoBackupPeriod:          viper.GetDuration("node-manager-auto-backup-period"),
				AutoBackupHostnameMatch:   viper.GetString("node-manager-auto-backup-hostname-match"),
				AutoSnapshotModulo:        viper.GetInt("node-manager-auto-snapshot-modulo"),
				AutoSnapshotPeriod:        viper.GetDuration("node-manager-auto-snapshot-period"),
				AutoSnapshotHostnameMatch: viper.GetString("node-manager-auto-snapshot-hostname-match"),
			}, &nodeManagerApp.Modules{
				Operator:                     chainOperator,
				MetricsAndReadinessManager:   metricsAndReadinessManager,
				LaunchConnectionWatchdogFunc: chainSuperviser.LaunchConnectionWatchdog,
			}, appLogger), nil

		},
	})
}
