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

package kv

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	kvdbstore "github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
)

func (db *DB) Flush(ctx context.Context) error {
	return db.writeStore.FlushPuts(ctx)
}

func (db *DB) SetWriterChainID(chainID []byte) {
	db.writerChainID = chainID
}

func (db *DB) purgeSetupAndAttempt(ctx context.Context, s kvdbstore.KVStore, blkNumber uint64) error {
	if s, ok := s.(kvdbstore.Purgeable); ok {
		s.MarkCurrentHeight(blkNumber)
		if blkNumber > 0 && (blkNumber%db.purgeInterval) == 0 {
			if traceEnabled {
				db.logger.Debug("purging keys", zap.Uint64("block_num", blkNumber))
			}
			if err := s.PurgeKeys(ctx); err != nil {
				return fmt.Errorf("unable to purge store: %w", err)
			}
		}
	}
	return nil
}

func (db *DB) PutBlock(ctx context.Context, blk *pbcodec.Block) error {
	if db.enableTrxWrite {
		err := db.purgeSetupAndAttempt(ctx, db.writeStore, uint64(blk.Number))
		if err != nil {
			return err
		}

		if traceEnabled {
			db.logger.Debug("put transactions (trx, trace, dtrx)")
		}

		if err := db.putTransactions(ctx, blk); err != nil {
			return fmt.Errorf("put block: unable to putTransactions: %w", err)
		}

		if err := db.putTransactionTraces(ctx, blk); err != nil {
			return fmt.Errorf("put block: unable to putTransactions: %w", err)
		}

		if err := db.putImplicitTransactions(ctx, blk); err != nil {
			return fmt.Errorf("put block: unable to putTransactions: %w", err)
		}
	} else {
		db.logger.Debug("skipping transaction write")
	}

	if db.enableBlkWrite {
		err := db.purgeSetupAndAttempt(ctx, db.writeStore, uint64(blk.Number))
		if err != nil {
			return err
		}

		return db.putBlock(ctx, blk)
	}

	// NOTE: what happens to the blockNum, for the IrrBlock rows?? Do we truncate it when it
	// becomes irreversible?
	db.logger.Debug("skipping block write")
	return nil
}

func (db *DB) putTransactions(ctx context.Context, blk *pbcodec.Block) error {
	for _, trxReceipt := range blk.Transactions() {
		if trxReceipt.PackedTransaction == nil {
			// This means we deal with a deferred transaction receipt, and that it
			// has been handled through DtrxOps already
			continue
		}

		signedTransaction, err := codec.ExtractEOSSignedTransactionFromReceipt(trxReceipt)
		if err != nil {
			return fmt.Errorf("unable to extract EOS signed transaction from transaction receipt: %w", err)
		}

		signedTrx := codec.SignedTransactionToDEOS(signedTransaction)
		pubKeyProto := &pbcodec.PublicKeys{
			PublicKeys: codec.GetPublicKeysFromSignedTransaction(db.writerChainID, signedTransaction),
		}

		trxRow := &pbtrxdb.TrxRow{
			Receipt:    trxReceipt,
			SignedTrx:  signedTrx,
			PublicKeys: pubKeyProto,
		}

		key := Keys.PackTrxsKey(trxReceipt.Id, blk.Id)
		// NOTE: This function is guarded by the parent with db.enableTrxWrite
		err = db.writeStore.Put(ctx, key, db.enc.MustProto(trxRow))

		if err != nil {
			return fmt.Errorf("put trx: write to db: %w", err)
		}
	}

	return nil
}

