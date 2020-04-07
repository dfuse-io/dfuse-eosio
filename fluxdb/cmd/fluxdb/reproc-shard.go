// Copyright 2020 dfuse Platform Inc.
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
	"fmt"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	reprocShardCmd.Flags().Uint32("start-block", 0, "Start processing block logs at this height. Should be on the 100-blocks boundaries.")
	reprocShardCmd.Flags().Uint32("stop-block", 0, "Stop the projection of block logs at target end block. Put on an 100-blocks boundary.")
}

func reprocShard(cmd *cobra.Command, args []string) error {
	blockStoreURL := viper.GetString("global-blocks-store")
	shardStoreURL := viper.GetString("reproc-global-shards-store")
	startBlock := viper.GetUint32("reproc-shard-cmd-start-block")
	stopBlock := viper.GetUint32("reproc-shard-cmd-stop-block")
	shardCount := viper.GetInt("reproc-global-shard-count")
	numOfThread := viper.GetInt("global-threads")

	zlog.Info("starting reproc shard",
		zap.String("block_store_url", blockStoreURL),
		zap.String("shard_store_url", shardStoreURL),
		zap.Uint32("start_block", startBlock),
		zap.Uint32("stop_block", stopBlock),
		zap.Int("shard_count", shardCount),
		zap.Int("number_of_threads", numOfThread),
	)

	blockStore, err := dstore.NewDBinStore(blockStoreURL)
	derr.Check("setting up source blocks store", err)

	shardStore, err := dstore.NewStore(shardStoreURL, "shard.zst", "zstd", true)
	derr.Check(fmt.Sprintf("setting up source shards store %q", shardStoreURL), err)

	shardingPipe := fluxdb.NewSharder(shardStore, shardCount, startBlock, stopBlock)

	src := fluxdb.BuildReprocessingPipeline(shardingPipe, blockStore, uint64(startBlock), 300, numOfThread)

	src.Run()
	<-src.Terminating()

	if err := src.Err(); err != nil {
		// FIXME: This `HasSuffix` is sh**ty, need to replace with a better pattern, `source.Shutdown(nil)` is one of them
		if !strings.HasSuffix(err.Error(), fluxdb.ErrCleanSourceStop.Error()) {
			return err
		}
	}

	return nil
}
