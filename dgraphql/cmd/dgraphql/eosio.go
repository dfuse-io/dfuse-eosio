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
	dgraphqlApp "github.com/dfuse-io/dfuse-eosio/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dmetering"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	eosioCmd.Flags().String("abi-addr", "localhost:9001", "Base URL for abicodec service")
	eosioCmd.Flags().String("block-meta-addr", "localhost:9001", "Base URL for blockmeta service")
	eosioCmd.Flags().String("tokenmeta-addr", "localhost:9001", "Base URL tokenmeta servgit pulice")
}

func runEosioE(cmd *cobra.Command, args []string) (err error) {
	setup()

	app := dgraphqlApp.New(&dgraphqlApp.Config{
		HTTPListenAddr:  viper.GetString("global-http-addr"),
		GRPCListenAddr:  viper.GetString("global-grpc-addr"),
		SearchAddr:      viper.GetString("global-search-addr"),
		SearchAddrV2:    viper.GetString("global-search-addr-v2"),
		KVDBDSN:         viper.GetString("global-kvdb-dsn"),
		NetworkID:       viper.GetString("global-network-id"),
		AuthPlugin:      viper.GetString("global-auth-plugin"),
		MeteringPlugin:  viper.GetString("global-metering-plugin"),
		ABICodecAddr:    viper.GetString("eosio-cmd-abi-addr"),
		BlockMetaAddr:   viper.GetString("eosio-cmd-block-meta-addr"),
		TokenmetaAddr:   viper.GetString("eosio-cmd-tokenmeta-addr"),
		OverrideTraceID: true,
	})
	derr.Check("running dgraphql EOS", app.Run())

	select {
	case <-app.Terminated():
		if err = app.Err(); err != nil {
			zlog.Error("search indexer shutdown with error", zap.Error(err))
		}
	case sig := <-derr.SetupSignalHandler(viper.GetDuration("global-graceful-shutdown-delay")):
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		app.Shutdown(nil)
	}

	dmetering.WaitToFlush()

	return
}
