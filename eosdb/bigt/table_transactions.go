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

package bigt

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	"github.com/dfuse-io/kvdb"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type TransactionsTable struct {
	*basebigt.BaseTable

	ColTrx              string
	ColTrace            string
	ColDTrxCreatedBy    string
	ColDTrxCanceledBy   string
	ColMetaBlockHeader  string
	ColMetaIrreversible string
	ColMetaPubkeys      string
	ColMetaWritten      string
}

func NewTransactionsTable(name string, client *bigtable.Client) *TransactionsTable {
	return &TransactionsTable{
		BaseTable: basebigt.NewBaseTable(name, []string{"trx", "trace", "dtrx", "meta"}, client),

		ColTrx:              "trx:proto",
		ColTrace:            "trace:proto",
		ColDTrxCreatedBy:    "dtrx:created-by",
		ColDTrxCanceledBy:   "dtrx:canceled-by",
		ColMetaBlockHeader:  "meta:blockheader",
		ColMetaIrreversible: "meta:irreversible",
		ColMetaPubkeys:      "meta:pubkeys",
		ColMetaWritten:      "meta:written",
	}
}

func (tbl *TransactionsTable) ReadRows(ctx context.Context, rowRange bigtable.RowSet, opts ...bigtable.ReadOption) (out []*TransactionRow, err error) {
	var innerErr error
	err = tbl.BaseTable.ReadRows(ctx, rowRange, func(row bigtable.Row) bool {
		trxRow, err := tbl.ParseRowAs(row)
		if err != nil {
			innerErr = err
			return false
		}

		out = append(out, trxRow)
		return true
	}, opts...)

	if err != nil {
		return nil, fmt.Errorf("read transaction rows: %s", err)
	}

	if innerErr != nil {
		return nil, fmt.Errorf("read transaction rows, inner: %s", innerErr)
	}

	return out, nil
}

func (tbl *TransactionsTable) ReadEvents(ctx context.Context, rowRange bigtable.RowSet, opts ...bigtable.ReadOption) (out []*pbdeos.TransactionEvent, err error) {
	rows, err := tbl.ReadRows(ctx, rowRange, opts...)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		// FIXME: please don't reach here if it's not fully Written, parseRowAs should handle it first thing
		if !row.Written {
			continue
		}

		trxID, blockID, err := Keys.ReadTransaction(row.Key)
		if err != nil {
			return nil, err
		}

		newEv := func() *pbdeos.TransactionEvent {
			ev := &pbdeos.TransactionEvent{
				Id:           trxID,
				BlockId:      blockID,
				Irreversible: row.Irreversible,
			}
			out = append(out, ev)
			return ev
		}

		// Conditions for each Event types
		if row.TransactionTrace != nil {
			ev := newEv()
			ev.Event = &pbdeos.TransactionEvent_Execution{
				Execution: &pbdeos.TransactionEvent_Executed{
					Trace:       row.TransactionTrace,
					BlockHeader: row.BlockHeader,
				},
			}
		}
		if row.CreatedBy != nil {
			ev := newEv()
			ev.Event = &pbdeos.TransactionEvent_DtrxScheduling{
				DtrxScheduling: &pbdeos.TransactionEvent_DtrxScheduled{
					CreatedBy:   row.CreatedBy,
					Transaction: row.Transaction,
				},
			}
		} else if row.Transaction != nil {
			var pubKeys *pbdeos.PublicKeys
			if row.PublicKeys != nil {
				pubKeys = &pbdeos.PublicKeys{PublicKeys: row.PublicKeys}
			}
			ev := newEv()
			ev.Event = &pbdeos.TransactionEvent_Addition{
				Addition: &pbdeos.TransactionEvent_Added{
					// Receipt:     receipt, // FIXME: We currently don't have the receipt in the Bigtable implementation.. and this prevents us from reconstructing the blocks as they were originally (the receipt exists in the blocks logs).  It was later added to the SQL layer..  We could add it to bigtable, but we'd want to measure what's the increase in storage, and perhaps add it when we also implement compression at the bigtable-layer.
					Transaction: row.Transaction,
					PublicKeys:  pubKeys,
				},
			}

			// FIXME: if we ever implement the receipt, the internal additions do NOT have a
			// receipt.. so we'd distinguish them here. OR we instrument the node better and
			// retrieve the receipt for even those internal transactions, in which case we don't
			// need to have Internal vs Addition..
			// ev.Event = &pbdeos.TransactionEvent_InternalAddition{
			// 	InternalAddition: &pbdeos.TransactionEvent_AddedInternally{
			// 		Transaction: row.Transaction,
			// 	},
			// }
		}
		if row.CanceledBy != nil {
			ev := newEv()
			ev.Event = &pbdeos.TransactionEvent_DtrxCancellation{
				DtrxCancellation: &pbdeos.TransactionEvent_DtrxCanceled{
					CanceledBy: row.CanceledBy,
				},
			}
		}

	}

	return
}

