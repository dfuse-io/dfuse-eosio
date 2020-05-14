package sqlsync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type SQLSync struct {
	*shutter.Shutter
	db     *DB
	fluxdb fluxdb.Client

	source bstream.Source

	watchedAccounts map[eos.AccountName]*account
	blockstreamAddr string
	blocksStore     dstore.Store
}

func (t *SQLSync) getABI(contract eos.AccountName, blockNum uint32) (*eos.ABI, error) {
	resp, err := t.fluxdb.GetABI(context.Background(), blockNum, contract)
	if err != nil {
		return nil, err
	}

	return resp.ABI, nil
}

func (t *SQLSync) decodeDBOpToRow(data []byte, tableName eos.TableName, contract eos.AccountName, blocknum uint32) (json.RawMessage, error) {
	abi, err := t.getABI(contract, blocknum)
	if err != nil {
		return nil, fmt.Errorf("cannot get ABI: %w", err)
	}

	return decodeTableRow(data, tableName, abi)
}

func NewSQLSync(db *DB, fluxCli fluxdb.Client, blockstreamAddr string, blocksStore dstore.Store) *SQLSync {
	return &SQLSync{
		Shutter:         shutter.New(),
		blockstreamAddr: blockstreamAddr,
		blocksStore:     blocksStore,
		db:              db,
		fluxdb:          fluxCli,
	}
}

func (s *SQLSync) Launch(bootstrapRequired bool, startBlock bstream.BlockRef) error {
	accs, err := s.getWatchedAccounts(startBlock)
	if err != nil {
		return err
	}
	s.watchedAccounts = accs

	if bootstrapRequired {
		if err := s.bootstrapDatabase(startBlock); err != nil {
			return err
		}
	}

	if err := s.db.db.Close(); err != nil {
		return err
	}

	time.Sleep(1 * time.Hour)

	s.setupPipeline(startBlock)

	zlog.Info("launching pipeline")
	go s.source.Run()

	<-s.source.Terminated()
	if err := s.source.Err(); err != nil {
		zlog.Error("source shutdown with error", zap.Error(err))
		return err
	}
	zlog.Info("source is done")

	return nil
}

var parsableFieldTypes = []string{
	"name",
	"string",
	"symbol",
	"bool",
	"int64",
	"uint64",
	"int32",
	"uint32",
	"asset",
}

func (s *SQLSync) getWatchedAccounts(startBlock bstream.BlockRef) (map[eos.AccountName]*account, error) {
	out := make(map[eos.AccountName]*account)
	abi, err := s.getABI("simpleassets", uint32(startBlock.Num()))
	if err != nil {
		return nil, err
	}

	out["simpleassets"] = &account{
		abi:  abi,
		name: "simpleassets",
	}
	out["simpleassets"].extractTables()
	return out, nil
}

var chainToSQLTypes = map[string]string{
	"name":   "varchar(13) NOT NULL",
	"string": "varchar(1024) NOT NULL",
	"symbol": "varchar(8) NOT NULL",
	"bool":   "boolean",
	"int64":  "int NOT NULL",
	"uint64": "int unsigned NOT NULL",
	"int32":  "int NOT NULL", // make smaller
	"uint32": "int unsigned NOT NULL",
	"asset":  "varchar(64) NOT NULL",
}

func (s *SQLSync) bootstrapDatabase(startBlock bstream.BlockRef) error {
	// s.db.createTables

	// get all tables that are in watchedAccounts
	zlog.Info("bootstrapping SQL database", zap.Int("accounts", len(s.watchedAccounts)))
	for acctName, acct := range s.watchedAccounts {
		zlog.Info("bootstrapping SQL tables for account", zap.String("account", string(acctName)), zap.Int("tables", len(acct.tables)))

		for _, tbl := range acct.tables {
			stmt := "CREATE TABLE IF NOT EXISTS " + string(tbl.name) + `(
  _scope varchar(13) NOT NULL,
  _key varchar(13) NOT NULL,
`
			for _, field := range tbl.mappings {
				stmt = stmt + " " + field.ChainField + " "
				if field.KeepJSON {
					stmt = stmt + "text NOT NULL,"
				} else {
					stmt = stmt + chainToSQLTypes[field.Type] + " NOT NULL,"
				}
			}

			stmt += ` PRIMARY KEY (_scope, _key)
);`
			zlog.Debug("creating table", zap.String("stmt", stmt))
			_, err := s.db.db.ExecContext(context.Background(), stmt)
			if err != nil {
				return fmt.Errorf("create table %s for account %s: %w", tbl.name, acctName, err)
			}
		}
	}

	return s.fetchInitialSnapshots(startBlock)
}

func (s *SQLSync) fetchInitialSnapshots(startBlock bstream.BlockRef) error {
	for acctName, acct := range s.watchedAccounts {
		for _, tbl := range acct.tables {
			// get all scopes for that table in that account, and insert all rows
			stmt := "INSERT INTO " + tbl.name + "(_scope, _primkey"
			for _, field := range tbl.mappings {
				stmt = stmt + ", " + field.DBField
			}
			stmt = stmt + ") VALUES (" + strings.TrimLeft(strings.Repeat(",?", len(tbl.mappings)), ",") + ")"

			_, err := s.db.db.ExecContext(context.Background(), stmt)
			if err != nil {
				return fmt.Errorf("create table %s for account %s: %w", tbl.name, acctName, err)
			}
		}
	}

	return nil
}
