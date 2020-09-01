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
			cmd.Flags().String("accounthist-grpc-listen-addr", AccountHistGRPCServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("accounthist-dsn", AccountHistDSN, "kvdb connection string to the accoun thistory database.")
			cmd.Flags().Int("accounthist-shard-num", 0, "[BATCH] Shard number, between 0 and 255 inclusive. Keep default for live process")
			cmd.Flags().Int("accounthist-max-entries-per-account", 1000, "Number of actions to keep in history for each account")
			cmd.Flags().Int("accounthist-flush-blocks-interval", 1000, "Flush to storage each X blocks.  Use 1 when live. Use a high number in batch, serves as checkpointing between restarts.")
			cmd.Flags().Bool("accounthist-enable-injection-mode", true, "Enable mode where blocks are ingested, processed and saved to the database, when false, no write operations happen.")
			cmd.Flags().Bool("accounthist-enable-server-mode", true, "Enable mode where the gRPC server is started and answers request(s), when false, the server is disabled and no requet(s) will be handled.")
			cmd.Flags().Int("accounthist-start-block-num", 0, "[BATCH] Start at this block")
			cmd.Flags().Int("accounthist-stop-block-num", 0, "[BATCH] Stop at this block (exclusive)")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			blockstreamAddr := viper.GetString("common-blockstream-addr")
			shardNum := viper.GetInt("accounthist-shard-num")
			if shardNum > 255 {
				return nil, fmt.Errorf("--accounthist-shard-num must be between 0 and 255 inclusively")
			}

			flushBlocksInterval := viper.GetUint64("accounthist-flush-blocks-interval")
			if flushBlocksInterval == 0 {
				return nil, fmt.Errorf("--accounthist-flush-blocks-interval must be above zero")
			}

			return accounthistApp.New(&accounthistApp.Config{
				GRPCListenAddr:       viper.GetString("accounthist-grpc-listen-addr"),
				KvdbDSN:              mustReplaceDataDir(dfuseDataDir, viper.GetString("accounthist-dsn")),
				BlocksStoreURL:       mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				BlockstreamAddr:      blockstreamAddr,
				EnableInjection:      viper.GetBool("accounthist-enable-injection-mode"),
				EnableServer:         viper.GetBool("accounthist-enable-server-mode"),
				ShardNum:             byte(shardNum),
				MaxEntriesPerAccount: viper.GetUint64("accounthist-max-entries-per-account"),
				FlushBlocksInterval:  flushBlocksInterval,
				StartBlockNum:        viper.GetUint64("accounthist-start-block-num"),
				StopBlockNum:         viper.GetUint64("accounthist-stop-block-num"),
			}, &accounthistApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
				Tracker:     runtime.Tracker,
			}), nil
		},
	})
}
