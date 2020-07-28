package cli

import (
	"strings"

	"github.com/dfuse-io/dfuse-eosio/apiproxy"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	//API proxy
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "apiproxy",
		Title:       "API Proxy",
		Description: "Reverse proxies all API services under one port",
		MetricsID:   "apiproxy",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/apiproxy.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("apiproxy-http-listen-addr", APIProxyHTTPListenAddr, "HTTP Listener address")
			cmd.Flags().String("apiproxy-https-listen-addr", "", "If non-empty, will listen for HTTPS connections on this address")
			cmd.Flags().String("apiproxy-autocert-domains", "", "If non-empty, requests certificates from Let's Encrypt for this comma-separated list of domains")
			cmd.Flags().String("apiproxy-autocert-cache-dir", "{dfuse-data-dir}/api-proxy", "Path to directory where certificates will be saved to disk")
			cmd.Flags().String("apiproxy-eosws-http-addr", EoswsHTTPServingAddr, "Target address of the eosws API endpoint")
			cmd.Flags().String("apiproxy-eosrest-http-addr", EosrestHTTPServingAddr, "Target address of the eosws API endpoint")
			cmd.Flags().String("apiproxy-dgraphql-http-addr", DgraphqlHTTPServingAddr, "Target address of the dgraphql API endpoint")
			cmd.Flags().String("apiproxy-nodeos-http-addr", NodeosAPIAddr, "Address of a queriable nodeos instance")
			cmd.Flags().String("apiproxy-root-http-addr", EosqHTTPServingAddr, "What to serve at the root of the proxy (defaults to eosq)")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			autocertDomains := strings.Split(viper.GetString("apiproxy-autocert-domains"), ",")
			dfuseDataDir := runtime.AbsDataDir
			return apiproxy.New(&apiproxy.Config{
				HTTPListenAddr:   viper.GetString("apiproxy-http-listen-addr"),
				HTTPSListenAddr:  viper.GetString("apiproxy-https-listen-addr"),
				AutocertDomains:  autocertDomains,
				AutocertCacheDir: mustReplaceDataDir(dfuseDataDir, viper.GetString("apiproxy-autocert-cache-dir")),
				EoswsHTTPAddr:    viper.GetString("apiproxy-eosws-http-addr"),
				EosrestHTTPAddr:  viper.GetString("apiproxy-eosrest-http-addr"),
				DgraphqlHTTPAddr: viper.GetString("apiproxy-dgraphql-http-addr"),
				NodeosHTTPAddr:   viper.GetString("apiproxy-nodeos-http-addr"),
				RootHTTPAddr:     viper.GetString("apiproxy-root-http-addr"),
			}), nil
		},
	})
}
