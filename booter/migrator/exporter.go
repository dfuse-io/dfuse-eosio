package migrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	zapbox "github.com/dfuse-io/dfuse-eosio/zap-box"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
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

	contracts, err := e.fetchAllContracts()
	if err != nil {
		return fmt.Errorf("fetch contracts: %w", err)
	}

	e.logger.Printf("Retrieved %d contracts, fetching all tables now", len(contracts))
	for act, acctInfo := range accounts {
		acct, err := newAccount(e.common.dataDir, act)
		if err != nil {
			return fmt.Errorf("unable to initialize account %q storage: %w", act, err)
		}

		if err = acct.createDir(); err != nil {
			return fmt.Errorf("unable to create account storage path: %w", err)
		}

		if err := acct.writeAccount(acctInfo); err != nil {
			return fmt.Errorf("unable to write account for %q: %w", act, err)
		}

		if _, ok := contracts[act]; ok {
			code, err := e.fetchCode(act)
			if err == errCodeNotFound {
				e.logger.Printf("no code found for contract %s, will NOT migrate data of this contract", act)
				e.notFoundCodes = append(e.notFoundCodes, act)
				continue
			}

			if err != nil {
				return fmt.Errorf("unable to fetch code for %q: %w", act, err)
			}

			abi, err := e.fetchABI(act)
			if err == errABINotFound {
				e.logger.Printf("no ABI found for contract %s, will NOT migrate data of this contract", act)
				e.notFoundABIs = append(e.notFoundABIs, act)
				continue
			}

			if err == errABIInvalid {
				e.logger.Debug("abi was found but was invalid, continuing", zap.String("contract", act))
				e.invalidABIs = append(e.invalidABIs, act)
				continue
			}

			if err != nil {
				return fmt.Errorf("unable to fetch ABI for %q: %w", act, err)
			}

			if err := acct.writeCode(code); err != nil {
				return fmt.Errorf("unable to write ABI for %q: %w", act, err)
			}

			if err := acct.writeABI(abi); err != nil {
				return fmt.Errorf("unable to write ABI for %q: %w", act, err)
			}

			if err := e.writeAllTables(act, acct, abi); err != nil {
				return fmt.Errorf("unable to write all tables for %q: %w", act, err)
			}
		}
	}
	return nil
}

func (e *Exporter) fetchAllAccounts() (map[string]*accountInfo, error) {
	e.logger.Debug("fetching all accounts")

	stream, err := e.fluxdb.StreamAccounts(e.ctx, &pbfluxdb.StreamAccountsRequest{
		BlockNum: uint64(e.irrBlockNum),
	})
	if err != nil {
		return nil, fmt.Errorf("accounts stream: %w", err)
	}

	accounts := map[string]*accountInfo{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return accounts, nil
		}

		if err != nil {
			return nil, fmt.Errorf("stream account: %w", err)
		}

		linkAuths := make([]*linkAuth, len(resp.LinkedPermissions))
		for i, linkedPermission := range resp.LinkedPermissions {
			linkAuths[i] = &linkAuth{
				Permission: linkedPermission.PermissionName,
				Contract:   linkedPermission.Contract,
				Action:     linkedPermission.Action,
			}
		}

		accounts[resp.Account] = newAccountInfo(resp.Permissions, linkAuths)
	}
}

func (e *Exporter) fetchAllContracts() (map[string]bool, error) {
	// FIXME: We need a maximum timeout value for the initial call so that if the client is misconfigured,
	//        the user does not wait like 15m before seeing the error.
	e.logger.Debug("fetching all contracts")

	contracts := map[string]bool{}

	stream, err := e.fluxdb.StreamContracts(e.ctx, &pbfluxdb.StreamContractsRequest{
		BlockNum: uint64(e.irrBlockNum),
	})
	if err != nil {
		return nil, fmt.Errorf("contracts stream: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return contracts, nil
		}

		if err != nil {
			return nil, fmt.Errorf("stream account: %w", err)
		}

		contracts[resp.Contract] = true
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
			return fmt.Errorf("write table %q: %w", table.Name, err)
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

		outRow := tableRow{
			Key:   tabletRow.Key,
			Payer: tabletRow.Payer,
		}

		if tabletRow.Json != "" {
			outRow.DataJSON = json.RawMessage(tabletRow.Json)
		} else {
			outRow.DataHex = eos.HexBytes(tabletRow.Data)
		}

		file.WriteString("\n  ")
		err := encoder.Encode(outRow)
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

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}
