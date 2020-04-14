package kv

import (
	"context"

	pbrows "github.com/dfuse-io/dfuse-eosio/eosdb/kv/pb"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/kvdb/store"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/eoscanada/eos-go"
)

func (db *DB) GetTransactionTraces(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	out, err = db.getTransactionExecutionEvents(ctx, idPrefix)
	return
}

func (db *DB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) (out [][]*pbdeos.TransactionEvent, err error) {
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

func (db *DB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbdeos.TransactionEvent, err error) {
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

func (db *DB) GetTransactionEvents(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	evs, err := db.getTransactionAdditionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)

	evs, err = db.getTransactionExecutionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)

	evs, err = db.getTransactionDtrxEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)

	evs, err = db.getTransactionImplicitEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	out = append(out, evs...)

	return
}

func (db *DB) getTransactionAdditionEvents(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbrows.TrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxsKey(it.Item().Key)

		ev := &pbdeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.Receipt != nil {
			ev.Event = &pbdeos.TransactionEvent_Addition{
				Addition: &pbdeos.TransactionEvent_Added{
					Receipt:     row.Receipt,
					Transaction: row.SignedTrx,
					PublicKeys:  row.PublicKeys,
				},
			}
		} else {
			ev.Event = &pbdeos.TransactionEvent_InternalAddition{
				InternalAddition: &pbdeos.TransactionEvent_AddedInternally{
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

func (db *DB) getTransactionImplicitEvents(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackImplicitTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbrows.ImplicitTrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackImplicitTrxsKey(it.Item().Key)

		ev := &pbdeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbdeos.TransactionEvent_InternalAddition{
			InternalAddition: &pbdeos.TransactionEvent_AddedInternally{
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

func (db *DB) getTransactionExecutionEvents(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxTracesPrefix(idPrefix))
	for it.Next() {
		row := &pbrows.TrxTraceRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxTracesKey(it.Item().Key)

		ev := &pbdeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbdeos.TransactionEvent_Execution{
			Execution: &pbdeos.TransactionEvent_Executed{
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

func (db *DB) getTransactionDtrxEvents(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackDtrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbrows.DtrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackDtrxsKey(it.Item().Key)

		ev := &pbdeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.CreatedBy != nil {
			ev.Event = &pbdeos.TransactionEvent_DtrxScheduling{
				DtrxScheduling: &pbdeos.TransactionEvent_DtrxScheduled{
					CreatedBy:   row.CreatedBy,
					Transaction: row.SignedTrx,
				},
			}
		} else {
			ev.Event = &pbdeos.TransactionEvent_DtrxCancellation{
				DtrxCancellation: &pbdeos.TransactionEvent_DtrxCanceled{
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
