package cli

import "github.com/dfuse-io/dfuse-eosio/tools"

func init() {
	RootCmd.AddCommand(tools.Cmd)
}
