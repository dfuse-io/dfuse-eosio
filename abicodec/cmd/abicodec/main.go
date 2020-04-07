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
	"time"

	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	_ "github.com/dfuse-io/kvdb/eosdb/bigt"

	//	_ "github.com/dfuse-io/kvdb/eosdb/sql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{Use: "abicodec", Short: "Operate the abicodec", RunE: runRootE}

func main() {
	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "ABICODEC")
	})

	rootCmd.PersistentFlags().String("grpc-listen-addr", ":9000", "TCP Listener addr for gRPC")
	rootCmd.PersistentFlags().String("search-addr", ":7004", "Base URL for search service")
	rootCmd.PersistentFlags().String("kvdb-dsn", "bigtable://dev.dev/test", "Bigtable database connection information") // Used on EOSIO right now, eventually becomes the reference.
	rootCmd.PersistentFlags().String("cache-base-url", "file:///tmp", "path where the cache store is state")
	rootCmd.PersistentFlags().String("cache-file-name", "abicodec_cache.bin", "path where the cache store is state")
	rootCmd.PersistentFlags().Bool("export-cache", false, "Export cache and exit")
	rootCmd.PersistentFlags().String("export-cache-url", "file:///tmp", "path where to export the cache store")
	rootCmd.PersistentFlags().Duration("graceful-shutdown-delay", 0*time.Millisecond, "delay before shutting down, after the health endpoint returns unhealthy")

	derr.Check("running abicodec", rootCmd.Execute())
}

func runRootE(cmd *cobra.Command, args []string) (err error) {
	setup()

	app := abicodecApp.New(&abicodecApp.Config{
		GRPCListenAddr: viper.GetString("global-grpc-listen-addr"),
		SearchAddr:     viper.GetString("global-search-addr"),
		KvdbDSN:        viper.GetString("global-kvdb-dsn"),
		CacheBaseURL:   viper.GetString("global-cache-base-url"),
		CacheStateName: viper.GetString("global-cache-file-name"),
		ExportCache:    viper.GetBool("global-export-cache"),
		ExportCacheURL: viper.GetString("global-export-cache-url"),
	})
	derr.Check("running abicodec", app.Run())

	select {
	case <-app.Terminated():
		if err = app.Err(); err != nil {
			zlog.Error("abicodec shutdown with error", zap.Error(err))
		}
	case sig := <-derr.SetupSignalHandler(viper.GetDuration("global-graceful-shutdown-delay")):
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		app.Shutdown(nil)
	}

	return
}
