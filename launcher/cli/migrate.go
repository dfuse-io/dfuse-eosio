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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dfuse-io/dfuse-eosio/launcher"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dgrpc"
	"github.com/eoscanada/eos-go"
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

	// We really initialize the config so that flags are injected in and we can properly
	// resolve them. We don't really need the actual config object for now.
	_, err := launcher.NewConfig(viper.GetString("global-config-file"), true)
	if err != nil {
		cliErrorAndExit("Unable to read provided config file: %s", err)
	}

	fluxdbGRPCListenAddr := viper.GetString("fluxdb-grpc-listen-addr")

	userLog.Debug("creating grpc connection to fluxdb", zap.String("addr", fluxdbGRPCListenAddr))
	conn, err := dgrpc.NewInternalClient(fluxdbGRPCListenAddr)
	if err != nil {
		cliErrorAndExit("Unable to connect to fluxdb GRPC endpoint: %s", err)
	}

	userLog.Debug("performing actual migration")
	migrater := migrater{
		ctx:         context.Background(),
		fluxdb:      pbfluxdb.NewStateClient(conn),
		exportDir:   exportDir,
		irrBlockNum: uint64(irrBlockNum),
	}

	err = migrater.migrate()
	if err != nil {
		cliErrorAndExit("Exporting migration data failed: %s", err)
	}

	return nil
}

type migrater struct {
	ctx         context.Context
	fluxdb      pbfluxdb.StateClient
	exportDir   string
	irrBlockNum uint64

	notFoundABIs []string
	invalidABIs  []string
}

func (m *migrater) migrate() error {
	contracts, err := m.fetchAllContracts()
	if err != nil {
		return fmt.Errorf("fetch contracts: %w", err)
	}

	userLog.Printf("Retrieved %d contracts, fetching all tables now", len(contracts))
	for _, contract := range contracts {
		abi, err := m.fetchABI(contract)
		if err == errABINotFound {
			userLog.Warn("no ABI found for contract, this is unlikely to be possible, to investigate", zap.String("account", contract))
			m.notFoundABIs = append(m.notFoundABIs, contract)
			continue
		}

		if err == errABIInvalid {
			userLog.Debug("abi was found but was invalid, continuing", zap.String("account", contract))
			m.invalidABIs = append(m.invalidABIs, contract)
			continue
		}

		if err != nil {
			return fmt.Errorf("unable to fetch ABI for %q: %w", contract, err)
		}

		accountPath := m.accountStorage(contract)
		if err = os.MkdirAll(accountPath, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create account storage path: %w", err)
		}

		if err := m.writeABI(accountPath, abi); err != nil {
			return fmt.Errorf("unable to write ABI for %q: %w", contract, err)
		}

		if err := m.writeAllTables(contract, abi); err != nil {
			return fmt.Errorf("unable to write all tables for %q: %w", contract, err)
		}
	}

	return nil
}

func (m *migrater) fetchAllContracts() ([]string, error) {
	// FIXME: We need a maximum timeout value for the initial call so that if the client is misconfigured,
	//        the user does not wait like 15m before seeing the error.
	userLog.Debug("fetching all contracts")

	stream, err := m.fluxdb.StreamContracts(m.ctx, &pbfluxdb.StreamContractsRequest{
		BlockNum: uint64(m.irrBlockNum),
	})
	if err != nil {
		return nil, fmt.Errorf("contracts stream: %w", err)
	}

	var contracts []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return contracts, nil
		}

		if err != nil {
			return nil, fmt.Errorf("stream account: %w", err)
		}

		contracts = append(contracts, resp.Contract)
	}
}

func (m *migrater) writeAllTables(contract string, abi *eos.ABI) error {
	userLog.Debug("writing all tables", zap.String("contract", contract))
	for _, table := range abi.Tables {
		if err := m.writeTable(contract, string(table.Name)); err != nil {
			return fmt.Errorf("write table %q: %w", table, err)
		}
	}

	return nil
}

var allScopes = []string{"*"}

func (m *migrater) writeTable(contract string, table string) error {
	stream, err := m.fluxdb.GetMultiScopesTableRows(m.ctx, &pbfluxdb.GetMultiScopesTableRowsRequest{
		BlockNum:         uint64(m.irrBlockNum),
		IrreversibleOnly: true,
		Contract:         contract,
		Table:            table,
		ToJson:           true,
		KeyType:          "name",
		Scopes:           allScopes,
	})
	if err != nil {
		return fmt.Errorf("multi table scopes stream: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("stream multi table scopes: %w", err)
		}

		tablePath := m.tableStorage(contract, table, resp.Scope)
		if err = os.MkdirAll(tablePath, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create table storage path: %w", err)
		}

		if err = m.writeTableRows(tablePath, resp.Row); err != nil {
			return fmt.Errorf("write table rows: %w", err)
		}
	}
}

