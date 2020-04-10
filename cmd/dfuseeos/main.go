package main

import "github.com/dfuse-io/dfuse-eosio/launcher/cli"

var version = "dev"
var commit = ""

func init() {
	cli.RootCmd.Version = version + "-" + commit
}

func main() {
	cli.Main()
}
