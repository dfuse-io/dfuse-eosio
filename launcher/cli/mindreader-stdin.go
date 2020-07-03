package cli

import (
	"fmt"
	"io"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/manageos/mindreader"
	nodeosMindreaderStdinApp "github.com/dfuse-io/node-manager/app/nodeos_mindreader_stdin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
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
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			archiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))

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
				WorkingDir:                 mustReplaceDataDir(dfuseDataDir, viper.GetString("mindreader-working-dir")),
				DisableProfiler:            viper.GetBool("mindreader-disable-profiler"),
			}, &nodeosMindreaderStdinApp.Modules{
				ConsoleReaderFactory:     consoleReaderFactory,
				ConsoleReaderTransformer: consoleReaderBlockTransformer,
			}), nil
		},
	})
}