func (tbl *TransactionsTable) ParseRowAs(row bigtable.Row) (*TransactionRow, error) {
	response := &TransactionRow{}
	response.Key = row.Key()

	protoResolver := func() proto.Message { response.Transaction = &pbdeos.SignedTransaction{}; return response.Transaction }
	err := basebigt.ProtoColumnItem(row, tbl.ColTrx, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction trx:proto: %s", err)
	}

	protoResolver = func() proto.Message {
		response.TransactionTrace = &pbdeos.TransactionTrace{}
		return response.TransactionTrace
	}
	err = basebigt.ProtoColumnItem(row, tbl.ColTrace, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction trace:proto: %s", err)
	}

	protoResolver = func() proto.Message { response.CreatedBy = &pbdeos.ExtDTrxOp{}; return response.CreatedBy }
	err = basebigt.ProtoColumnItem(row, tbl.ColDTrxCreatedBy, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction dtrx:created-by: %s", err)
	}

	protoResolver = func() proto.Message { response.CanceledBy = &pbdeos.ExtDTrxOp{}; return response.CanceledBy }
	err = basebigt.ProtoColumnItem(row, tbl.ColDTrxCanceledBy, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction dtrx:canceled-by: %s", err)
	}

	protoResolver = func() proto.Message { response.BlockHeader = &pbdeos.BlockHeader{}; return response.BlockHeader }
	err = basebigt.ProtoColumnItem(row, tbl.ColMetaBlockHeader, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction meta:blockheader: %s", err)
	}

	response.PublicKeys, err = basebigt.StringListColumnItem(row, tbl.ColMetaPubkeys, ":")
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction meta:pubkeys: %s", err)
	}

	response.Irreversible, err = basebigt.BoolColumnItem(row, tbl.ColMetaIrreversible)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction meta:irreversible: %s", err)
	}

	response.Written, err = basebigt.BoolColumnItem(row, tbl.ColMetaWritten)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("transaction meta:written: %s", err)
	}

	return response, nil
}

func (tbl *TransactionsTable) PutTrx(key string, trx *pbdeos.SignedTransaction) {
	tbl.SetKey(key, tbl.ColTrx, kvdb.MustProtoMarshal(trx))
}

func (tbl *TransactionsTable) PutTrace(key string, trace *pbdeos.TransactionTrace) {
	tbl.SetKey(key, tbl.ColTrace, kvdb.MustProtoMarshal(trace))
}

func (tbl *TransactionsTable) PutBlockHeader(key string, header *pbdeos.BlockHeader) {
	tbl.SetKey(key, tbl.ColMetaBlockHeader, kvdb.MustProtoMarshal(header))
}

func (tbl *TransactionsTable) PutPublicKeys(key string, publicKeys []string) {
	tbl.SetKey(key, tbl.ColMetaPubkeys, kvdb.StringListToBytes(publicKeys, ":"))
}

func (tbl *TransactionsTable) PutDTrxCreatedBy(key string, extDTrxOp *pbdeos.ExtDTrxOp) {
	tbl.SetKey(key, tbl.ColDTrxCreatedBy, kvdb.MustProtoMarshal(extDTrxOp))
}

func (tbl *TransactionsTable) PutDTrxCanceledBy(key string, extDTrxOp *pbdeos.ExtDTrxOp) {
	tbl.SetKey(key, tbl.ColDTrxCanceledBy, kvdb.MustProtoMarshal(extDTrxOp))
}

func (tbl *TransactionsTable) PutMetaWritten(key string, written bool) {
	tbl.SetKey(key, tbl.ColMetaWritten, []byte{kvdb.BoolToByte(written)})
}

func (tbl *TransactionsTable) PutMetaIrreversible(key string, irreversible bool) {
	tbl.SetKey(key, tbl.ColMetaIrreversible, []byte{kvdb.BoolToByte(irreversible)})
}

