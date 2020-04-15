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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func TrashTestReconstructBlock(t *testing.T) {
	// Get block 102660000 and following
	db, err := NewDriver("chain", "dev", "dev", false, time.Second, 10)
	require.NoError(t, err)

	// Missing: 102660094
	blocks, err := db.GetBlockByNum(context.Background(), 102660000)
	require.NoError(t, err)

	require.Equal(t, 1, len(blocks))
	block := blocks[0]

	for _, trx := range block.TransactionRefs.Hashes {
		trxID := hex.EncodeToString(trx)
		row, err := db.GetTransactionRow(context.Background(), trxID)
		require.NoError(t, err)

		if !bytes.Equal(row.BlockHeader.TransactionMroot, block.Block.Header.TransactionMroot) {
			require.NoError(t, fmt.Errorf("failed, found row with prev block id %s while looking for previous block id %s", row.BlockHeader.Previous, block.Block.Header.Previous))
		}

		signed := codec.SignedTransactionToEOS(row.Transaction)
		packed, err := signed.Pack(eos.CompressionNone)
		require.NoError(t, err)

		_ = packed
		// packed, err = pbeosPackedTransactionToDEOS(packed)
		// require.NoError(t, err)

		// block.Block.Transactions = append(block.Block.Transactions, packed)
	}

	for _, trace := range block.TransactionTraceRefs.Hashes {
		trxID := hex.EncodeToString(trace)
		row, err := db.GetTransactionRow(context.Background(), trxID)
		require.NoError(t, err)

		if !bytes.Equal(row.BlockHeader.TransactionMroot, block.Block.Header.TransactionMroot) {
			require.NoError(t, fmt.Errorf("failed, found row with prev block id %s while looking for previous block id %s", row.BlockHeader.Previous, block.Block.Header.Previous))
		}

		block.Block.TransactionTraces = append(block.Block.TransactionTraces, row.TransactionTrace)
	}

	// Fetch all TransactionRefs
	// Fetch all TranscationTraces
}
