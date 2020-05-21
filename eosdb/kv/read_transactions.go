package kv

import (
	"context"

	"go.uber.org/zap"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbeosdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/eosdb/v1"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
)

func (db *DB) GetTransactionTraces(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	out, err = db.getTransactionExecutionEvents(ctx, idPrefix)
	return
}

func (db *DB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	// OPTIMIZE: Parallelize access, or do requests to get things in parallel
	for _, idPrefix := range idPrefixes {
		trxResult, err := db.GetTransactionEvents(ctx, idPrefix)
		if err != nil {
			return nil, err
		}
		out = append(out, trxResult)
	}
	return
}

func (db *DB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	// OPTIMIZE: Parallelize access, or do requests to get things in parallel
	for _, idPrefix := range idPrefixes {
		trxResult, err := db.GetTransactionTraces(ctx, idPrefix)
		if err != nil {
			return nil, err
		}
		out = append(out, trxResult)
	}
	return
}

func (db *DB) GetTransactionEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	zlog.Debug("get transaction events", zap.String("trx_prefix", idPrefix))

	evs, err := db.getTransactionAdditionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)
	zlog.Debug("added transaction addition events", zap.Int("event_count", len(out)))

	evs, err = db.getTransactionExecutionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)
	zlog.Debug("added transaction execution events", zap.Int("event_count", len(out)))

	evs, err = db.getTransactionDtrxEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)
	zlog.Debug("added transaction dtrx events", zap.Int("event_count", len(out)))

	evs, err = db.getTransactionImplicitEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)
	zlog.Debug("added transaction implicit events", zap.Int("event_count", len(out)))

	for _, o := range out {
		var typeStr string
		switch o.Event.(type) {
		case *pbcodec.TransactionEvent_Addition:
			typeStr = "addition"
		case *pbcodec.TransactionEvent_InternalAddition:
			typeStr = "internal addition"
		case *pbcodec.TransactionEvent_Execution:
			typeStr = "execution"
		case *pbcodec.TransactionEvent_DtrxScheduling:
			typeStr = "dtrx scheduling"
		case *pbcodec.TransactionEvent_DtrxCancellation:
			typeStr = "dtrx cancellation"
		default:
			typeStr = "unknown type"
		}
		zlog.Debug("trx event",
			zap.String("trx_id", o.Id),
			zap.String("blk_id", o.BlockId),
			zap.String("event_type", typeStr),
		)
	}
	return
}

func (db *DB) getTransactionAdditionEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbeosdb.TrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxsKey(it.Item().Key)

		ev := &pbcodec.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.Receipt != nil {
			ev.Event = &pbcodec.TransactionEvent_Addition{
				Addition: &pbcodec.TransactionEvent_Added{
					Receipt:     row.Receipt,
					Transaction: row.SignedTrx,
					PublicKeys:  row.PublicKeys,
				},
			}
		} else {
			ev.Event = &pbcodec.TransactionEvent_InternalAddition{
				InternalAddition: &pbcodec.TransactionEvent_AddedInternally{
					Transaction: row.SignedTrx,
				},
			}
		}

		out = append(out, ev)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	// TODO: does bigtable return `ErrNotFound` when none are found here?

	return
}

func (db *DB) getTransactionImplicitEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackImplicitTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbeosdb.ImplicitTrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackImplicitTrxsKey(it.Item().Key)

		ev := &pbcodec.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbcodec.TransactionEvent_InternalAddition{
			InternalAddition: &pbcodec.TransactionEvent_AddedInternally{
				Transaction: row.SignedTrx,
			},
		}

		out = append(out, ev)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	// TODO: does bigtable return `ErrNotFound` when none are found here?

	return
}

func (db *DB) getTransactionExecutionEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxTracesPrefix(idPrefix))
	for it.Next() {
		row := &pbeosdb.TrxTraceRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxTracesKey(it.Item().Key)

		ev := &pbcodec.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbcodec.TransactionEvent_Execution{
			Execution: &pbcodec.TransactionEvent_Executed{
				Trace:       row.TrxTrace,
				BlockHeader: row.BlockHeader,
			},
		}

		out = append(out, ev)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return
}

func (db *DB) getTransactionDtrxEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackDtrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbeosdb.DtrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackDtrxsKey(it.Item().Key)

		ev := &pbcodec.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.CreatedBy != nil {
			ev.Event = &pbcodec.TransactionEvent_DtrxScheduling{
				DtrxScheduling: &pbcodec.TransactionEvent_DtrxScheduled{
					CreatedBy:   row.CreatedBy,
					Transaction: row.SignedTrx,
				},
			}
		} else {
			ev.Event = &pbcodec.TransactionEvent_DtrxCancellation{
				DtrxCancellation: &pbcodec.TransactionEvent_DtrxCanceled{
					CanceledBy: row.CanceledBy,
				},
			}
		}

		out = append(out, ev)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	return
}
