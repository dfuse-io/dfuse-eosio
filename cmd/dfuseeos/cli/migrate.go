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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCmd = &cobra.Command{Use: "migrate", Short: "Create chain migration data", RunE: dfuseMigrateE}

func init() {
	RootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().StringP("export-dir", "e", "migration-data", "The directory where to export all the migration data.")
	migrateCmd.Flags().StringP("snapshot-path", "s", "", "The path to the snapshot file used to export the data")
	migrateCmd.Flags().StringP("fallback-config", "f", "", "The path to config file for table decoding to fall back to old abi when failing.")
}

func dfuseMigrateE(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	exportDir := viper.GetString("export-dir")
	if exportDir == "" {
		cliErrorAndExit("The export-dir flag must be set")
	}

	snapshotPath := viper.GetString("snapshot-path")
	if snapshotPath == "" {
		cliErrorAndExit("The snapshot-path flag must be set")
	}

	fallbackConfig := viper.GetString("fallback-config")

	userLog.Printf("Starting migration with snapshot %q into directory %q", snapshotPath, exportDir)

	exporter, err := migrator.NewExporter(snapshotPath, exportDir, fallbackConfig, migrator.WithLogger(zlog))
	if err != nil {
		cliErrorAndExit("Started migration failed: %s", err)
	}

	err = exporter.Export()
	if err != nil {
		cliErrorAndExit("Exporting migration data failed: %s", err)
	}

	return nil
}
