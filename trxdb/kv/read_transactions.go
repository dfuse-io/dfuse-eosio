package kv

import (
	"bytes"
	"context"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/codec"
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
	out, err = db.getTransactionEvents(ctx, []string{idPrefix}, TrxExecutionEvent)
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, out)
	return
}

func (db *DB) GetTransactionEvents(ctx context.Context, idPrefix string) (out []*pbcodec.TransactionEvent, err error) {
	out, err = db.getTransactionEvents(ctx, []string{idPrefix})
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, out)
	return
}

func splitEventsPerTrx(inPrefixes []string, flatEvents []*pbcodec.TransactionEvent) (out [][]*pbcodec.TransactionEvent) {
	for _, pref := range inPrefixes {
		var trxEvs []*pbcodec.TransactionEvent
		for _, ev := range flatEvents {
			if strings.HasPrefix(ev.Id, pref) {
				trxEvs = append(trxEvs, ev)
			}
		}
		out = append(out, trxEvs)
	}
	return
}

func (db *DB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	flat, err := db.getTransactionEvents(ctx, idPrefixes)
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, flat)
	if err != nil {
		return nil, err
	}
	out = splitEventsPerTrx(idPrefixes, flat)
	return
}

func (db *DB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	flat, err := db.getTransactionEvents(ctx, idPrefixes, TrxExecutionEvent)
	if err != nil {
		return nil, err
	}
	err = db.fillIrreversibilityData(ctx, flat)
	if err != nil {
		return nil, err
	}
	out = splitEventsPerTrx(idPrefixes, flat)
	return
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

	it := db.trxReadStore.BatchGet(ctx, prefixes)
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

func (db *DB) getTransactionEvents(ctx context.Context, idPrefixes []string, eventTypes ...TrxEventType) (out []*pbcodec.TransactionEvent, err error) {
	var keys [][]byte
	if len(eventTypes) == 0 { //default behavior is get all events
		eventTypes = []TrxEventType{TrxAdditionEvent, TrxExecutionEvent, ImplicitTrxEvent, DtrxEvent}
	}
	for _, idPrefix := range idPrefixes {
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
	}

	it := db.trxReadStore.BatchPrefix(ctx, keys, store.Unlimited)
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

			codec.ReduplicateTransactionTrace(row.TrxTrace)
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
