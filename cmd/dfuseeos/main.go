package main

import (
	"github.com/dfuse-io/dfuse-eosio/launcher/cli"
	_ "github.com/dfuse-io/dfuse-eosio/tools"
)

var version = "dev"
var commit = ""

func init() {
	cli.RootCmd.Version = version + "-" + commit
}

func main() {
	cli.Main()
}
