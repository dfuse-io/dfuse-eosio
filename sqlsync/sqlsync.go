package sqlsync

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type SQLSync struct {
	*shutter.Shutter
	db     *DB
	fluxdb fluxdb.Client

	tablePrefix    string
	truncateScopes int

	source bstream.Source

	watchedAccounts map[string]*account // account name -> account struct
	blockstreamAddr string
	blocksStore     dstore.Store
}

func NewSQLSync(db *DB, fluxCli fluxdb.Client, blockstreamAddr string, blocksStore dstore.Store, truncateScopes int, tablePrefix string) *SQLSync {
	return &SQLSync{
		Shutter:         shutter.New(),
		truncateScopes:  truncateScopes,
		tablePrefix:     tablePrefix,
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

	zlog.Info("setting up pipeline")

	s.setupPipeline(startBlock)

	zlog.Info("launching pipeline")
	go s.source.Run()

	<-s.source.Terminated()
	if err := s.source.Err(); err != nil {
		zlog.Error("source shutdown with error", zap.Error(err))
		return err
	}
	zlog.Info("source is done")

	if err := s.db.db.Close(); err != nil {
		return err
	}

	return nil
}

func (s *SQLSync) getWatchedAccounts(startBlock bstream.BlockRef) (map[string]*account, error) {

	// TODO: use the command-line flags for what to watch.

	out := make(map[string]*account)
	abi, err := s.getABI("simpleassets", uint32(startBlock.Num()))
	if err != nil {
		return nil, err
	}

	out["simpleassets"] = &account{
		abi:  abi,
		name: "simpleassets",
	}
	out["simpleassets"].extractTables(s.tablePrefix)
	return out, nil
}

func (t *SQLSync) getABI(contract eos.AccountName, blockNum uint32) (*eos.ABI, error) {
	resp, err := t.fluxdb.GetABI(context.Background(), blockNum, contract)
	if err != nil {
		return nil, err
	}

	return resp.ABI, nil
}

func (s *SQLSync) bootstrapDatabase(startBlock bstream.BlockRef) error {
	_, err := s.db.db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS sqlsync_markers (
  table_prefix char(64) NOT NULL,
  block_id char(64) NOT NULL,
  block_num `+chainToSQLTypes["uint64"]+` NOT NULL,

  PRIMARY KEY (table_prefix)
)`)
	if err != nil {
		return fmt.Errorf("creating sqlsync_markers table: %w", err)
	}

	// get all tables that are in watchedAccounts
	zlog.Info("bootstrapping SQL database", zap.Int("accounts", len(s.watchedAccounts)))
	for acctName, acct := range s.watchedAccounts {
		zlog.Info("bootstrapping SQL tables for account", zap.String("account", string(acctName)), zap.Int("tables", len(acct.tables)))

		for _, tbl := range acct.tables {
			stmt := tbl.createTableStatement()

			zlog.Debug("creating table", zap.String("stmt", stmt))
			_, err := s.db.db.ExecContext(context.Background(), stmt)
			if err != nil {
				return fmt.Errorf("create table %s for account %s: %w", tbl.dbName, acctName, err)
			}
		}
	}

	return s.fetchInitialSnapshots(startBlock)
}

func (s *SQLSync) fetchInitialSnapshots(startBlock bstream.BlockRef) error {
	ctx := context.Background()

	atBlock := uint32(startBlock.Num())

	for acctName, acct := range s.watchedAccounts {
		for eosTableName, tbl := range acct.tables {
			// get all scopes for that table in that account, and insert all rows
			scopesResp, err := s.fluxdb.GetTableScopes(ctx, atBlock, &fluxdb.GetTableScopesRequest{
				Account: eos.AccountName(acctName),
				Table:   eos.TableName(eosTableName),
			})
			if err != nil {
				// FIXME: wut? we have it listed in the ABI flux
				// provided to us, and flux says it's not in the list
				// of tables?? Is it because there's no data actually?
				// If that,s the case, let's use the error code to
				// know we don't store anything.
				zlog.Error("get table scopes, skipping", zap.Error(err))
				//return fmt.Errorf("get table scopes: %w", err)
				continue
			}

			scopes := scopesResp.Scopes
			if s.truncateScopes != 0 && len(scopes) > s.truncateScopes {
				zlog.Info("truncating the number of scopes we retrieve", zap.String("table", tbl.dbName), zap.Int("max_scopes", s.truncateScopes))
				scopes = scopes[:s.truncateScopes]
			}

			chunkSize := 1000
			for i := 0; i < len(scopes); i += chunkSize {
				scopesChunk := scopes[i : i+min(len(scopes)-i, chunkSize)]
				resp, err := s.fluxdb.GetTablesMultiScopes(ctx, atBlock, &fluxdb.GetTablesMultiScopesRequest{
					Account: eos.AccountName(acctName),
					KeyType: "name",
					Table:   eos.TableName(eosTableName),
					Scopes:  scopesChunk,
					JSON:    true, // TODO: this could/should be done locally, and ideally sped up like crazy, decoding directly to the types useful to SQL. eos-go/abidecoder needs to be boosted for that to happen
				})
				if err != nil {
					return fmt.Errorf("get tables multi scopes: %w", err)
				}

				for _, tblResp := range resp.Tables {
					scopeResp := tblResp.Scope
					result := gjson.ParseBytes(tblResp.Rows)
					var innerErr error
					result.ForEach(func(k, v gjson.Result) bool {
						if !v.Exists() {
							return false
						}

						// fmt.Println("ONE ROW:", v.Raw)

						stmt, values, err := tbl.insertStatement(
							s.db,
							scopeResp,
							v.Get("key").String(),
							v.Get("payer").String(),
							v.Get("json"),
						)
						if err != nil {
							innerErr = err
							return false
						}

						_, err = s.db.db.ExecContext(ctx, stmt, values...)
						if err != nil {
							innerErr = fmt.Errorf("insert into %s: %w", tbl.dbName, err)
							return false
						}

						return true
					})
					if err := innerErr; err != nil {
						return err
					}
				}
			}

		}
	}

	zlog.Info("bootstrap done")

	return nil
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
