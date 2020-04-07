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

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	reprocInjectCmd.PersistentFlags().Int("shard-index", -1, "Index of the shard to perform injection for, should be lower than shard-count")
}

func reprocInject(cmd *cobra.Command, args []string) error {
	shardIndex := viper.GetInt("reproc-inject-cmd-shard-index")
	shardCount := viper.GetInt("reproc-global-shard-count")
	shardStoreURL := viper.GetString("reproc-global-shards-store")
	storeDSN := viper.GetString("global-store-dsn")

	zlog.Info("starting reproc injector",
		zap.Int("shard_indext", shardIndex),
		zap.Int("shard_count", shardCount),
		zap.String("shard_store", shardStoreURL),
		zap.String("store_dsn", storeDSN),
	)

	if shardCount == 0 {
		derr.Check("--shard-count", fmt.Errorf("required"))
	}

	kvStore, err := fluxdb.NewKVStore(storeDSN)
	derr.Check("setting up database", err)

	db := fluxdb.New(kvStore)

	db.SetSharding(shardIndex, shardCount)
	err = db.CheckCleanDBForSharding()
	derr.Check("ensure DB is clean before injecting shards", err)

	shardStoreFullURL := shardStoreURL + "/" + fmt.Sprintf("%03d", shardIndex)
	zlog.Info("using shards url", zap.String("store_url", shardStoreFullURL))

	shardStore, err := dstore.NewStore(shardStoreFullURL, "shard.zst", "zstd", true)
	derr.Check(fmt.Sprintf("setting up source shards store %q", shardStoreFullURL), err)

	shardInjector := fluxdb.NewShardInjector(shardStore, db)

	if err := shardInjector.Run(); err != nil {
		return fmt.Errorf("running injector: %s", err)
	}

	if err := db.VerifyAllShardsWritten(); err != nil {
		return fmt.Errorf("verify all shards written: %s", err)
	}

	return nil
}
