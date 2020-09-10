package main

import (
	"github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos/cli"

	_ "github.com/dfuse-io/dauth/ratelimiter/null"
	_ "github.com/dfuse-io/dauth/ratelimiter/olric"
)

var version = "dev"
var commit = ""

func init() {
	cli.RootCmd.Version = version + "-" + commit
}

func main() {
	cli.Main()
}
