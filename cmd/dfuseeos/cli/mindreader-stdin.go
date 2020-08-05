package cli

import (
	"fmt"
	"io"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	nodeManager "github.com/dfuse-io/node-manager"
	nodeMindreaderStdinApp "github.com/dfuse-io/node-manager/app/node_mindreader_stdin"
	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/mindreader"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()
	logging.Register("github.com/dfuse-io/dfuse-eosio/mindreader_stdin", &appLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader-stdin",
		Title:       "deep-mind reader node (stdin)",
		Description: "Blocks reading node, unmanaged, reads deep mind from standard input",
		MetricsID:   "mindreader-stdin",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/mindreader_stdin$", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			archiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))

			consoleReaderFactory := func(reader io.Reader) (mindreader.ConsolerReader, error) {
				return codec.NewConsoleReader(reader)
			}

			consoleReaderBlockTransformer := func(obj interface{}) (*bstream.Block, error) {
				blk, ok := obj.(*pbcodec.Block)
				if !ok {
					return nil, fmt.Errorf("expected *pbcodec.Block, got %T", obj)
				}

				return codec.BlockFromProto(blk)
			}

			metricID := "mindreader-stdin"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("mindreader-readiness-max-latency"))

			return nodeMindreaderStdinApp.New(&nodeMindreaderStdinApp.Config{
				GRPCAddr:                     viper.GetString("mindreader-grpc-listen-addr"),
				ArchiveStoreURL:              archiveStoreURL,
				MergeArchiveStoreURL:         mergeArchiveStoreURL,
				BatchMode:                    viper.GetBool("mindreader-batch-mode"),
				MergeThresholdBlockAge:       viper.GetDuration("mindreader-merge-threshold-block-age"),
				MindReadBlocksChanCapacity:   viper.GetInt("mindreader-blocks-chan-capacity"),
				StartBlockNum:                viper.GetUint64("mindreader-start-block-num"),
				StopBlockNum:                 viper.GetUint64("mindreader-stop-block-num"),
				DiscardAfterStopBlock:        viper.GetBool("mindreader-discard-after-stop-num"),
				FailOnNonContinuousBlocks:    viper.GetBool("mindreader-fail-on-non-contiguous-block"),
				WorkingDir:                   mustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-working-dir")),
				WaitUploadCompleteOnShutdown: viper.GetDuration("mindreader-wait-upload-complete-on-shutdown"),
			}, &nodeMindreaderStdinApp.Modules{
				ConsoleReaderFactory:       consoleReaderFactory,
				ConsoleReaderTransformer:   consoleReaderBlockTransformer,
				MetricsAndReadinessManager: metricsAndReadinessManager,
			}, appLogger), nil
		},
	})
}
