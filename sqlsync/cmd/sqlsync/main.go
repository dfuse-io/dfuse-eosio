package main

import (
	"github.com/dfuse-io/derr"

	"github.com/abourget/viperbind"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "sqlsync", Short: "SQL syncer for EOSIO state db"}
var serveCmd = &cobra.Command{Use: "start", Short: "starts syncing your chain data to sql", RunE: startRunE}

func main() {

	cobra.OnInitialize(func() {
		viperbind.AutoBind(rootCmd, "SQLSYNC")
	})

	rootCmd.AddCommand(serveCmd)
	derr.Check("running sqlsync", rootCmd.Execute())
}
