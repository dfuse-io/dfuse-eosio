package main

import (
	_ "github.com/dfuse-io/dauth/authenticator/null"   // auth null plugin
	_ "github.com/dfuse-io/dauth/authenticator/secret" // auth secret/hard-coded plugin
	_ "github.com/dfuse-io/dauth/ratelimiter/null"     // ratelimiter plugin

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
