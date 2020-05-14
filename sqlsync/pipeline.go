package sqlsync

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

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

func extractTables(abi *eos.ABI) map[eos.TableName]*Table {
	out := make(map[eos.TableName]*Table)
	for _, table := range abi.Tables {

		var mappings []Mapping
		for i := 0; i < len(table.KeyNames); i++ {
			mappings = append(mappings, Mapping{
				ChainField: table.KeyNames[i],
				DBField:    string(table.KeyNames[i]),
				KeepJSON:   !stringInFilter(table.KeyTypes[i], parsableFieldTypes),
				Type:       table.KeyTypes[i],
			})
		}
		out[table.Name] = &Table{
			mappings: mappings,
		}
	}
	return nil
}

func (s *SQLSync) getWatchedAccounts(startBlock bstream.BlockRef) (map[eos.AccountName]*account, error) {
	out := make(map[eos.AccountName]*account)
	abi, err := s.getABI("simpleassets", uint32(startBlock.Num()))
	if err != nil {
		return nil, err
	}

	out["simpleassets"] = &account{
		abi:    abi,
		tables: extractTables(abi),
	}
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

func (s *SQLSync) bootstrapFromFlux(startBlock bstream.BlockRef) error {
	// s.db.createTables

	// get all tables that are in watchedAccounts
	zlog.Info("bootstrapping SQL database", zap.Int("accounts", len(s.watchedAccounts)))
	for acctName, acct := range s.watchedAccounts {

		for tblName, tbl := range acct.tables {
			stmt := "CREATE TABLE IF NOT EXISTS " + string(tblName) + `(
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
			zlog.Info("creating table", zap.String("stmt", stmt))
			_, err := s.db.db.ExecContext(context.Background(), stmt)
			if err != nil {
				return fmt.Errorf("create table %s for account %s: %w", tblName, acctName, err)
			}
		}
	}

	// get snapshots

	//s.fluxdb.GetTable()
	return nil
}

func (s *SQLSync) Launch(bootstrapRequired bool, startBlock bstream.BlockRef) error {
	accs, err := s.getWatchedAccounts(startBlock)
	if err != nil {
		return err
	}
	s.watchedAccounts = accs

	if bootstrapRequired {
		if err := s.bootstrapFromFlux(startBlock); err != nil {
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

func (s *SQLSync) setupPipeline(startBlock bstream.BlockRef) {

	sf := bstream.SourceFromRefFactory(func(startBlockRef bstream.BlockRef, h bstream.Handler) bstream.Source {
		if startBlockRef.ID() == "" {
			startBlockRef = startBlock
		}

		archivedBlockSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			src := bstream.NewFileSource(s.blocksStore, startBlockRef.Num(), 1, nil, subHandler)
			return src
		})

		zlog.Info("new live joining source", zap.Uint64("start_block_num", startBlockRef.Num()))
		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				context.Background(),
				s.blockstreamAddr,
				200,
				subHandler,
			)
		})

		options := []bstream.JoiningSourceOption{}
		if startBlockRef.ID() != "" {
			options = append(options, bstream.JoiningSourceTargetBlockID(startBlockRef.ID()))
		}

		js := bstream.NewJoiningSource(
			archivedBlockSourceFactory,
			liveSourceFactory,
			h,
			options...)
		return js
	})

	forkOptions := []forkable.Option{
		forkable.WithFilters(forkable.StepIrreversible), // FIXME eventually keep last saved LIB as well as last saved head, so we can start from LIB but gate at the last processed head block and manage undos
	}
	if startBlock.ID() != "" {
		forkOptions = append(forkOptions, forkable.WithExclusiveLIB(startBlock))
	}
	forkableHandler := forkable.New(s, forkOptions...)

	s.source = bstream.NewEternalSource(sf, forkableHandler)

	s.OnTerminating(func(e error) {
		s.source.Shutdown(e)
	})
}
