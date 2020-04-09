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

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/spf13/viper"

	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	eoswsapp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	_ "github.com/dfuse-io/kvdb/eosdb/bigt"
	_ "github.com/eoscanada/eos-go/system"
	_ "github.com/eoscanada/eos-go/token"
	"github.com/spf13/cobra"
	"go.opencensus.io/trace"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{Use: "dfuseWS", Short: "Operate the merger", RunE: wsRunE}

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
func main() {
	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "dfuseWS")
	})

	rootCmd.PersistentFlags().String("listen-addr", "0.0.0.0:8000", "Interface to listen on, with main application")
	rootCmd.PersistentFlags().Duration("graceful-shutdown-delay", time.Second*1, "delay before shutting down, after the health endpoint returns unhealthy")
	rootCmd.PersistentFlags().String("block-meta-addr", "localhost:9001", "Address of the Blockmeta service")
	rootCmd.PersistentFlags().String("api-addr", "http://localhost:8001", "RPC endpoint of the nodeos instance")
	rootCmd.PersistentFlags().String("kvdb-dsn", "bigtable://dev.dev/test", "KVDB storage DSN")
	rootCmd.PersistentFlags().Duration("realtime-tolerance", 15*time.Second, "longest delay to consider this service as real-time(ready) on initialization")
	rootCmd.PersistentFlags().Int("blocks-buffer-size", 300, "Number of blocks to keep in memory when initializing")
	rootCmd.PersistentFlags().String("source-store", "gs://example/blocks", "URL to source store")
	rootCmd.PersistentFlags().String("block-stream-addr", "localhost:9001", "gRPC endpoint to get streams of blocks (relayer)")
	rootCmd.PersistentFlags().String("fluxdb-addr", "http://localhost:8001", "FluxDB server address")
	rootCmd.PersistentFlags().Bool("fetch-price", true, "Enable regularly fetching token price from a known source")
	rootCmd.PersistentFlags().Bool("fetch-vote-tally", true, "Enable regularly fetching vote tally")
	rootCmd.PersistentFlags().String("search-addr", "localhost:9001", "search grpc endpoin")
	rootCmd.PersistentFlags().String("search-addr-secondary", "", "search grpc endpoin")
	rootCmd.PersistentFlags().Duration("filesource-ratelimit", 2*time.Millisecond, "time to sleep between blocks coming from filesource to control replay speed")
	rootCmd.PersistentFlags().String("auth-plugin", "null://", "authenticator plugin URI configuration")
	rootCmd.PersistentFlags().String("metering-plugin", "null://", "metering plugin URI configuration")
	rootCmd.PersistentFlags().String("network-id", "dev", "Network ID (for tracing purposes)")
	rootCmd.PersistentFlags().String("dipp-secret", "this is a long-assed string that will sign our proof requests, once set, do not change it", "Data Integrity Proof Protocol secret")
	rootCmd.PersistentFlags().String("healthz-secret", "dfuse", "healthz endpoint secret")
	rootCmd.PersistentFlags().Bool("authenticate-nodeos-api", false, "Gate access to native nodeos APIs with authentication")

	derr.Check("running merger", rootCmd.Execute())
}

func wsRunE(cmd *cobra.Command, args []string) (err error) {
	flag.Parse()

	setupTracing(trace.ProbabilitySampler(1/5.0), viper.GetString("network-id"))

	go func() {
		listenAddr := "localhost:6060"
		err := http.ListenAndServe(listenAddr, nil)
		if err != nil {
			zlog.Error("unable to start profiling server", zap.Error(err), zap.String("listen_addr", listenAddr))
		}
	}()

	go metrics.ServeMetrics()

	config := &eoswsapp.Config{
		HTTPListenAddr:              viper.GetString("global-listen-addr"),
		NodeosRPCEndpoint:           viper.GetString("global-api-addr"),
		BlockmetaAddr:               viper.GetString("global-block-meta-addr"),
		KVDBDSN:                     viper.GetString("global-kvdb-dsn"),
		BlockStreamAddr:             viper.GetString("global-block-stream-addr"),
		SourceStoreURL:              viper.GetString("global-source-store"),
		SearchAddr:                  viper.GetString("global-search-addr"),
		SearchAddrSecondary:         viper.GetString("global-search-addr-secondary"),
		FluxHTTPAddr:                viper.GetString("global-fluxdb-addr"),
		MeteringPlugin:              viper.GetString("global-metering-plugin"),
		AuthPlugin:                  viper.GetString("global-auth-plugin"),
		UseOpencensusStackdriver:    true,
		FetchPrice:                  viper.GetBool("global-fetch-price"),
		FetchVoteTally:              viper.GetBool("global-fetch-vote-tally"),
		FilesourceRateLimitPerBlock: viper.GetDuration("global-filesource-ratelimit"),
		BlocksBufferSize:            viper.GetInt("global-blocks-buffer-size"),
		RealtimeTolerance:           viper.GetDuration("global-realtime-tolerance"),
		DataIntegrityProofSecret:    viper.GetString("global-dipp-secret"),
		HealthzSecret:               viper.GetString("global-healthz-secret"),
		AuthenticateNodeosAPI:       viper.GetBool("global-authenticate-nodeos-api"),
	}

	app := eoswsapp.New(config)
	derr.Check("eosws app run", app.Run())

	select {
	case <-app.Terminated():
		zlog.Info("eosws is done", zap.Error(app.Err()))
	case sig := <-derr.SetupSignalHandler(viper.GetDuration("global-graceful-shutdown-delay")):
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		app.Shutdown(nil)
	}

	return

}
