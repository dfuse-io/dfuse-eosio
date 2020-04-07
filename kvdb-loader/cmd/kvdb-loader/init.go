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
	"github.com/dfuse-io/derr"
	kvdbInitApp "github.com/dfuse-io/dfuse-eosio/kvdb-loader/app/kvdb-init"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runKvdbLoaderInitE(cmd *cobra.Command, args []string) (err error) {
	setup()

	app := kvdbInitApp.New(&kvdbInitApp.Config{
		ChainId:  viper.GetString("global-chain-id"),
		KvdbDsn:  viper.GetString("global-kvdb-dsn"),
		Protocol: viper.GetString("global-protocol"),
	})
	err = app.Run()
	derr.Check("running kvdb-loader initiated", err)
	return
}
