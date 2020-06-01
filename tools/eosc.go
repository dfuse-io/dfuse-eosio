package tools

import (
	eosc "github.com/eoscanada/eosc/eosc/cmd"
	"github.com/spf13/cobra"
)

var eoscCmd = &cobra.Command{Use: "eosc", Short: "manage smart contract"}

func init() {
	eosc.BootCmd.Flags().StringP("global-vault-file", "", "./eosc-vault.json", "Wallet file that contains encrypted key material")

	eoscCmd.AddCommand(eosc.SystemSetcontractCmd)
	eoscCmd.AddCommand(eosc.BootCmd)
	Cmd.AddCommand(eoscCmd)
}
