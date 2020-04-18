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
	"math"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

func (db *DB) prepareStatements() error {
	// TODO: implement prepared statements, to speed-up insertion
	return nil
}

func (db *DB) SetWriterChainID(chainID []byte) {
	db.writerChainID = chainID
}

func (db *DB) GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error) {
	return db.GetClosestIrreversibleIDAtBlockNum(ctx, math.MaxUint32)
}

func (db *DB) Close() error {
	// TODO: close something..
	return nil
}

func (db *DB) PutBlock(ctx context.Context, blk *pbcodec.Block) error {
	defer db.db.ExecContext(ctx, "SET autocommit=1")

	_, err := db.db.ExecContext(ctx, "SET autocommit=0")
	_, err = db.db.ExecContext(ctx, BEGIN_TRANSACTION)

	if err != nil {
		zlog.Error("Rollback", zap.Error(err))
		_, err = db.db.ExecContext(ctx, "ROLLBACK")
		return err
	}

	if err := db.putTransactions(ctx, blk); err != nil {
		zlog.Error("Rollback", zap.Error(err))
		_, err = db.db.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("put transactions: %w", err)
	}

	if err := db.putTransactionTraces(ctx, blk); err != nil {
		zlog.Error("Rollback", zap.Error(err))
		_, err = db.db.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("put transaction traces: %w", err)
	}

	if err := db.putImplicitTransactions(ctx, blk); err != nil {
		zlog.Error("Rollback", zap.Error(err))
		_, err = db.db.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("put implicit transactions: %w", err)
	}

	implicitTrxRefs, trxRefs, tracesRefs := db.getRefs(blk)

	holdTransactions := blk.Transactions
	holdTransactionTraces := blk.TransactionTraces
	holdImplicitTransactionOps := blk.ImplicitTransactionOps

	blk.ImplicitTransactionOps = nil
	blk.Transactions = nil
	blk.TransactionTraces = nil

	_, err = db.db.ExecContext(ctx, INSERT_IGNORE+"INTO blks (id, number, previousId, irrBlockNum, blockProducer, blockTime, block, blockHeader, trxRefs, traceRefs, implicitTrxRefs) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		blk.ID(), blk.Num(), blk.Header.Previous, blk.DposIrreversibleBlocknum, blk.Header.Producer, blk.MustTime(), db.enc.MustProto(blk), db.enc.MustProto(blk.Header), db.enc.MustProto(trxRefs), db.enc.MustProto(tracesRefs), db.enc.MustProto(implicitTrxRefs))
	if err != nil {
		zlog.Error("Rollback", zap.Error(err))
		_, err = db.db.ExecContext(ctx, "ROLLBACK")
		return err
	}
	_, err = db.db.ExecContext(ctx, "COMMIT")

	blk.ImplicitTransactionOps = holdImplicitTransactionOps
	blk.Transactions = holdTransactions
	blk.TransactionTraces = holdTransactionTraces

	return nil
}

func (db *DB) Flush(ctx context.Context) error {
	return nil
}

func (db *DB) putTransactions(ctx context.Context, blk *pbcodec.Block) error {
	for _, trxReceipt := range blk.Transactions {
		if trxReceipt.PackedTransaction == nil {
			// This means we deal with a deferred transaction receipt, and that it has been handled through DtrxOps already
			continue
		}

		signedTransaction, err := codec.ExtractEOSSignedTransactionFromReceipt(trxReceipt)
		if err != nil {
			return fmt.Errorf("unable to extract EOS signed transaction from transaction receipt: %s", err)
		}

		signedTrx := codec.SignedTransactionToDEOS(signedTransaction)
		pubKeyProto := &pbcodec.PublicKeys{
			PublicKeys: codec.GetPublicKeysFromSignedTransaction(db.writerChainID, signedTransaction),
		}

		// trxKey := eosdb.Keys.Transaction(trxReceipt.Id, blk.ID())
		// db.Transactions.PutTrx(trxKey, signedTrx)
		// db.Transactions.PutPublicKeys(trxKey, pubKeys)

		_, err = db.db.ExecContext(ctx, INSERT_IGNORE+"INTO trxs (id, blockId, receipt, signedTrx, publicKeys) VALUES (?, ?, ?, ?, ?)", trxReceipt.Id, blk.ID(), db.enc.MustProto(trxReceipt), db.enc.MustProto(signedTrx), db.enc.MustProto(pubKeyProto))
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) putTransactionTraces(ctx context.Context, blk *pbcodec.Block) error {
	//marshaledHeader := db.enc.MustProto(blk.Header)

	for _, trxTrace := range blk.TransactionTraces {

		// CHECK: can we have multiple dtrxops for the same transactionId in the same block?
		for _, dtrxOp := range trxTrace.DtrxOps {
			extDtrxOp := dtrxOp.ToExtDTrxOp(blk, trxTrace)

			var signedTrx, created, canceled []byte

			if dtrxOp.IsCreateOperation() {
				signedTrx = db.enc.MustProto(dtrxOp.Transaction)
				created = db.enc.MustProto(extDtrxOp)
			} else if dtrxOp.IsCancelOperation() {
				canceled = db.enc.MustProto(extDtrxOp)
			}

			_, err := db.db.ExecContext(ctx, INSERT_IGNORE+"INTO dtrxs (id, blockId, signedTrx, createdBy, canceledBy) VALUES (?, ?, ?, ?, ?)",
				dtrxOp.TransactionId, blk.ID(), signedTrx, created, canceled)
			if err != nil {
				return err
			}
		}

		// Specialized indexing for `newaccount` on the chain.
		for _, act := range trxTrace.ActionTraces {
			if act.Account() == "eosio" && act.Receiver == "eosio" && act.Name() == "newaccount" {
				err := db.putNewAccount(blk, trxTrace, act)
				if err != nil {
					return fmt.Errorf("failed to put new account: %w", err)
				}
			}
		}

		// trxTraceKey := Keys.Transaction(trxTrace.Id, blockID)
		// db.Transactions.PutTrace(trxTraceKey, trxTrace)
		// db.Transactions.PutBlockHeader(trxTraceKey, blk.Header)
		_, err := db.db.Exec(INSERT_IGNORE+"INTO trxtraces (id, blockId, trace) VALUES (?, ?, ?)", trxTrace.Id, blk.ID(), db.enc.MustProto(trxTrace))
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) putImplicitTransactions(ctx context.Context, blk *pbcodec.Block) error {
	// name is "onblock" or "onerror"
	for _, trxOp := range blk.ImplicitTransactionOps {
		_, err := db.db.ExecContext(ctx, INSERT_IGNORE+"INTO implicittrxs (id, blockId, name, signedTrx) VALUES (?, ?, ?, ?)", trxOp.TransactionId, blk.ID(), trxOp.Name, db.enc.MustProto(trxOp.Transaction))
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) getRefs(blk *pbcodec.Block) (implicitTrxRefs, trxRefs, tracesRefs *pbcodec.TransactionRefs) {
	implicitTrxRefs = &pbcodec.TransactionRefs{}
	for _, trxOp := range blk.ImplicitTransactionOps {
		implicitTrxRefs.Hashes = append(implicitTrxRefs.Hashes, eosdb.MustHexDecode(trxOp.TransactionId))
	}

	trxRefs = &pbcodec.TransactionRefs{}
	for _, trx := range blk.Transactions {
		trxRefs.Hashes = append(trxRefs.Hashes, eosdb.MustHexDecode(trx.Id))
	}

	tracesRefs = &pbcodec.TransactionRefs{}
	for _, trx := range blk.TransactionTraces {
		tracesRefs.Hashes = append(tracesRefs.Hashes, eosdb.MustHexDecode(trx.Id))
	}

	return
}

func (db *DB) UpdateNowIrreversibleBlock(ctx context.Context, blk *pbcodec.Block) error {
	_, err := db.db.ExecContext(ctx, INSERT_IGNORE+"INTO irrblks (id, irreversible) VALUES (?, ?)", blk.ID(), true)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) FlushAllMutations(context.Context) error {
	return nil
}
