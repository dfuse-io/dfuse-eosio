// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"github.com/dfuse-io/dfuse-eosio/booter/migrator"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dgrpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var migrateCmd = &cobra.Command{Use: "migrate", Short: "Create chain migration data", RunE: dfuseMigrateE}

func init() {
	RootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().StringP("export-dir", "e", "migration-data", "The directory where to export all the migration data.")
	migrateCmd.Flags().Uint32P("irreversible-block-num", "i", 0, "The irreversible block at which migration should be taken, it's your responsibility for now to ensure the block num received is irreversible.")
}

func dfuseMigrateE(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	exportDir := viper.GetString("export-dir")
	if exportDir == "" {
		cliErrorAndExit("The export-dir flag must be set")
	}

	irrBlockNum := viper.GetUint32("irreversible-block-num")
	if irrBlockNum <= 1 {
		cliErrorAndExit("The irreversible-block-num flag must be set to a block higher than 1")
	}

	userLog.Printf("Starting migration at irreversible block num #%d into directory %q", irrBlockNum, exportDir)

	fluxdbGRPCListenAddr := viper.GetString("fluxdb-grpc-listen-addr")

	userLog.Debug("creating grpc connection to fluxdb", zap.String("addr", fluxdbGRPCListenAddr))
	conn, err := dgrpc.NewInternalClient(fluxdbGRPCListenAddr)
	if err != nil {
		cliErrorAndExit("Unable to connect to fluxdb GRPC endpoint: %s", err)
	}

	userLog.Debug("performing actual migration")
	exporter := migrator.NewExporter(cmd.Context(), pbfluxdb.NewStateClient(conn), exportDir, uint64(irrBlockNum), userLog)

	err = exporter.Export()
	if err != nil {
		cliErrorAndExit("Exporting migration data failed: %s", err)
	}

	return nil
}
