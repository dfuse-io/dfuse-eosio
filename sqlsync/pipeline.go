package sqlsync

import (
	"context"

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

func (s *SQLSync) bootstrapFromFlux(startBlock bstream.BlockRef) error {
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
		s.bootstrapFromFlux(startBlock)
	}

	s.db.db.Close()
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
