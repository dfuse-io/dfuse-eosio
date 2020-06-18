package tools

import "github.com/dfuse-io/dfuse-eosio/launcher/cli"

func init() {
	cli.RootCmd.AddCommand(Cmd)
}
