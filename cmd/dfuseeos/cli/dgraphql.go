package cli

import (
	dgraphqlEosio "github.com/dfuse-io/dfuse-eosio/dgraphql"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Dgraphql
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dgraphql",
		Title:       "GraphQL",
		Description: "Serves GraphQL queries to clients",
		MetricsID:   "dgraphql",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dgraphql.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dgraphql-http-addr", DgraphqlHTTPServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dgraphql-grpc-addr", DgraphqlGRPCServingAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dgraphql-abi-addr", ABICodecServingAddr, "Base URL for abicodec service")
			cmd.Flags().Duration("dgraphql-graceful-shutdown-delay", 0, "delay before shutting down, after the health endpoint returns unhealthy")
			cmd.Flags().Bool("dgraphql-disable-authentication", false, "disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "flag to override trace id or not")
			cmd.Flags().String("dgraphql-protocol", "eos", "name of the protocol")
			cmd.Flags().String("dgraphql-auth-url", JWTIssuerURL, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("dgraphql-api-key", DgraphqlAPIKey, "API key used in graphiql")
			cmd.Flags().String("dgraphql-tokenmeta-addr", TokenmetaGRPCServingAddr, "Tokenmeta client endpoint url")
			cmd.Flags().String("dgraphql-accounthist-account-addr", AccountHistGRPCServingAddr, "Account history account indexed server client endpoint url, empty string disables the operation")
			cmd.Flags().String("dgraphql-accounthist-account-contract-addr", "", "Account history account-contract indexed server client endpoint url, empty string disables the operation")

			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			return dgraphqlEosio.NewApp(&dgraphqlEosio.Config{
				// eos specifc configs
				SearchAddr:                     viper.GetString("common-search-addr"),
				ABICodecAddr:                   viper.GetString("dgraphql-abi-addr"),
				BlockMetaAddr:                  viper.GetString("common-blockmeta-addr"),
				TokenmetaAddr:                  viper.GetString("dgraphql-tokenmeta-addr"),
				AccountHistAccountAddr:         viper.GetString("dgraphql-accounthist-account-addr"),
				AccountHistAccountContractAddr: viper.GetString("dgraphql-accounthist-account-contract-addr"),
				KVDBDSN:                        mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				RatelimiterPlugin:              viper.GetString("common-ratelimiter-plugin"),
				Config: dgraphqlApp.Config{
					// base dgraphql configs
					// need to be passed this way because promoted fields
					HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
					GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
					AuthPlugin:      viper.GetString("common-auth-plugin"),
					MeteringPlugin:  viper.GetString("common-metering-plugin"),
					NetworkID:       viper.GetString("common-network-id"),
					OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
					Protocol:        viper.GetString("dgraphql-protocol"),
					JwtIssuerURL:    viper.GetString("dgraphql-auth-url"),
					APIKey:          viper.GetString("dgraphql-api-key"),
				},
			})
		},
	})
}
