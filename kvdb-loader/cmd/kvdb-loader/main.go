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

	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/kvdb/eosdb/sql"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "kvdb-loader", Short: "Operate the kvdb loader"}
var initCmd = &cobra.Command{Use: "init", Short: "Init kvdb loader", RunE: runKvdbLoaderInitE}

func main() {
	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "KVDB-LOADER")
	})

	rootCmd.PersistentFlags().String("chain-id", "68c4335171ad518f7ebf8930b8f1740ed9d2638e4a6898a18472f4e360994a8f", "Chain ID")
	rootCmd.PersistentFlags().String("kvdb-dsn", "bigtable://dev.dev/dev?createTables=true", "Storage connection string")
	rootCmd.PersistentFlags().String("protocol", "", "Protocol to load, EOS or ETH")

	rootCmd.AddCommand(initCmd)

	derr.Check("running kvdb-loader", rootCmd.Execute())

}
