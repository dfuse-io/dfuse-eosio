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
	"github.com/dfuse-io/derr"
	fluxdbApp "github.com/dfuse-io/dfuse-eosio/fluxdb/app/fluxdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func init() {
	serveCmd.Flags().String("http-listen-addr", ":8080", "Address to server FluxDB queries on")
	serveCmd.Flags().Bool("dev-no-pipeline", false, "Set to true to have a fluxdb not syncing with an actual live block source (**never** use this in prod)")
	serveCmd.Flags().String("network", "dev1", "Network name")
}

func serve(cmd *cobra.Command, args []string) (err error) {
	setupTracing(trace.ProbabilitySampler(1/5.0), viper.GetString("serve-cmd-network"))

	app := fluxdbApp.New(&fluxdbApp.Config{
		EnableServerMode:   true,
		EnableInjectMode:   false,
		StoreDSN:           viper.GetString("global-store-dsn"),
		NetworkID:          viper.GetString("global-network"),
		EnableLivePipeline: viper.GetBool("global-live"),
		BlockStreamAddr:    viper.GetString("global-block-stream-addr"),
		ThreadsNum:         viper.GetInt("global-threads"),
		HTTPListenAddr:     viper.GetString("serve-cmd-http-listen-addr"),
		EnableDevMode:      viper.GetBool("serve-cmd-dev-no-pipeline"),
		BlockStoreURL:      viper.GetString("global-blocks-store"),
	})

	derr.Check("running fluxdb server", app.Run())

	select {
	case <-app.Terminated():
		if err = app.Err(); err != nil {
			zlog.Error("fluxdb injector shutdown with error", zap.Error(err))
		}
	case sig := <-derr.SetupSignalHandler(viper.GetDuration("global-graceful-shutdown-delay")):
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		app.Shutdown(nil)
	}

	return
}
