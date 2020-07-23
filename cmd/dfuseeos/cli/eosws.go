package cli

import (
	"time"

	eoswsApp "github.com/dfuse-io/dfuse-eosio/eosws/app/eosws"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// EOSWS
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosws",
		Title:       "EOSWS",
		Description: "Serves websocket and http queries to clients",
		MetricsID:   "eosws",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/eosws.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosws-http-listen-addr", EoswsHTTPServingAddr, "Address to listen for incoming http requests")
			cmd.Flags().String("eosws-nodeos-rpc-addr", NodeosAPIAddr, "RPC endpoint of the nodeos instance")
			cmd.Flags().Duration("eosws-realtime-tolerance", 15*time.Second, "longest delay to consider this service as real-time(ready) on initialization")
			cmd.Flags().Int("eosws-blocks-buffer-size", 10, "Number of blocks to keep in memory when initializing")
			cmd.Flags().String("eosws-fluxdb-addr", FluxDBHTTPServingAddr, "FluxDB server address")
			cmd.Flags().Bool("eosws-fetch-price", false, "Enable regularly fetching token price from a known source")
			cmd.Flags().Bool("eosws-fetch-vote-tally", false, "Enable regularly fetching vote tally")
			cmd.Flags().String("eosws-search-addr-secondary", "", "secondary search grpc endpoint")
			cmd.Flags().Duration("eosws-filesource-ratelimit", 2*time.Millisecond, "time to sleep between blocks coming from filesource to control replay speed")
			cmd.Flags().String("eosws-healthz-secret", "", "Secret to access healthz")
			cmd.Flags().String("eosws-data-integrity-proof-secret", "boo", "Data integrity secret for DIPP middleware")
			cmd.Flags().Bool("eosws-authenticate-nodeos-api", false, "Gate access to native superviser APIs with authentication")
			cmd.Flags().Bool("eosws-use-opencensus-stack-driver", false, "Enables stack driver tracing")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			return eoswsApp.New(&eoswsApp.Config{
				HTTPListenAddr:              viper.GetString("eosws-http-listen-addr"),
				NodeosRPCEndpoint:           viper.GetString("eosws-nodeos-rpc-addr"),
				BlockmetaAddr:               viper.GetString("common-blockmeta-addr"),
				KVDBDSN:                     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")),
				BlockStreamAddr:             viper.GetString("common-blockstream-addr"),
				SourceStoreURL:              mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				SearchAddr:                  viper.GetString("common-search-addr"),
				SearchAddrSecondary:         viper.GetString("eosws-search-addr-secondary"),
				FluxHTTPAddr:                viper.GetString("eosws-fluxdb-addr"),
				AuthenticateNodeosAPI:       viper.GetBool("eosws-authenticate-nodeos-api"),
				MeteringPlugin:              viper.GetString("common-metering-plugin"),
				AuthPlugin:                  viper.GetString("common-auth-plugin"),
				UseOpencensusStackdriver:    viper.GetBool("eosws-use-opencensus-stack-driver"),
				FetchPrice:                  viper.GetBool("eosws-fetch-price"),
				FetchVoteTally:              viper.GetBool("eosws-fetch-vote-tally"),
				FilesourceRateLimitPerBlock: viper.GetDuration("eosws-filesource-ratelimit"),
				BlocksBufferSize:            viper.GetInt("eosws-blocks-buffer-size"),
				RealtimeTolerance:           viper.GetDuration("eosws-realtime-tolerance"),
				DataIntegrityProofSecret:    viper.GetString("eosws-data-integrity-proof-secret"),
				HealthzSecret:               viper.GetString("eosws-healthz-secret"),
			}), nil
		},
	})
}