func (m *migrater) fetchAllAccounts() ([]string, error) {
	// FIXME: We need a maximum timeout value for the initial call so that if the client is misconfigured,
	//        the user does not wait like 15m before seeing the error.
	userLog.Debug("fetching all account")
	// FIXME: What if the user does not have any more delband things, are we screwed ...?
	//        That might be good enough or we might need to have a definitive tablet of all
	//        existing accounts within EOSIO realm.
	//
	//        The trxdb interface has a `ListAccountNames` that could be useful. Would be
	//        probably a best bet. We should compare both outputs but I think the db version
	//        will be more complete.
	stream, err := m.fluxdb.GetTableScopes(m.ctx, &pbfluxdb.GetTableScopesRequest{
		BlockNum: uint64(m.irrBlockNum),
		// IrreversibleOnly: true,
		Contract: "eosio",
		Table:    "delband",
	})
	if err != nil {
		return nil, fmt.Errorf("new accounts stream: %w", err)
	}

	var scopes []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return scopes, nil
		}

		if err != nil {
			return nil, fmt.Errorf("stream account: %w", err)
		}

		scopes = append(scopes, resp.Scope)
	}
}

var errABINotFound = errors.New("abi not found")
var errABIInvalid = errors.New("abi invalid")

func (m *migrater) fetchABI(contract string) (*eos.ABI, error) {
	resp, err := m.fluxdb.GetABI(m.ctx, &pbfluxdb.GetABIRequest{
		BlockNum: m.irrBlockNum,
		Contract: contract,
		ToJson:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch abi: %w", err)
	}

	if len(resp.RawAbi) <= 0 {
		return nil, errABINotFound
	}

	abi := new(eos.ABI)
	err = eos.UnmarshalBinary(resp.RawAbi, abi)
	if err != nil {
		userLog.Debug("unable to decode ABI", zap.String("contract", contract))
		return nil, errABIInvalid
	}

	return abi, nil
}

func (m *migrater) writeABI(storagePath string, abi *eos.ABI) error {
	file, err := os.Create(filepath.Join(storagePath, "abi.json"))
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(abi)
}

type TableRow struct {
	Key   string          `json:"key"`
	Payer string          `json:"payer"`
	Data  json.RawMessage `json:"data"`
}

func (m *migrater) writeTableRows(tablePath string, rows []*pbfluxdb.TableRowResponse) error {
	userLog.Debug("writing table", zap.String("table_path", tablePath), zap.Int("row_count", len(rows)))
	file, err := os.Create(filepath.Join(tablePath, "rows.json"))
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	lastIndex := len(rows) - 1
	file.WriteString("[")
	for i, tabletRow := range rows {
		encoder := json.NewEncoder(file)
		encoder.SetEscapeHTML(false)

		file.WriteString("\n  ")
		err := encoder.Encode(TableRow{
			Key:   tabletRow.Key,
			Payer: tabletRow.Payer,
			Data:  json.RawMessage(tabletRow.Json),
		})
		if err != nil {
			return fmt.Errorf("unable to encode row %d: %w", i, err)
		}

		if i != lastIndex {
			file.WriteString(",")
		}
	}
	file.WriteString("]")

	return nil
}

func (m *migrater) accountStorage(account string) string {
	if len(account) == 0 {
		cliErrorAndExit("Received a fully empty account, refusing to procede")
	}

	path := m.exportDir
	if len(account) <= 2 {
		path = filepath.Join(path, account)
	} else if len(account) <= 4 {
		path = filepath.Join(path, account[0:2], account)
	} else {
		path = filepath.Join(path, account[0:2], account[2:4], account)
	}

	return path
}

func (m *migrater) tableStorage(account string, table string, scope string) string {
	if len(table) == 0 {
		cliErrorAndExit("Received a fully empty table, refusing to procede")
	}

	if len(scope) == 0 {
		cliErrorAndExit("Received a fully empty scope, refusing to procede")
	}

	path := filepath.Join(m.accountStorage(account), "tables", table)

	if len(scope) <= 2 {
		path = filepath.Join(path, scope)
	} else if len(table) <= 4 {
		path = filepath.Join(path, scope[0:2], scope)
	} else {
		path = filepath.Join(path, scope[0:2], scope[2:4], scope)
	}

	return path
}