func (db *DB) putTransactionTraces(ctx context.Context, blk *pbcodec.Block) error {
	for _, trxTrace := range blk.TransactionTraces() {
		// CHECK: can we have multiple dtrxops for the same transactionId in the same block?
		for _, dtrxOp := range trxTrace.DtrxOps {
			extDtrxOp := dtrxOp.ToExtDTrxOp(blk, trxTrace)

			dtrxRow := &pbtrxdb.DtrxRow{}

			var key []byte
			if dtrxOp.IsCreateOperation() {
				dtrxRow.SignedTrx = dtrxOp.Transaction
				dtrxRow.CreatedBy = extDtrxOp
				key = Keys.PackDtrxsKeyCreated(dtrxOp.TransactionId, blk.Id)
			} else if dtrxOp.IsCancelOperation() {
				dtrxRow.CanceledBy = extDtrxOp
				key = Keys.PackDtrxsKeyCancelled(dtrxOp.TransactionId, blk.Id)
			} else if dtrxOp.IsFailedOperation() {
				key = Keys.PackDtrxsKeyFailed(dtrxOp.TransactionId, blk.Id)
			} else {
				return fmt.Errorf("put dtrxRow: handle dtrxOp Operation: unknown dtrxOp operation for trx id %s at action %d", trxTrace.Id, dtrxOp.ActionIndex)
			}

			// NOTE: This function is guarded by the parent with db.enableTrxWrite
			if err := db.writeStore.Put(ctx, key, db.enc.MustProto(dtrxRow)); err != nil {
				return fmt.Errorf("put dtrxRow: write to db: %w", err)
			}
		}

		codec.DeduplicateTransactionTrace(trxTrace)

		trxTraceRow := &pbtrxdb.TrxTraceRow{
			BlockHeader: blk.Header,
			TrxTrace:    trxTrace,
		}

		if traceEnabled {
			db.logger.Debug("put transaction trace row", zap.String("trx_id", trxTrace.Id), zap.String("block_id", blk.Id))
		}

		key := Keys.PackTrxTracesKey(trxTrace.Id, blk.Id)
		// NOTE: This function is guarded by the parent with db.enableTrxWrite
		if err := db.writeStore.Put(ctx, key, db.enc.MustProto(trxTraceRow)); err != nil {
			return fmt.Errorf("put trxTraceRow: write to db: %w", err)
		}

		codec.ReduplicateTransactionTrace(trxTrace)
	}

	return nil
}

func (db *DB) putNewAccount(ctx context.Context, blk *pbcodec.Block, trace *pbcodec.TransactionTrace, act *pbcodec.ActionTrace) error {
	t, err := ptypes.TimestampProto(blk.MustTime())
	if err != nil {
		return fmt.Errorf("block time to proto: %w", err)
	}

	acctRow := &pbtrxdb.AccountRow{
		Name:      act.GetData("name").String(),
		Creator:   act.GetData("creator").String(),
		BlockTime: t,
		BlockId:   blk.Id,
		TrxId:     trace.Id,
	}

	if traceEnabled {
		db.logger.Debug("put account row", zap.String("name", acctRow.Name))
	}

	// NOTE: This function is guarded by the parent with db.enableBlkWrite
	key := Keys.PackAccountKey(acctRow.Name)
	if err := db.writeStore.Put(ctx, key, db.enc.MustProto(acctRow)); err != nil {
		return fmt.Errorf("put acctRow: write to db: %w", err)
	}

	return nil
}

func (db *DB) putImplicitTransactions(ctx context.Context, blk *pbcodec.Block) error {
	for _, trxOp := range blk.ImplicitTransactionOps() {
		implTrxRow := &pbtrxdb.ImplicitTrxRow{
			Name:      trxOp.Name,
			SignedTrx: trxOp.Transaction,
		}

		key := Keys.PackImplicitTrxsKey(trxOp.TransactionId, blk.Id)
		// NOTE: This function is guarded by the parent with db.enableTrxWrite
		if err := db.writeStore.Put(ctx, key, db.enc.MustProto(implTrxRow)); err != nil {
			return fmt.Errorf("put implTrx: write to db: %w", err)
		}
	}

	return nil
}

func (db *DB) getRefs(blk *pbcodec.Block) (implicitTrxRefs, trxRefs, tracesRefs *pbcodec.TransactionRefs) {
	implicitTrxRefs = &pbcodec.TransactionRefs{}
	for _, trxOp := range blk.ImplicitTransactionOps() {
		implicitTrxRefs.Hashes = append(implicitTrxRefs.Hashes, trxdb.MustHexDecode(trxOp.TransactionId))
	}

	trxRefs = &pbcodec.TransactionRefs{}
	for _, trx := range blk.Transactions() {
		trxRefs.Hashes = append(trxRefs.Hashes, trxdb.MustHexDecode(trx.Id))
	}

	tracesRefs = &pbcodec.TransactionRefs{}
	for _, trx := range blk.TransactionTraces() {
		tracesRefs.Hashes = append(tracesRefs.Hashes, trxdb.MustHexDecode(trx.Id))
	}

	return
}

