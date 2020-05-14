package sqlsync

import (
	"context"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

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

func (t *SQLSync) ProcessBlock(block *bstream.Block, obj interface{}) error {
	// forkable setup will only yield irreversible blocks
	blk := block.ToNative().(*pbcodec.Block)

	if (blk.Number % 120) == 0 {
		zlog.Info("process blk 1/120", zap.String("block_id", block.ID()), zap.Uint64("blocker_number", block.Number))
	}

	for _, trx := range blk.TransactionTraces {
		zlogger := zlog.With(zap.Uint64("blk_id", block.Num()), zap.String("trx_id", trx.Id))

		for _, dbop := range trx.DbOps {
			if !shouldProcessDbop(dbop) {
				continue
			}
			zlog.Debug("processing dbop", zap.String("contract", dbop.Code), zap.String("table", dbop.TableName), zap.String("scope", dbop.Scope), zap.String("primary_key", dbop.PrimaryKey))

			rowData := dbop.NewData
			if rowData == nil {
				zlog.Info("using db row old data")
				rowData = dbop.OldData
			}
			contract := eos.AccountName("whatever")
			row, err := t.decodeDBOpToRow(rowData, eos.TableName(dbop.TableName), contract, uint32(block.Number))
			if err != nil {
				zlogger.Error("cannot decode table row",
					zap.String("contract", string(contract)),
					zap.String("table_name", dbop.TableName),
					zap.String("transaction_id", trx.Id),
					zap.Error(err))
				continue
			}
			_ = row

			switch dbop.TableName {
			}
		}
	}
	return nil
}

func shouldProcessDbop(dbop *pbcodec.DBOp) bool {
	//	if dbop.TableName == string(...) {
	//		return true
	//	}
	//	return false
	return false
}

func shouldProcessAction(actionTrace *pbcodec.ActionTrace) bool {
	if actionTrace.Action.Name == "close" {
		return true
	}
	return false
}
