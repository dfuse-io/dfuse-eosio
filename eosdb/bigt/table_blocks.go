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
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	"github.com/golang/protobuf/proto"
)

type BlocksTable struct {
	*basebigt.BaseTable

	ColBlock                string
	ColMetaWritten          string
	ColMetaIrreversible     string
	ColTransactionRefs      string
	ColTransactionTraceRefs string
}

func NewBlocksTable(name string, client *bigtable.Client) *BlocksTable {
	return &BlocksTable{
		BaseTable:               basebigt.NewBaseTable(name, []string{"block", "meta", "trxs"}, client),
		ColBlock:                "block:proto",
		ColMetaWritten:          "meta:written",
		ColMetaIrreversible:     "meta:irreversible",
		ColTransactionRefs:      "trxs:trxRefsProto",
		ColTransactionTraceRefs: "trxs:traceRefsProto",
	}
}

func (tbl *BlocksTable) ReadRows(ctx context.Context, rowRange bigtable.RowSet, opts ...bigtable.ReadOption) (out []*pbcodec.BlockWithRefs, err error) {
	var innerErr error
	err = tbl.BaseTable.ReadRows(ctx, rowRange, func(row bigtable.Row) bool {
		response, err := tbl.ParseRowAs(row)
		if err != nil {
			innerErr = err
			return false
		}

		if response != nil {
			out = append(out, response)
		}
		return true
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("read block rows: %s", err)
	}
	if innerErr != nil {
		return nil, fmt.Errorf("read block rows, inner: %s", innerErr)
	}

	return
}

func (tbl *BlocksTable) ReadIrrCell(ctx context.Context, rowRange bigtable.RowSet, opts ...bigtable.ReadOption) (out []*pbcodec.BlockWithRefs, err error) {
	err = tbl.BaseTable.ReadRows(ctx, rowRange, func(row bigtable.Row) bool {
		out = append(out, &pbcodec.BlockWithRefs{
			Id: kvdb.ReversedBlockID(row.Key()),
		})
		return true
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("read block rows: %s", err)
	}

	return
}

func (tbl *BlocksTable) ParseRowAs(row bigtable.Row) (*pbcodec.BlockWithRefs, error) {
	fullyWritten, _ := basebigt.BoolColumnItem(row, tbl.ColMetaWritten)
	if !fullyWritten {
		return nil, nil
	}

	blk := &pbcodec.BlockWithRefs{
		Id:    kvdb.ReversedBlockID(row.Key()),
		Block: &pbcodec.Block{},
	}

	err := basebigt.ProtoColumnItem(row, tbl.ColBlock, func() proto.Message { return blk.Block })
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("block %s: %s", tbl.ColBlock, err)
	}

	// NotFound means false in this case, ignore error
	blk.Irreversible, _ = basebigt.BoolColumnItem(row, tbl.ColMetaIrreversible)

	protoResolver := func() proto.Message {
		blk.TransactionRefs = &pbcodec.TransactionRefs{}
		return blk.TransactionRefs
	}
	err = basebigt.ProtoColumnItem(row, tbl.ColTransactionRefs, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) && !basebigt.IsErrEmptyValue(err) {
		return nil, fmt.Errorf("block %s: %s", tbl.ColTransactionRefs, err)
	}

	protoResolver = func() proto.Message {
		blk.TransactionTraceRefs = &pbcodec.TransactionRefs{}
		return blk.TransactionTraceRefs
	}
	err = basebigt.ProtoColumnItem(row, tbl.ColTransactionTraceRefs, protoResolver)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) && !basebigt.IsErrEmptyValue(err) {
		return nil, fmt.Errorf("block %s: %s", tbl.ColTransactionTraceRefs, err)
	}

	return blk, nil
}

// PutBlock will strip the Header, Transactions and TransactionTraces from the block
// so you should put them separately.
//
// **Warning** This method is not concurrent safe! While in the method, we assume full
//             control of the passed `block` instance. You are responsible for ensuring
//             this preconditions holds.
func (tbl *BlocksTable) PutBlock(key string, block *pbcodec.Block) {
	// Keep a reference, so we can re-put the block correctly
	holdTransactions := block.Transactions
	holdTransactionTraces := block.TransactionTraces

	// Nullify not saved structure(s)
	block.Transactions = nil
	block.TransactionTraces = nil
	tbl.SetKey(key, tbl.ColBlock, kvdb.MustProtoMarshal(block))

	// Re-put back the held reference to the actual object
	block.Transactions = holdTransactions
	block.TransactionTraces = holdTransactionTraces
}

func (tbl *BlocksTable) PutTransactionRefs(key string, refs *pbcodec.TransactionRefs) {
	tbl.SetKey(key, tbl.ColTransactionRefs, kvdb.MustProtoMarshal(refs))
}

func (tbl *BlocksTable) PutTransactionTraceRefs(key string, refs *pbcodec.TransactionRefs) {
	tbl.SetKey(key, tbl.ColTransactionTraceRefs, kvdb.MustProtoMarshal(refs))
}

func (tbl *BlocksTable) PutMetaWritten(key string) {
	tbl.SetKey(key, tbl.ColMetaWritten, []byte{0x01})
}

func (tbl *BlocksTable) PutMetaIrreversible(key string, irreversible bool) {
	tbl.SetKey(key, tbl.ColMetaIrreversible, []byte{kvdb.BoolToByte(irreversible)})
}
