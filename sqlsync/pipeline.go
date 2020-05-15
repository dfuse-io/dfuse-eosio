package sqlsync

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/tidwall/gjson"
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

func (s *SQLSync) ProcessBlock(block *bstream.Block, obj interface{}) (err error) {
	// forkable setup will only yield irreversible blocks
	blk := block.ToNative().(*pbcodec.Block)

	if (blk.Number % 50) == 0 {
		zlog.Info("sqlsync processing block", zap.String("block_id", block.ID()), zap.Uint64("blocker_number", block.Number))
	}

	ctx := context.Background()
	_, err = s.db.db.ExecContext(ctx, BEGIN_TRANSACTION)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_, newerr := s.db.db.ExecContext(ctx, "ROLLBACK")
			if newerr != nil {
				err = fmt.Errorf("rollback failed (%s) after error: %w", newerr, err)
			}
		} else {
			_, err = s.db.db.ExecContext(ctx, "COMMIT")
			if err != nil {
				err = fmt.Errorf("commit failed: %w", err)
			}
		}
	}()

	// FIXME: Check for ABI changes, update a local cache, like
	// `abicodec` or ConsoleReader's gseq-based cache.

	for _, trx := range blk.TransactionTraces {
		//zlogger := zlog.With(zap.Uint64("blk_id", block.Num()), zap.String("trx_id", trx.Id))

		for _, dbop := range trx.DbOps {
			watch := s.watchedAccounts[dbop.Code]
			if watch == nil {
				continue
			}
			tableDef := watch.tables[dbop.TableName]

			//zlog.Debug("processing dbop", zap.String("contract", dbop.Code), zap.String("table", dbop.TableName), zap.String("scope", dbop.Scope), zap.String("primary_key", dbop.PrimaryKey))

			switch dbop.Operation {
			case pbcodec.DBOp_OPERATION_INSERT:
				jsonData, err := watch.abi.DecodeTableRow(eos.TableName(dbop.TableName), dbop.NewData)
				if err != nil {
					zlog.Error("decoding row on insert", zap.Error(err))
					continue
				}

				stmt, values, err := tableDef.insertStatement(s.db, dbop.Scope, dbop.PrimaryKey, dbop.NewPayer, gjson.ParseBytes(jsonData))
				if err != nil {
					return fmt.Errorf("building insert statement: %w", err)
				}
				_, err = s.db.db.ExecContext(ctx, stmt, values...)
				if err != nil {
					return fmt.Errorf("executing statement %q: %w", stmt, err)
				}

			case pbcodec.DBOp_OPERATION_UPDATE:
				// TODO: check if it's only a payer change, don't
				// update the full row, compare OldData and NewData,
				// and OldPayer, and NewPayer. Update accordingly.

				// HERE we assume there's always a change in content, we update the full row.
				jsonData, err := watch.abi.DecodeTableRow(eos.TableName(dbop.TableName), dbop.NewData)
				if err != nil {
					zlog.Error("decoding row on update", zap.Error(err))
					// FIXME: If we wanted to update, we need to check
					// if the row was there previously, because it
					// would mean FAKE data still being shown there.
					// Is it better to DELETE the row in that case?
					// There's no good answer.  If it was present, we
					// need to hard-fail.  If it was not present, then
					// maybe it was never valid according to this ABI,
					// so we will continue to ignore it.
					continue
				}

				stmt, values, err := tableDef.updateStatement(s.db, dbop.Scope, dbop.PrimaryKey, dbop.NewPayer, gjson.ParseBytes(jsonData))
				if err != nil {
					return fmt.Errorf("building update statement: %w", err)
				}
				_, err = s.db.db.ExecContext(ctx, stmt, values...)
				if err != nil {
					return fmt.Errorf("executing statement %q: %w", stmt, err)
				}

			case pbcodec.DBOp_OPERATION_REMOVE:
				stmt := tableDef.deleteStatement()
				_, err = s.db.db.ExecContext(ctx, stmt, dbop.Scope, dbop.PrimaryKey)
				if err != nil {
					return fmt.Errorf("delete statement: %w", err)
				}
			}
		}
	}

	// Make this DB-agnostic.. this is `postgres` specific.
	_, err = s.db.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO sqlsync_markers (table_prefix, block_id, block_num) VALUES ($1, $2, $3) ON CONFLICT (table_prefix) DO UPDATE SET block_id = $2, block_num = $3`), s.tablePrefix, blk.ID(), blk.Num())
	if err != nil {
		return fmt.Errorf("flushing markers: %w", err)
	}

	return nil
}
