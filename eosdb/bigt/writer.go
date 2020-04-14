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
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/dfuse-io/bstream"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
)

func (db *EOSDatabase) flushIfNeeded(ctx context.Context) error {
	if db.blocksSinceFlush >= db.maxBlocksBeforeFlush || time.Since(db.lastFlushTime) > db.maxDurationBeforeFlush {
		err := db.FlushAllMutations(ctx)
		db.lastFlushTime = time.Now()
		db.blocksSinceFlush = 0
		return err
	}
	return nil

}

// TODO we should lowercase all methods under db.Blocks or itself to prevent people from not getting the autoflush
// TODO get context right here
func (db *EOSDatabase) PutBlock(ctx context.Context, blk *pbdeos.Block) error {
	btKey := Keys.Block(blk.Id)

	db.Blocks.PutBlock(btKey, blk)
	db.Blocks.PutTransactionRefs(btKey, getTransactionRefs(blk))
	db.Blocks.PutTransactionTraceRefs(btKey, getTransactionTraceRefs(blk))

	transactionKeys, err := db.storeTransactions(blk)
	if err != nil {
		return fmt.Errorf("unable to store transactions in block #%d (%s): %s", blk.Number, blk.Id, err)
	}

	for transactionKey := range transactionKeys {
		db.Transactions.PutMetaWritten(transactionKey, true)
	}

	db.BlocksLast.PutMetaWritten(btKey)
	db.blocksSinceFlush++
	return db.flushIfNeeded(ctx)

	return nil
}

func (db *EOSDatabase) UpdateNowIrreversibleBlock(ctx context.Context, blk *pbdeos.Block) error {
	blockID := blk.ID()
	blockTime := blk.MustTime()

	// Timeline
	fKey := Keys.TimelineBlockForward(blockTime, blockID)
	db.Timeline.PutMetaExists(fKey)

	//if _, ok := db.Db.(ReversibleKVStore); !ok   {
	rKey := Keys.TimelineBlockReverse(blockTime, blockID)
	db.Timeline.PutMetaExists(rKey)
	//}

	// Irreversible Transactions
	for _, trxTrace := range blk.TransactionTraces {
		btKey := Keys.Transaction(trxTrace.Id, blockID)
		db.Transactions.PutMetaIrreversible(btKey, true)

		for _, act := range trxTrace.ActionTraces {
			if act.FullName() == "eosio:eosio:newaccount" {
				db.storeNewAccount(blk, trxTrace, act)
			}
		}
	}

	for _, trxID := range blk.CreatedDTrxIDs() {
		btKey := Keys.Transaction(trxID, blockID)
		db.Transactions.PutMetaIrreversible(btKey, true)
	}

	for _, trxID := range blk.CanceledDTrxIDs() {
		btKey := Keys.Transaction(trxID, blockID)
		db.Transactions.PutMetaIrreversible(btKey, true)
	}

	// Irreversible Block
	btKey := Keys.Block(blockID)
	db.Blocks.PutMetaIrreversible(btKey, true)

	db.blocksSinceFlush++
	if db.blocksSinceFlush >= db.maxBlocksBeforeFlush || time.Since(db.lastFlushTime) > db.maxDurationBeforeFlush {
		db.FlushAllMutations(context.Background()) // FIXME context
	}

	return nil
}

func (db *EOSDatabase) GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error) {
	return db.GetClosestIrreversibleIDAtBlockNum(ctx, math.MaxUint32)
}

func getTransactionRefs(blk *pbdeos.Block) *pbdeos.TransactionRefs {
	var hashes [][]byte
	for _, trxOp := range blk.ImplicitTransactionOps {
		hashes = append(hashes, mustHexDecode(trxOp.TransactionId))
	}

	for _, trxReceipt := range blk.Transactions {
		hashes = append(hashes, mustHexDecode(trxReceipt.Id))
	}

	return &pbdeos.TransactionRefs{
		Hashes: hashes,
	}
}

func getTransactionTraceRefs(blk *pbdeos.Block) *pbdeos.TransactionRefs {
	var hashes [][]byte
	for _, trxTrace := range blk.TransactionTraces {
		hashes = append(hashes, mustHexDecode(trxTrace.Id))
	}

	return &pbdeos.TransactionRefs{
		Hashes: hashes,
	}
}

func mustHexDecode(input string) []byte {
	value, err := hex.DecodeString(input)
	if err != nil {
		panic(fmt.Errorf("should have been possible to transform decode %q as hex: %s", input, err))
	}

	return value
}
