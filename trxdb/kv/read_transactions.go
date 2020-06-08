package kv

import (
	"bytes"
	"context"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
)

type TrxEventType int

const (
	TrxAdditionEvent TrxEventType = iota + 1
	TrxExecutionEvent
	ImplicitTrxEvent
	DtrxEvent
)

func (db *DB) GetTransactionTraces(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	out, err = db.getTransactionEvents(ctx, idPrefix, TrxExecutionEvent)
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, out)
	return
}

func (db *DB) GetTransactionEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	out, err = db.getTransactionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, out)
	return
}

func (db *DB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	// OPTIMIZE: Parallelize access, or do requests to get things in parallel
	for _, idPrefix := range idPrefixes {
		trxResult, err := db.getTransactionEvents(ctx, idPrefix)
		if err != nil {
			return nil, err
		}
		out = append(out, trxResult)
	}
	err = db.fillIrreversibilityDataArray(ctx, out)
	return
}

func (db *DB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	// OPTIMIZE: Parallelize access, or do requests to get things in parallel
	for _, idPrefix := range idPrefixes {
		trxResult, err := db.getTransactionEvents(ctx, idPrefix, TrxExecutionEvent)
		if err != nil {
			return nil, err
		}
		out = append(out, trxResult)
	}
	err = db.fillIrreversibilityDataArray(ctx, out)
	return
}

func (db *DB) fillIrreversibilityDataArray(ctx context.Context, eventsArray [][]*pbcodec.TransactionEvent) error {
	var flatEvs []*pbcodec.TransactionEvent
	for _, evs := range eventsArray {
		flatEvs = append(flatEvs, evs...)
	}
	return db.fillIrreversibilityData(ctx, flatEvs)
}

func (db *DB) fillIrreversibilityData(ctx context.Context, events []*pbcodec.TransactionEvent) error {
	blockIDs := make(map[string]bool)
	for _, ev := range events {
		blockIDs[ev.BlockId] = false
	}
	var prefixes [][]byte
	for id := range blockIDs {
		prefixes = append(prefixes, Keys.PackIrrBlocksKey(id))
	}

	it := db.store.BatchGet(ctx, prefixes)
	for it.Next() {
		blockIDs[Keys.UnpackIrrBlocksKey(it.Item().Key)] = true
	}
	if it.Err() != nil && it.Err() != store.ErrNotFound {
		return it.Err()
	}

	for _, ev := range events {
		ev.Irreversible = blockIDs[ev.BlockId]
	}

	return nil
}

func (db *DB) getTransactionEvents(ctx context.Context, idPrefix string, eventTypes ...TrxEventType) (out []*pbcodec.TransactionEvent, err error) {
	var keys [][]byte
	if len(eventTypes) == 0 { //default behavior is get all events
		eventTypes = []TrxEventType{TrxAdditionEvent, TrxExecutionEvent, ImplicitTrxEvent, DtrxEvent}
	}
	for _, t := range eventTypes {
		switch t {
		case TrxAdditionEvent:
			keys = append(keys, Keys.PackTrxsPrefix(idPrefix))
		case TrxExecutionEvent:
			keys = append(keys, Keys.PackTrxTracesPrefix(idPrefix))
		case ImplicitTrxEvent:
			keys = append(keys, Keys.PackImplicitTrxsPrefix(idPrefix))
		case DtrxEvent:
			keys = append(keys, Keys.PackDtrxsPrefix(idPrefix))
		default:
			panic("invalid trx event")
		}
	}

	it := db.store.BatchPrefix(ctx, keys, store.Unlimited)
	for it.Next() {
		switch {
		// Implicit Transaction Addition
		case bytes.HasPrefix(it.Item().Key, Keys.StartOfImplicitTrxsTable()):
			row := &pbtrxdb.ImplicitTrxRow{}
			db.dec.MustInto(it.Item().Value, row)

			trxID, blockID := Keys.UnpackImplicitTrxsKey(it.Item().Key)
			ev := &pbcodec.TransactionEvent{
				Id:       trxID,
				BlockId:  blockID,
				BlockNum: eos.BlockNum(blockID),
			}
			ev.Event = &pbcodec.TransactionEvent_InternalAddition{
				InternalAddition: &pbcodec.TransactionEvent_AddedInternally{
					Transaction: row.SignedTrx,
				},
			}
			out = append(out, ev)

		// Transaction Addition
		case bytes.HasPrefix(it.Item().Key, Keys.StartOfTrxsTable()):
			row := &pbtrxdb.TrxRow{}
			db.dec.MustInto(it.Item().Value, row)

			trxID, blockID := Keys.UnpackTrxsKey(it.Item().Key)

			ev := &pbcodec.TransactionEvent{
				Id:       trxID,
				BlockId:  blockID,
				BlockNum: eos.BlockNum(blockID),
			}

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

		// Transaction Execution
		case bytes.HasPrefix(it.Item().Key, Keys.StartOfTrxTracesTable()):
			row := &pbtrxdb.TrxTraceRow{}
			db.dec.MustInto(it.Item().Value, row)

			trxID, blockID := Keys.UnpackTrxTracesKey(it.Item().Key)

			ev := &pbcodec.TransactionEvent{
				Id:       trxID,
				BlockId:  blockID,
				BlockNum: eos.BlockNum(blockID),
			}

			ev.Event = &pbcodec.TransactionEvent_Execution{
				Execution: &pbcodec.TransactionEvent_Executed{
					Trace:       row.TrxTrace,
					BlockHeader: row.BlockHeader,
				},
			}

			out = append(out, ev)

		// Deferred Trx
		case bytes.HasPrefix(it.Item().Key, Keys.StartOfDtrxsTable()):
			row := &pbtrxdb.DtrxRow{}
			db.dec.MustInto(it.Item().Value, row)

			trxID, blockID := Keys.UnpackDtrxsKey(it.Item().Key)

			ev := &pbcodec.TransactionEvent{
				Id:       trxID,
				BlockId:  blockID,
				BlockNum: eos.BlockNum(blockID),
			}

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
	}

	// TODO: does bigtable return `ErrNotFound` when none are found here?
	if err := it.Err(); err != nil {
		return nil, err
	}

	return
}
