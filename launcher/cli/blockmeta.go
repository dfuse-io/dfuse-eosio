package cli

import (
	"strings"

	blockmetaApp "github.com/dfuse-io/blockmeta/app/blockmeta"
	dblockmeta "github.com/dfuse-io/dfuse-eosio/blockmeta"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Blockmeta
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "blockmeta",
		Title:       "Blockmeta",
		Description: "Serves information about blocks",
		MetricsID:   "blockmeta",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/blockmeta.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("blockmeta-grpc-listen-addr", BlockmetaServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("blockmeta-live-source", true, "Whether we want to connect to a live block source or not.")
			cmd.Flags().Bool("blockmeta-enable-readiness-probe", true, "Enable blockmeta's app readiness probe")
			cmd.Flags().StringSlice("blockmeta-eos-api-upstream-addr", []string{NodeosAPIAddr}, "EOS API address to fetch info from running chain, must be in-sync")
			cmd.Flags().StringSlice("blockmeta-eos-api-extra-addr", []string{MindreaderNodeosAPIAddr}, "Additional EOS API address for ID lookups (valid even if it is out of sync or read-only)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			for _, addr := range viper.GetStringSlice("blockmeta-eos-api-upstream-addr") {
				if !strings.HasPrefix(addr, "http") {
					addr = "http://" + addr
				}
				dblockmeta.APIs = append(dblockmeta.APIs, eos.New(addr))
			}
			for _, addr := range viper.GetStringSlice("blockmeta-eos-api-extra-addr") {
				if !strings.HasPrefix(addr, "http") {
					addr = "http://" + addr
				}
				dblockmeta.ExtraAPIs = append(dblockmeta.ExtraAPIs, eos.New(addr))
			}

			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}

			trxdbClient, err := trxdb.New(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-trxdb-dsn")))
			if err != nil {
				return nil, err
			}

			//todo: add db to a modules struct in blockmeta
			db := &dblockmeta.EOSBlockmetaDB{
				Driver: trxdbClient,
			}

			return blockmetaApp.New(&blockmetaApp.Config{
				Protocol:        Protocol,
				BlockStreamAddr: viper.GetString("common-blockstream-addr"),
				GRPCListenAddr:  viper.GetString("blockmeta-grpc-listen-addr"),
				BlocksStoreURL:  mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				LiveSource:      viper.GetBool("blockmeta-live-source"),
			}, db), nil
		},
	})
}
