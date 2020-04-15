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
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/codec"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
)

func (db *EOSDatabase) storeTransactions(blk *pbeos.Block) (map[string]bool, error) {
	blockID := blk.Id
	transactionKeys := map[string]bool{}

	for _, trxOp := range blk.ImplicitTransactionOps {
		trxKey := Keys.Transaction(trxOp.TransactionId, blockID)

		db.Transactions.PutTrx(trxKey, trxOp.Transaction)
		transactionKeys[trxKey] = true
	}

	for _, trxReceipt := range blk.Transactions {
		if trxReceipt.PackedTransaction == nil {
			// This means we deal with a deferred transaction receipt, and that it has been handled through DtrxOps already
			continue
		}

		signedTransaction, err := codec.ExtractEOSSignedTransactionFromReceipt(trxReceipt)
		if err != nil {
			return nil, fmt.Errorf("unable to extract EOS signed transaction from transaction receipt: %s", err)
		}

		trxKey := Keys.Transaction(trxReceipt.Id, blockID)
		db.Transactions.PutTrx(trxKey, codec.SignedTransactionToDEOS(signedTransaction))
		db.Transactions.PutPublicKeys(trxKey, codec.GetPublicKeysFromSignedTransaction(db.writerChainID, signedTransaction))

		transactionKeys[trxKey] = true
	}

	for _, trxTrace := range blk.TransactionTraces {
		trxTraceKey := Keys.Transaction(trxTrace.Id, blockID)

		for _, dtrxOp := range trxTrace.DtrxOps {
			dtrxKey := Keys.Transaction(dtrxOp.TransactionId, blockID)
			extDtrxOp := dtrxOp.ToExtDTrxOp(blk, trxTrace)

			if dtrxOp.IsCreateOperation() {
				db.Transactions.PutTrx(dtrxKey, dtrxOp.Transaction)
				db.Transactions.PutDTrxCreatedBy(dtrxKey, extDtrxOp)
			} else if dtrxOp.IsCancelOperation() {
				db.Transactions.PutDTrxCanceledBy(dtrxKey, extDtrxOp)
			}

			transactionKeys[dtrxKey] = true
		}

		db.Transactions.PutTrace(trxTraceKey, trxTrace)
		db.Transactions.PutBlockHeader(trxTraceKey, blk.Header)
		transactionKeys[trxTraceKey] = true
	}

	return transactionKeys, nil
}
