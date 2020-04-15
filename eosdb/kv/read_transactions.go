package kv

import (
	"context"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	pbkv "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosdb/kv/v1"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
)

func (db *DB) GetTransactionTraces(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	out, err = db.getTransactionExecutionEvents(ctx, idPrefix)
	return
}

func (db *DB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) (out [][]*pbeos.TransactionEvent, err error) {
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

func (db *DB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbeos.TransactionEvent, err error) {
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

func (db *DB) GetTransactionEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
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

func (db *DB) getTransactionAdditionEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbkv.TrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxsKey(it.Item().Key)

		ev := &pbeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.Receipt != nil {
			ev.Event = &pbeos.TransactionEvent_Addition{
				Addition: &pbeos.TransactionEvent_Added{
					Receipt:     row.Receipt,
					Transaction: row.SignedTrx,
					PublicKeys:  row.PublicKeys,
				},
			}
		} else {
			ev.Event = &pbeos.TransactionEvent_InternalAddition{
				InternalAddition: &pbeos.TransactionEvent_AddedInternally{
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

func (db *DB) getTransactionImplicitEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackImplicitTrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbkv.ImplicitTrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackImplicitTrxsKey(it.Item().Key)

		ev := &pbeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbeos.TransactionEvent_InternalAddition{
			InternalAddition: &pbeos.TransactionEvent_AddedInternally{
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

func (db *DB) getTransactionExecutionEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackTrxTracesPrefix(idPrefix))
	for it.Next() {
		row := &pbkv.TrxTraceRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackTrxTracesKey(it.Item().Key)

		ev := &pbeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		ev.Event = &pbeos.TransactionEvent_Execution{
			Execution: &pbeos.TransactionEvent_Executed{
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

func (db *DB) getTransactionDtrxEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	it := db.store.Prefix(ctx, Keys.PackDtrxsPrefix(idPrefix))
	for it.Next() {
		row := &pbkv.DtrxRow{}
		db.dec.MustInto(it.Item().Value, row)

		trxID, blockID := Keys.UnpackDtrxsKey(it.Item().Key)

		ev := &pbeos.TransactionEvent{
			Id:       trxID,
			BlockId:  blockID,
			BlockNum: eos.BlockNum(blockID),
		}
		_, err := db.store.Get(ctx, Keys.PackIrrBlocksKey(blockID))
		ev.Irreversible = err != store.ErrNotFound

		if row.CreatedBy != nil {
			ev.Event = &pbeos.TransactionEvent_DtrxScheduling{
				DtrxScheduling: &pbeos.TransactionEvent_DtrxScheduled{
					CreatedBy:   row.CreatedBy,
					Transaction: row.SignedTrx,
				},
			}
		} else {
			ev.Event = &pbeos.TransactionEvent_DtrxCancellation{
				DtrxCancellation: &pbeos.TransactionEvent_DtrxCanceled{
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
