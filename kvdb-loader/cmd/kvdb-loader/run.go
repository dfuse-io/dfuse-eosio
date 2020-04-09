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

package main

import (
	"math"

	"github.com/dfuse-io/derr"
	kvdbLoaderApp "github.com/dfuse-io/dfuse-eosio/kvdb-loader/app/kvdb-loader"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	runCmd.PersistentFlags().String("processing-type", "batch", "The actual processing type to perform, either `live`, `batch` or `patch`")
	runCmd.PersistentFlags().String("source-store", "gs://example/blocks", "GS path to read batch files from")
	runCmd.PersistentFlags().String("block-stream-addr", "localhost:9000", "[LIVE] Address of grpc endpoint")
	runCmd.PersistentFlags().Uint64("batch-size", 100, "DB batch size")
	//runCmd.PersistentFlags().Bool("exit-after-create-tables", false, "Wheter or not the loader should immediately exit after table creation")
	runCmd.PersistentFlags().Uint64("start-block-num", 0, "[BATCH] Block number where we start processing")
	runCmd.PersistentFlags().Uint64("stop-block-num", math.MaxUint32, "[BATCH] Block number where we stop processing")
	runCmd.PersistentFlags().Uint64("num-blocks-before-start", 300, "[BATCH] Number of blocks to fetch before start block")
	runCmd.PersistentFlags().Int("parallel-file-download-count", 12, "Number of threads of parallel file download")
	runCmd.PersistentFlags().Bool("allow-live-on-empty-table", false, "[LIVE] force pipeline creation if live request and table is empty")
	runCmd.PersistentFlags()
}

func runKvdbLoaderRunE(cmd *cobra.Command, args []string) (err error) {
	setup()

	app := kvdbLoaderApp.New(&kvdbLoaderApp.Config{
		ChainId:                   viper.GetString("global-chain-id"),
		KvdbDsn:                   viper.GetString("global-kvdb-dsn"),
		Protocol:                  viper.GetString("global-protocol"),
		ProcessingType:            viper.GetString("run-cmd-processing-type"),
		BlockStoreURL:             viper.GetString("run-cmd-block-store-url"),
		BlockStreamAddr:           viper.GetString("run-cmd-block-stream-addr"),
		BatchSize:                 viper.GetUint64("run-cmd-batch-size"),
		StartBlockNum:             viper.GetUint64("run-cmd-start-block-num"),
		StopBlockNum:              viper.GetUint64("run-cmd-stop-block-num"),
		NumBlocksBeforeStart:      viper.GetUint64("run-cmd-num-blocks-before-start"),
		ParallelFileDownloadCount: viper.GetInt("run-cmd-parallel-file-download-count"),
		AllowLiveOnEmptyTable:     viper.GetBool("run-cmd-allow-live-on-empty-table"),
		HTTPListenAddr:            viper.GetString("run-cmd-http-listen-addr"),
	})

	derr.Check("running kvdb-loader", app.Run())

	select {
	case <-app.Terminated():
		if err = app.Err(); err != nil {
			zlog.Error("kvdb-loader shutdown with error", zap.Error(err))
		}
	case sig := <-derr.SetupSignalHandler(viper.GetDuration("global-graceful-shutdown-delay")):
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		app.Shutdown(nil)
	}

	return
}
