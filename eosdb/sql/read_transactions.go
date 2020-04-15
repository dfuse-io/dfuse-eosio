// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sql

import (
	"context"
	"fmt"

	"github.com/dfuse-io/kvdb"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/eosdb/mdl"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
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
	// TODO: LOOP to get all those addition events (they can come from different blocks)
	q := `SELECT trxs.id, trxs.blockId, receipt, signedTrx, publicKeys, irrblks.irreversible
          FROM trxs
          LEFT JOIN irrblks ON (trxs.blockId = irrblks.id)
          WHERE trxs.id LIKE ?`
	rows, err := db.db.QueryContext(ctx, q, idPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("running query: %s", err)
	}

	for rows.Next() {
		ev := &pbeos.TransactionEvent{}
		var receiptData, signedTrxData, pubkeysData []byte
		var irr *bool
		err := rows.Scan(
			&ev.Id, &ev.BlockId, &receiptData, &signedTrxData, &pubkeysData, &irr,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning: %s", err)
		}

		ev.Irreversible = eosdb.BoolPtr(irr)

		signedTrx := &pbeos.SignedTransaction{}
		if err := db.dec.Into(signedTrxData, signedTrx); err != nil {
			return nil, fmt.Errorf("decode signed trx: %s", err)
		}

		if receiptData != nil {
			receipt := &pbeos.TransactionReceipt{}
			if err := db.dec.Into(receiptData, receipt); err != nil {
				return nil, fmt.Errorf("decode trx receipt: %s", err)
			}

			publicKeys := &pbeos.PublicKeys{}
			if err := db.dec.Into(pubkeysData, publicKeys); err != nil {
				return nil, fmt.Errorf("decode public keys: %s", err)
			}

			ev.Event = &pbeos.TransactionEvent_Addition{
				Addition: &pbeos.TransactionEvent_Added{
					Receipt:     receipt,
					Transaction: signedTrx,
					PublicKeys:  publicKeys,
				},
			}
		} else {
			ev.Event = &pbeos.TransactionEvent_InternalAddition{
				InternalAddition: &pbeos.TransactionEvent_AddedInternally{
					Transaction: signedTrx,
				},
			}
		}

		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

func (db *DB) getTransactionImplicitEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	q := `SELECT implicittrxs.id, implicittrxs.blockId, implicittrxs.signedTrx, irrblks.irreversible
          FROM implicittrxs
          LEFT JOIN irrblks ON (implicittrxs.blockId = irrblks.id)
          WHERE implicittrxs.id LIKE ?`
	rows, err := db.db.QueryContext(ctx, q, idPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("running query: %s", err)
	}

	for rows.Next() {
		ev := &pbeos.TransactionEvent{}
		var signedTrxData []byte
		var irr *bool
		err := rows.Scan(
			&ev.Id, &ev.BlockId, &signedTrxData, &irr,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning: %s", err)
		}

		ev.Irreversible = eosdb.BoolPtr(irr)

		signedTrx := &pbeos.SignedTransaction{}
		if err := db.dec.Into(signedTrxData, signedTrx); err != nil {
			return nil, fmt.Errorf("decode signed trx: %s", err)
		}

		ev.Event = &pbeos.TransactionEvent_InternalAddition{
			InternalAddition: &pbeos.TransactionEvent_AddedInternally{
				Transaction: signedTrx,
			},
		}

		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

func (db *DB) getTransactionExecutionEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	q := `SELECT trxtraces.id, trxtraces.blockId, trace, irrblks.irreversible, blks.blockHeader
          FROM trxtraces
          LEFT JOIN irrblks ON (trxtraces.blockId = irrblks.id)
          LEFT JOIN blks ON (trxtraces.blockId = blks.id)
          WHERE trxtraces.id LIKE ?`
	rows, err := db.db.QueryContext(ctx, q, idPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("running query: %s", err)
	}

	for rows.Next() {
		ev := &pbeos.TransactionEvent{}
		var traceData, blockHeaderData []byte
		var irr *bool
		err := rows.Scan(
			&ev.Id, &ev.BlockId, &traceData, &irr, &blockHeaderData,
		)
		if err != nil {
			return nil, err
		}

		ev.Irreversible = eosdb.BoolPtr(irr)

		trace := &pbeos.TransactionTrace{}
		if err := db.dec.Into(traceData, trace); err != nil {
			return nil, fmt.Errorf("decode trace: %s", err)
		}

		blockHeader := &pbeos.BlockHeader{}
		if err := db.dec.Into(blockHeaderData, blockHeader); err != nil {
			return nil, fmt.Errorf("decode block header: %s", err)
		}

		ev.Event = &pbeos.TransactionEvent_Execution{
			Execution: &pbeos.TransactionEvent_Executed{
				Trace:       trace,
				BlockHeader: blockHeader,
			},
		}

		out = append(out, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return
}

func (db *DB) getTransactionDtrxEvents(ctx context.Context, idPrefix string) (out []*pbeos.TransactionEvent, err error) {
	q := `SELECT dtrxs.id, dtrxs.blockId, signedTrx, createdBy, canceledBy, irrblks.irreversible
          FROM dtrxs
          LEFT JOIN irrblks ON (dtrxs.blockId = irrblks.id)
          WHERE dtrxs.id LIKE ?`
	rows, err := db.db.QueryContext(ctx, q, idPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("running query: %s", err)
	}

	for rows.Next() {
		ev := &pbeos.TransactionEvent{}
		var signedTrxData, createdByData, canceledByData []byte
		err := rows.Scan(
			&ev.Id, &ev.BlockId, &signedTrxData, &createdByData, &canceledByData, &ev.Irreversible,
		)
		if err != nil {
			return nil, err
		}

		// TODO: ev.Irreversible will crash when `irrblks` has no data,
		// as irreversible will be NULL, use `BoolPtr()` like in `read_blocks`

		signedTrx := &pbeos.SignedTransaction{}
		if err := db.dec.Into(signedTrxData, signedTrx); err != nil {
			return nil, fmt.Errorf("decode signed trx: %s", err)
		}

		if createdByData != nil {
			createdBy := &pbeos.ExtDTrxOp{}
			if err := db.dec.Into(createdByData, createdBy); err != nil {
				return nil, fmt.Errorf("decode createdBy: %s", err)
			}

			signedTrx := &pbeos.SignedTransaction{}
			if err := db.dec.Into(signedTrxData, signedTrx); err != nil {
				return nil, fmt.Errorf("decode signedTrx: %s", err)
			}

			ev.Event = &pbeos.TransactionEvent_DtrxScheduling{
				DtrxScheduling: &pbeos.TransactionEvent_DtrxScheduled{
					CreatedBy:   createdBy,
					Transaction: signedTrx,
				},
			}
		} else {
			canceledBy := &pbeos.ExtDTrxOp{}
			if err := db.dec.Into(canceledByData, canceledBy); err != nil {
				return nil, fmt.Errorf("decode canceledBy: %s", err)
			}

			ev.Event = &pbeos.TransactionEvent_DtrxCancellation{
				DtrxCancellation: &pbeos.TransactionEvent_DtrxCanceled{
					CanceledBy: canceledBy,
				},
			}
		}

		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

func (db *DB) ListMostRecentTransactions(ctx context.Context, startKey string, limit int, chainDiscriminator eosdb.ChainDiscriminator) (*mdl.TransactionList, error) {
	return nil, nil
}

func (db *DB) ListTransactionsForBlockID(ctx context.Context, id string, startKey string, limit int, chainDiscriminator eosdb.ChainDiscriminator) (*mdl.TransactionList, error) {
	return nil, nil
}
