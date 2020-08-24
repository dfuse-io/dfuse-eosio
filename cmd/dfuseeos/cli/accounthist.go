package cli

import (
	"fmt"

	accounthistApp "github.com/dfuse-io/dfuse-eosio/accounthist/app/accounthist"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Accounthist
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "accounthist",
		Title:       "Account History Server",
		Description: "Serves X most recent actions for each account",
		MetricsID:   "accounthist",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/accounthist/.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("accounthist-grpc-listen-addr", AccountHistServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("accounthist-dsn", AccountHistDSN, "kvdb connection string to the accoun thistory database.")
			cmd.Flags().Int("accounthist-shard-num", 0, "[BATCH] Shard number, between 0 and 255 inclusive. Keep default for live process")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			blockstreamAddr := viper.GetString("common-blockstream-addr")
			shardNum := viper.GetInt("accounthist-shard-num")
			if shardNum > 255 {
				return nil, fmt.Errorf("--accounthist-shard-num must be between 0 and 255 inclusively")
			}

			return accounthistApp.New(&accounthistApp.Config{
				GRPCListenAddr:  viper.GetString("accounthist-grpc-listen-addr"),
				KvdbDSN:         mustReplaceDataDir(dfuseDataDir, viper.GetString("accounthist-dsn")),
				BlocksStoreURL:  mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				BlockstreamAddr: blockstreamAddr,
				ShardNum:        byte(shardNum),
			}, &accounthistApp.Modules{}), nil
		},
	})
}
