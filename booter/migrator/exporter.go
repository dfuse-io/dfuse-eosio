package migrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/eoscanada/eos-go"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	zapbox "github.com/dfuse-io/dfuse-eosio/zap-box"
	"go.uber.org/zap"
)

var (
	errCodeNotFound = errors.New("code not found")
	errABINotFound  = errors.New("abi not found")
	errABIInvalid   = errors.New("abi invalid")
)

type Exporter struct {
	common

	ctx         context.Context
	fluxdb      pbfluxdb.StateClient
	irrBlockNum uint64
	logger      *zapbox.CLILogger

	notFoundCodes []string
	notFoundABIs  []string
	invalidABIs   []string
}

func NewExporter(ctx context.Context, fluxdb pbfluxdb.StateClient, exportDir string, irrBlockNum uint64, logger *zapbox.CLILogger) *Exporter {
	return &Exporter{
		ctx:         ctx,
		fluxdb:      fluxdb,
		irrBlockNum: irrBlockNum,
		common:      common{dataDir: exportDir},
		logger:      logger,
	}
}

func (e *Exporter) Export() error {
	accounts, err := e.fetchAllAccounts()
	if err != nil {
		return fmt.Errorf("fetch accounts: %w", err)
	}

	if err = e.common.createDataDir(); err != nil {
		return fmt.Errorf("unable to create export directory: %w", err)
	}

	if err = writeJSONFile(e.common.accountListPath(), accounts); err != nil {
		return fmt.Errorf("unable to write account list: %w", err)
	}

	contracts, err := e.fetchAllContracts()
	if err != nil {
		return fmt.Errorf("fetch contracts: %w", err)
	}

	e.logger.Printf("Retrieved %d contracts, fetching all tables now", len(contracts))
	for _, contract := range contracts {
		code, err := e.fetchCode(contract)
		if err == errCodeNotFound {
			e.logger.Printf("no code found for contract %s, will NOT migrate data of this contract", contract)
			e.notFoundCodes = append(e.notFoundCodes, contract)
			continue
		}

		if err != nil {
			return fmt.Errorf("unable to fetch code for %q: %w", contract, err)
		}

		abi, err := e.fetchABI(contract)
		if err == errABINotFound {
			e.logger.Printf("no ABI found for contract %s, will NOT migrate data of this contract", contract)
			e.notFoundABIs = append(e.notFoundABIs, contract)
			continue
		}

		if err == errABIInvalid {
			e.logger.Debug("abi was found but was invalid, continuing", zap.String("contract", contract))
			e.invalidABIs = append(e.invalidABIs, contract)
			continue
		}

		if err != nil {
			return fmt.Errorf("unable to fetch ABI for %q: %w", contract, err)
		}

		acct, err := newAccount(e.common.dataDir, contract)
		if err != nil {
			return fmt.Errorf("unable to initialize account storage: %w", err)
		}

		if err = acct.createDir(); err != nil {
			return fmt.Errorf("unable to create account storage path: %w", err)
		}

		if err := acct.writeAccount(); err != nil {
			return fmt.Errorf("unable to write account for %q: %w", contract, err)
		}

		if err := acct.writeCode(code); err != nil {
			return fmt.Errorf("unable to write ABI for %q: %w", contract, err)
		}

		if err := acct.writeABI(abi); err != nil {
			return fmt.Errorf("unable to write ABI for %q: %w", contract, err)
		}

		if err := e.writeAllTables(contract, acct, abi); err != nil {
			return fmt.Errorf("unable to write all tables for %q: %w", contract, err)
		}
	}

	return nil
}

func (e *Exporter) fetchAllAccounts() ([]string, error) {
	e.logger.Debug("fetching all accounts")

	stream, err := e.fluxdb.StreamAccounts(e.ctx, &pbfluxdb.StreamAccountsRequest{
		BlockNum: uint64(e.irrBlockNum),
	})
	if err != nil {
		return nil, fmt.Errorf("accounts stream: %w", err)
	}

	var accounts []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return accounts, nil
		}

		if err != nil {
			return nil, fmt.Errorf("stream account: %w", err)
		}

		accounts = append(accounts, resp.Account)
	}
}

func (e *Exporter) fetchAllContracts() ([]string, error) {
	// FIXME: We need a maximum timeout value for the initial call so that if the client is misconfigured,
	//        the user does not wait like 15m before seeing the error.
	e.logger.Debug("fetching all contracts")

	stream, err := e.fluxdb.StreamContracts(e.ctx, &pbfluxdb.StreamContractsRequest{
		BlockNum: uint64(e.irrBlockNum),
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

func (e *Exporter) fetchCode(contract string) ([]byte, error) {
	resp, err := e.fluxdb.GetCode(e.ctx, &pbfluxdb.GetCodeRequest{
		BlockNum: e.irrBlockNum,
		Contract: contract,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch code: %w", err)
	}

	if len(resp.RawCode) <= 0 {
		return nil, errCodeNotFound
	}

	return resp.RawCode, nil
}

func (e *Exporter) fetchABI(contract string) (*eos.ABI, error) {
	resp, err := e.fluxdb.GetABI(e.ctx, &pbfluxdb.GetABIRequest{
		BlockNum: e.irrBlockNum,
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
		e.logger.Debug("unable to decode ABI", zap.String("contract", contract))
		return nil, errABIInvalid
	}

	return abi, nil
}

func (e *Exporter) writeAllTables(contract string, acct *Account, abi *eos.ABI) error {
	e.logger.Debug("writing all tables", zap.String("contract", contract))
	for _, table := range abi.Tables {
		if err := e.writeTable(contract, acct, string(table.Name)); err != nil {
			return fmt.Errorf("write table %q: %w", table, err)
		}
	}

	return nil
}

var allScopes = []string{"*"}

func (e *Exporter) writeTable(contract string, acct *Account, table string) error {
	stream, err := e.fluxdb.GetMultiScopesTableRows(e.ctx, &pbfluxdb.GetMultiScopesTableRowsRequest{
		BlockNum:         uint64(e.irrBlockNum),
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

	tablePath, err := acct.TablePath(table)
	if err != nil {
		return fmt.Errorf("unable to determine table path: %w", err)
	}

	seenScopes := []string{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			if err = os.MkdirAll(string(tablePath), os.ModePerm); err != nil {
				return fmt.Errorf("unable to create table scope storage path: %w", err)
			}

			if err = writeJSONFile(acct.ScopeListPath(tablePath), seenScopes); err != nil {
				return fmt.Errorf("unable to write scope list: %w", err)
			}
			return nil
		}

		if err != nil {
			return fmt.Errorf("stream multi table scopes: %w", err)
		}

		scopePath, err := acct.ScopePath(tablePath, resp.Scope)
		if err != nil {
			return fmt.Errorf("unable to determine accout %q table %q scope %q path: %w", contract, table, resp.Scope, err)
		}
		seenScopes = append(seenScopes, resp.Scope)

		if err = os.MkdirAll(string(scopePath), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create table scope storage path: %w", err)
		}

		if err = e.writeTableRows(acct.RowsPath(scopePath), resp.Row); err != nil {
			return fmt.Errorf("write table scope rows: %w", err)
		}
	}
}

func (e *Exporter) writeTableRows(rowsPath string, rows []*pbfluxdb.TableRowResponse) error {
	e.logger.Debug("writing table", zap.String("table_scope_path", string(rowsPath)), zap.Int("row_count", len(rows)))
	file, err := os.Create(rowsPath)
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
		err := encoder.Encode(tableRow{
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
