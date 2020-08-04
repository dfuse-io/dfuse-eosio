package cli

import (
	"fmt"

	boot "github.com/dfuse-io/dfuse-eosio/booter"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "booter",
		Title:       "Booter",
		Description: "Boots chain baed on provided bootseq",
		MetricsID:   "booter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/(dfuse-eosio/booter.*|eosio-boot.*)", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("booter-bootseq", "./bootseq.yaml", "File path to the desired boot sequence")
			cmd.Flags().String("booter-nodeos-api-addr", fmt.Sprintf("http://localhost%s/", NodeosAPIAddr), "Target API address to communicate with underlying nodeos")
			cmd.Flags().String("booter-data-dir", "{dfuse-data-dir}/booter", "Booter's working directory")
			cmd.Flags().String("booter-vault-file", "", "Wallet file that contains encrypted key material")
			cmd.Flags().String("booter-private-key", "", "Genesis private key")

			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			return boot.New(&boot.Config{
				NodeosAPIAddress: viper.GetString("booter-nodeos-api-addr"),
				BootSeqFile:      viper.GetString("booter-bootseq"),
				Datadir:          mustReplaceDataDir(dfuseDataDir, viper.GetString("booter-data-dir")),
				VaultPath:        viper.GetString("booter-vault-file"),
				PrivateKey:       viper.GetString("booter-private-key"),
			}), nil
		},
	})
}