// stitchTransaction assumes that incoming rows will be ORDERED by
// blockID ascending, and assumes DEDUPLICATED rows.  In the cases of
// two rows in the same block, the rows list should be ordered as read
// by gjson or whatever.
// REMOVE ME, this is replaced by a pbdeos.MergeLifecycleEvents
func (tbl *TransactionsTable) stitchTransaction(rows []*TransactionRow, inCanonicalChain func(blockID string) bool) (*pbdeos.TransactionLifecycle, error) {
	inChain := func(isStepIrreversible, rowIrreversible bool, blockID string) bool {
		if rowIrreversible {
			return true
		}

		if !isStepIrreversible && inCanonicalChain(blockID) {
			return true
		}

		return false
	}

	responseKey := ""
	response := &pbdeos.TransactionLifecycle{
		PublicKeys: []string{},
	}

	executionIrr, creationIrr, cancelationIrr := computeStepsIrreversibility(rows)

	for _, row := range rows {
		if !row.Written {
			zlog.Debug("ignoring row that is not fully written yet", zap.String("row_key", row.Key))
			continue
		}

		_, blockID, err := Keys.ReadTransaction(row.Key)
		if err != nil {
			return nil, err
		}

		// FIXME: there's an execution_trace when we have a `PUSH_CREATE` dtrx.
		if row.TransactionTrace != nil {
			if !inChain(executionIrr, row.Irreversible, blockID) {
				continue
			}

			if row.Transaction != nil {
				responseKey = row.Key
			}

			response.ExecutionIrreversible = row.Irreversible
			if row.Irreversible {
				response.CreationIrreversible = true
			}

			if row.Transaction != nil {
				response.Transaction = row.Transaction
			}

			// This will override PREVIOUS rows: ex. the trace of
			// execution of the PUSH_CREATE dtrx creation, and also
			// any other CREATE.
			response.ExecutionBlockHeader = row.BlockHeader
			response.ExecutionTrace = row.TransactionTrace
			// PublicKeys might be set by the creation, so we want to
			// keep them. Doesn't hurt to append the execution time
			// (which will have none anyway).
			response.PublicKeys = append(response.PublicKeys, row.PublicKeys...)
		}

		if row.CreatedBy != nil {
			if !inChain(creationIrr, row.Irreversible, blockID) {
				continue
			}

			if row.Transaction != nil {
				responseKey = row.Key
				response.Transaction = row.Transaction
			}

			response.CreationIrreversible = row.Irreversible
			response.CreatedBy = row.CreatedBy
		}

		if row.CanceledBy != nil {
			if !inChain(cancelationIrr, row.Irreversible, blockID) {
				continue
			}

			response.CancelationIrreversible = row.Irreversible
			response.CanceledBy = row.CanceledBy
		}
	}

	if responseKey == "" {
		return nil, nil
	}

	trxID, _, err := Keys.ReadTransaction(responseKey)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction row key: %s", err)
	}

	response.Id = trxID
	response.TransactionStatus = getTransactionLifeCycleStatus(response)

	return response, nil
}

func getTransactionLifeCycleStatus(lifeCycle *pbdeos.TransactionLifecycle) pbdeos.TransactionStatus {
	// FIXME: this function belongs to the sample place as the stitcher, probably in `pbdeos`
	// alongside the rest.
	if lifeCycle.CanceledBy != nil {
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_CANCELED
	}

	if lifeCycle.ExecutionTrace == nil {
		if lifeCycle.CreatedBy != nil {
			return pbdeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED
		}

		// FIXME: It was `pending` before but not present anymore, what should we do?
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_NONE
	}

	if lifeCycle.ExecutionTrace.Receipt == nil {
		// That happen strangely on EOS Kylin where `eosio:onblock` started to fail and exhibit no Receipt
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL
	}

	// Expired Failed Executed
	return lifeCycle.ExecutionTrace.Receipt.Status
}

func computeStepsIrreversibility(rows []*TransactionRow) (executionIrr bool, creationIrr bool, cancelationIrr bool) {
	for _, row := range rows {
		if !row.Written {
			zlog.Debug("ignoring row that is not fully written yet", zap.String("row_key", row.Key))
			continue
		}

		if row.TransactionTrace != nil && row.Irreversible {
			if row.TransactionTrace.Receipt != nil && row.TransactionTrace.Receipt.Status == pbdeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED {
				creationIrr = true
			} else {
				executionIrr = true
			}
		}

		if row.CreatedBy != nil && row.Irreversible {
			creationIrr = true
		}

		// FIXME: eeeee boderek c'est tu du copy paste ça?
		if row.CreatedBy != nil && row.Irreversible {
			creationIrr = true
		}

		if row.CanceledBy != nil && row.Irreversible {
			cancelationIrr = true
		}
	}

	return
}
