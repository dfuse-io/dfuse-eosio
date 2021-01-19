package main

import (
	"github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos/cli"
)

var version = "dev"
var commit = ""

func init() {
	cli.RootCmd.Version = version + "-" + commit
}

func main() {
	cli.Main()
}