func (db *DB) putBlock(ctx context.Context, blk *pbcodec.Block) error {
	implicitTrxRefs, trxRefs, tracesRefs := db.getRefs(blk)

	holdUnfilteredTransactions := blk.UnfilteredTransactions
	holdUnfilteredTransactionTraces := blk.UnfilteredTransactionTraces
	holdUnfilteredImplicitTransactionOps := blk.UnfilteredImplicitTransactionOps

	holdFilteredTransactions := blk.FilteredTransactions
	holdFilteredTransactionTraces := blk.FilteredTransactionTraces
	holdFilteredImplicitTransactionOps := blk.FilteredImplicitTransactionOps

	blk.UnfilteredTransactions = nil
	blk.UnfilteredTransactionTraces = nil
	blk.UnfilteredImplicitTransactionOps = nil

	blk.FilteredTransactions = nil
	blk.FilteredTransactionTraces = nil
	blk.FilteredImplicitTransactionOps = nil

	blockRow := &pbtrxdb.BlockRow{
		Block:           blk,
		ImplicitTrxRefs: implicitTrxRefs,
		TrxRefs:         trxRefs,
		TraceRefs:       tracesRefs,
	}

	db.logger.Debug("put block", zap.Stringer("block", blk.AsRef()))
	key := Keys.PackBlocksKey(blk.Id)
	// NOTE: This function is guarded by the parent with db.enableBlkWrite
	if err := db.writeStore.Put(ctx, key, db.enc.MustProto(blockRow)); err != nil {
		return fmt.Errorf("put block: write to db: %w", err)
	}

	blk.UnfilteredTransactions = holdUnfilteredTransactions
	blk.UnfilteredTransactionTraces = holdUnfilteredTransactionTraces
	blk.UnfilteredImplicitTransactionOps = holdUnfilteredImplicitTransactionOps

	blk.FilteredTransactions = holdFilteredTransactions
	blk.FilteredTransactionTraces = holdFilteredTransactionTraces
	blk.FilteredImplicitTransactionOps = holdFilteredImplicitTransactionOps

	return nil
}

var oneByte = []byte{0x01}

func (db *DB) UpdateNowIrreversibleBlock(ctx context.Context, blk *pbcodec.Block) error {
	if db.enableBlkWrite {
		blockTime := blk.MustTime()
		if err := db.writeStore.Put(ctx, Keys.PackTimelineKey(true, blockTime, blk.Id), oneByte); err != nil {
			return err
		}
		if err := db.writeStore.Put(ctx, Keys.PackTimelineKey(false, blockTime, blk.Id), oneByte); err != nil {
			return err
		}
	} else {
		db.logger.Debug("timeline is not written, skipping")
	}

	if db.enableBlkWrite {
		// Specialized indexing for `newaccount` on the chain, this might loop on filtered transaction traces, so
		// the filtering rules might exclude the `newaccount`.
		for _, trxTrace := range blk.TransactionTraces() {
			for _, act := range trxTrace.ActionTraces {
				if act.FullName() == "eosio:eosio:newaccount" {
					if err := db.putNewAccount(ctx, blk, trxTrace, act); err != nil {
						return fmt.Errorf("failed to put new account: %w", err)
					}
				}
			}
		}
	} else {
		db.logger.Debug("account is not written, skipping")
	}

	if db.writeStore != nil {
		// We must do this operation regardless of the write only categories set since this is used
		// as our last block marker. If this would not be writing, it would never be possible to start
		// back where we left off.
		db.logger.Debug("adding irreversible block", zap.Stringer("block", blk.AsRef()))
		if err := db.writeStore.Put(ctx, Keys.PackIrrBlocksKey(blk.Id), oneByte); err != nil {
			return err
		}
	}

	// NOTE: what happens to the blockNum, for the IrrBlock rows?? Do we truncate it when it
	// becomes irreversible?

	return nil
}
