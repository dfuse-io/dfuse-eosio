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
	_ "net/http/pprof"
	"time"

	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	"github.com/spf13/cobra"
	_ "go.uber.org/automaxprocs"
)

var rootCmd = &cobra.Command{Use: "dgraghql", Short: "Operate the dgraphql servers"}
var eosioCmd = &cobra.Command{Use: "eosio", Short: "Run dgraphql for eos", RunE: runEosioE}

func main() {
	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "DGRAPHQL")
	})
	rootCmd.PersistentFlags().String("http-addr", ":8080", "TCP Listener addr for http")
	rootCmd.PersistentFlags().String("grpc-addr", ":9000", "TCP Listener addr for gRPC")
	rootCmd.PersistentFlags().String("search-addr", ":9001", "Base URL for search service")
	rootCmd.PersistentFlags().String("search-addr-v2", "", "Base URL for search service")
	rootCmd.PersistentFlags().String("kvdb-dsn", "bigtable://dev.dev/test", "Bigtable database connection information") // Used on EOSIO right now, eventually becomes the reference.
	rootCmd.PersistentFlags().String("auth-plugin", "null://", "Auth plugin, ese dauth repository")
	rootCmd.PersistentFlags().String("metering-plugin", "null://", "Metering plugin, see dmetering repository")
	rootCmd.PersistentFlags().String("network-id", "eos-mainnet", "Network ID, for billing (usually maps namespaces on deployments)")
	rootCmd.PersistentFlags().Duration("graceful-shutdown-delay", 0*time.Millisecond, "delay before shutting down, after the health endpoint returns unhealthy")
	rootCmd.PersistentFlags().Bool("disable-authentication", false, "disable authentication for both grpc and http services")

	rootCmd.AddCommand(eosioCmd)

	derr.Check("running dgraphql", rootCmd.Execute())
}
