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
	"github.com/tidwall/gjson"
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

func (s *SQLSync) bootstrapDatabase(startBlock bstream.BlockRef) error {
	// s.db.createTables

	// get all tables that are in watchedAccounts
	zlog.Info("bootstrapping SQL database", zap.Int("accounts", len(s.watchedAccounts)))
	for acctName, acct := range s.watchedAccounts {
		zlog.Info("bootstrapping SQL tables for account", zap.String("account", string(acctName)), zap.Int("tables", len(acct.tables)))

		for _, tbl := range acct.tables {
			stmt := "CREATE TABLE " + tbl.dbName + `(
  _scope varchar(13) NOT NULL,
  _key varchar(13) NOT NULL,
  _payer varchar(13) NOT NULL,
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
			stmt := "INSERT INTO " + tbl.dbName + "(_scope, _key, _payer"
			for _, field := range tbl.mappings {
				stmt = stmt + ", " + field.DBField
			}
			stmt = stmt + ") VALUES (?,?,?" + strings.Repeat(",?", len(tbl.mappings)) + ")"

			scopesResp, err := s.fluxdb.GetTableScopes(ctx, atBlock, &fluxdb.GetTableScopesRequest{
				Account: acctName,
				Table:   eosTableName,
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
			if len(scopes) > 1500 {
				zlog.Info("truncating table to 1500 rows", zap.String("table", tbl.dbName))
				scopes = scopes[:1500]
			}

			chunkSize := 1000
			for i := 0; i < len(scopes); i += chunkSize {
				scopesChunk := scopes[i : i+min(len(scopes)-i, chunkSize)]
				fmt.Println("Getting scopes", eosTableName, scopesChunk)
				resp, err := s.fluxdb.GetTablesMultiScopes(ctx, atBlock, &fluxdb.GetTablesMultiScopesRequest{
					Account: acctName,
					KeyType: "name",
					Table:   eosTableName,
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

						values := []interface{}{
							scopeResp,
							v.Get("key").String(),
							v.Get("payer").String(),
						}

						decoded := v.Get("json")
						for _, m := range tbl.mappings {
							val := decoded.Get(m.ChainField)
							if m.KeepJSON {
								values = append(values, val.Raw)
							} else {
								convertedValue, err := mapToSQLType(val, m.Type)
								if err != nil {
									innerErr = fmt.Errorf("converting raw JSON %q to %s: %w", val.Raw, m.Type, err)
									return false
								}
								values = append(values, convertedValue)
							}
						}
						if err := innerErr; err != nil {
							return false
						}

						//fmt.Printf("Inserting into %w, %q: %v\n", tbl.dbName, stmt, values)
						_, err := s.db.db.ExecContext(ctx, stmt, values...)
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
