package cli

import (
	eosrest "github.com/dfuse-io/dfuse-eosio/eosrest/app/eosrest"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// EOSWS
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosrest",
		Title:       "EOSRest",
		Description: "Serves HTTP queries to clients",
		MetricsID:   "eosrest",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/eosrest.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosrest-http-listen-addr", EosrestHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("eosrest-nodeos-rpc-addr", NodeosAPIAddr, "RPC endpoint of the nodeos instance")
			cmd.Flags().String("eosrest-statedb-http-addr", StateDBHTTPServingAddr, "StateDB HTTP server address")
			cmd.Flags().String("eosrest-statedb-grpc-addr", StateDBGRPCServingAddr, "StateDB GRPC server address")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			return eosrest.New(&eosrest.Config{
				HTTPListenAddr:    viper.GetString("eosrest-http-listen-addr"),
				NodeosRPCEndpoint: viper.GetString("eosrest-nodeos-rpc-addr"),
				StateDBHTTPAddr:   viper.GetString("eosrest-statedb-http-addr"),
				StateDBGRPCAddr:   viper.GetString("eosrest-statedb-grpc-addr"),
				BlockmetaAddr:     viper.GetString("common-blockmeta-addr"),
				KVDBDSN:           mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				SearchAddr:        viper.GetString("common-search-addr"),
				MeteringPlugin:    viper.GetString("common-metering-plugin"),
				AuthPlugin:        viper.GetString("common-auth-plugin"),
			}), nil
		},
	})
}
