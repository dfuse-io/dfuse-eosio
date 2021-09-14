package main

import (
	"github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos/cli"
)

// commit sha1 value, injected via go build `ldflags` at build time
var commit = ""

// version value, injected via go build `ldflags` at build time
var version = "dev"

// isDirty value, injected via go build `ldflags` at build time
var isDirty = ""

func init() {
	cli.RootCmd.Version = cli.Version(commit, version, isDirty)
}

func main() {
	cli.Main()
}
