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
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/dfuse-io/bstream/codecs/deos"

	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/metrics"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "fluxdb", Short: "A brief description of your application"}
var injectCmd = &cobra.Command{Use: "inject", Short: "Start injector pipeline that writes in flux tables and serve at the same time", RunE: inject}
var serveCmd = &cobra.Command{Use: "serve", Short: "Serve a fluxDB server instance (read-only mode)", RunE: serve}

var reprocCmd = &cobra.Command{Use: "reproc", Short: "Reprocessing commands"}
var reprocShardCmd = &cobra.Command{Use: "shard", Short: "Shard blocks logs into a projection of FluxDB Write Requests, to be picked up by the reproc-inject command", RunE: reprocShard}
var reprocInjectCmd = &cobra.Command{Use: "inject", Short: "Inject the shards, starting from a clean slate database.", RunE: reprocInject}

var errCleanExit = errors.New("clean exit")

func main() {
	go metrics.ServeMetrics()

	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "FLUXDB")
	})

	// Commands Tree
	rootCmd.AddCommand(injectCmd)
	rootCmd.AddCommand(reprocCmd)
	rootCmd.AddCommand(serveCmd)
	reprocCmd.AddCommand(reprocShardCmd)
	reprocCmd.AddCommand(reprocInjectCmd)

	// Cmd `fluxdb`
	rootCmd.PersistentFlags().String("store-dsn", "bigtable://dev.dev/dev?createTables=true", "Storage connection string")
	rootCmd.PersistentFlags().Duration("graceful-shutdown-delay", 0, "delay before shutting down, after the health endpoint returns unhealthy")
	rootCmd.PersistentFlags().String("blocks-store", "gs://example/blocks", "dbin blocks store")
	rootCmd.PersistentFlags().String("block-stream-addr", "localhost:9001", "gRPC endpoint to get real-time blocks")
	rootCmd.PersistentFlags().Int("threads", 2, "Number of threads of parallel processing")
	rootCmd.PersistentFlags().Bool("live", true, "Also connect to a live source, can be turn off when doing re-processing")

	// Cmd `fluxdb reproc`
	reprocCmd.PersistentFlags().String("shards-store", "./data/shards", "Storage path where all shard write requests should be written to")
	reprocCmd.PersistentFlags().Int("shard-count", 0, "Number of shards to split in (in 'shard' cmd), or join (in 'inject').")

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	derr.Check("fluxdb", rootCmd.Execute())
}
