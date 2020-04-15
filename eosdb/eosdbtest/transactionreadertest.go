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

package eosdbtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var transactionReaderTests = []struct {
	name string
	test func(t *testing.T, driverFactory DriverFactory)
}{
	{"TestGetTransactionTraces", TestGetTransactionTraces},
	{"TestGetTransactionTracesBatch", TestGetTransactionTracesBatch},
	{"TestGetTransactionEvents", TestGetTransactionEvents},
	{"TestGetTransactionEventsBatch", TestGetTransactionEventsBatch},
	{"TestReadTransactions", TestReadTransactions},
}

func TestAllTransactionsReader(t *testing.T, driverName string, driverFactory DriverFactory) {
	for _, rt := range transactionReaderTests {
		t.Run(driverName+"/"+rt.name, func(t *testing.T) {
			rt.test(t, driverFactory)
		})
	}
}

func TestReadTransactions(t *testing.T, driverFactory DriverFactory) {
	db, clean := driverFactory()
	defer clean()

	ctx := context.Background()
	in := testBlock1()

	require.NoError(t, db.PutBlock(ctx, in))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, in))
	require.NoError(t, db.Flush(ctx))

	// Block data
	evs, err := db.GetTransactionEvents(context.Background(), "00112233aa")
	require.NoError(t, err)
	assert.Len(t, evs, 2)
	var additions, executions int

	for _, ev := range evs {
		switch evt := ev.Event.(type) {
		case *pbeos.TransactionEvent_Addition:
			additions++
			assert.Equal(t, "00112233aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ev.Id)
			assert.Equal(t, "00000002aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ev.BlockId)
			assert.True(t, ev.Irreversible)

			if evt.Addition.Receipt != nil {
				// FIXME: this should be skipped ONLY for the old `bigt` implementation, which didn't save
				// the Receipt..
				assert.Equal(t, 32, int(evt.Addition.Receipt.NetUsageWords))
				assert.Equal(t, 32, int(evt.Addition.Receipt.CpuUsageMicroSeconds))
			}
			assert.Equal(t, []string{"SIG_K1_K7kTcvsznS2pSQ2unjW9nduqHieWnc5B6rFdbVif4RM1DCTVhQUpzwng3XTGewDhVZqNvqSAEwHgB8yBnfDYAHquRX4fBo"}, evt.Addition.Transaction.Signatures)
			assert.Len(t, evt.Addition.Transaction.Transaction.Actions, 1)
			assert.Equal(t, "name", evt.Addition.Transaction.Transaction.Actions[0].Name)
			assert.Equal(t, []string{"EOS7T3GcBYpYf2D63HGDG7qB9TiD56XT4m1hAQfkHWuV9LhMoQ1ZY"}, evt.Addition.PublicKeys.PublicKeys)

		case *pbeos.TransactionEvent_Execution:
			executions++
			assert.Equal(t, "00112233aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ev.Id)
			assert.Equal(t, "00000002aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ev.BlockId)
			assert.True(t, ev.Irreversible)

			assert.Equal(t, "00000001aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", evt.Execution.BlockHeader.Previous)
			assert.Equal(t, "tester", evt.Execution.BlockHeader.Producer)
			assert.Len(t, evt.Execution.Trace.DtrxOps, 2)
			assert.Equal(t, "aaa888aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", evt.Execution.Trace.DtrxOps[1].TransactionId)
			assert.Equal(t, "00112233aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", evt.Execution.Trace.Id)

		default:
			t.Error(fmt.Sprintf("unexpected type %T", ev))
		}
	}

	assert.Equal(t, 1, additions)
	assert.Equal(t, 1, executions)
}

func TestGetTransactionTraces(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name         string
		trxIDs       []string
		trxIDPrefix  string
		expectTrxIDs []string
		expectErr    error
	}{
		{
			name:         "sunny path",
			trxIDs:       []string{"a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "a2bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6b"},
			trxIDPrefix:  "a1",
			expectTrxIDs: []string{"a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6b"},
		},
		{
			name:         "only match prefix",
			trxIDs:       []string{"a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "a2bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6b"},
			trxIDPrefix:  "a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a",
			expectTrxIDs: []string{"a1bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, trxID := range test.trxIDs {
				putTransaction(t, db, trxID)
			}

			events, err := db.GetTransactionTraces(ctx, test.trxIDPrefix)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				ids := []string{}
				for _, event := range events {
					ids = append(ids, event.Id)
				}
				assert.ElementsMatch(t, test.expectTrxIDs, ids)
			}
		})
	}
}

func TestGetTransactionTracesBatch(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name         string
		trxIDs       []string
		trxIdsPrefix []string
		expectTrxIDs [][]string
		expectErr    error
	}{
		{

			name:         "sunny path",
			trxIDs:       []string{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1addffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eaaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eccffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "3ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
			trxIdsPrefix: []string{"1a", "2e"},
			expectTrxIDs: [][]string{{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1addffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}, {"2eaaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eccffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, trxID := range test.trxIDs {
				putTransaction(t, db, trxID)
			}

			events, err := db.GetTransactionTracesBatch(ctx, test.trxIdsPrefix)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				eventIds := [][]string{}
				for _, trxs := range events {
					ids := []string{}
					for _, event := range trxs {
						ids = append(ids, event.Id)
					}
					eventIds = append(eventIds, ids)
				}
				assert.ElementsMatch(t, test.expectTrxIDs, eventIds)
			}
		})
	}
}

func TestGetTransactionEvents(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name         string
		trxIDs       []string
		trxIDPrefix  string
		expectTrxIDs []string
		expectErr    error
	}{
		{
			name:         "sunny path",
			trxIDs:       []string{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1eddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
			trxIDPrefix:  "1a",
			expectTrxIDs: []string{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		},
		{
			name:         "only match prefix",
			trxIDs:       []string{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1eddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
			trxIDPrefix:  "1e",
			expectTrxIDs: []string{"1eddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1eddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, trxID := range test.trxIDs {
				putTransaction(t, db, trxID)
			}

			// TODO: the `GetTransactionEvents()` function should be
			// exercised with all the types of events. So fixtures
			// should write an implicit trx, two addition events
			// (internal and normal), a dtrx event and an execution
			// trace event.

			events, err := db.GetTransactionEvents(ctx, test.trxIDPrefix)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				ids := []string{}
				for _, event := range events {
					ids = append(ids, event.Id)
				}
				assert.ElementsMatch(t, test.expectTrxIDs, ids)
			}
		})
	}
}

func TestGetTransactionEventsBatch(t *testing.T, driverFactory DriverFactory) {
	t.Skip()
	tests := []struct {
		name         string
		trxIDs       []string
		trxIdsPrefix []string
		expectTrxIDs [][]string
		expectErr    error
	}{
		{
			name:         "sunny path",
			trxIDs:       []string{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1addffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eaaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eccffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "3ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
			trxIdsPrefix: []string{"1a", "2e"},
			expectTrxIDs: [][]string{{"1abbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1accffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1addffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}, {"2eaaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2ebbffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2eccffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, trxID := range test.trxIDs {
				putTransaction(t, db, trxID)
			}

			events, err := db.GetTransactionEventsBatch(ctx, test.trxIdsPrefix)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				eventIds := [][]string{}
				for _, trxs := range events {
					ids := []string{}
					for _, event := range trxs {
						ids = append(ids, event.Id)
					}
					eventIds = append(eventIds, ids)
				}
				assert.ElementsMatch(t, test.expectTrxIDs, eventIds)
			}
		})
	}
}

func putTransaction(t *testing.T, db eosdb.Driver, trxID string) {
	//it is important to use full length id for transaction
	blk := TestBlock(t, "06bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a", "06bc5790ef36d5779e2a0a849a11c09c999b5dc564afce6920e20b07af1f4b6a")
	// FIXME: when we create transaction, this code only creates *one
	// type* of transactions, where a lot of the `GetTransaction*`
	// code fetch some transactions of different type: deferred (like
	// this one), but also implicit, normal transactions and
	// transaction traces.
	blk.TransactionTraces = append(blk.TransactionTraces, &pbeos.TransactionTrace{
		Id:       trxID,
		BlockNum: 2,
		DtrxOps: []*pbeos.DTrxOp{
			{
				TransactionId: trxID, // FIXME: a dtrx that is created actually has a *different* transaction ID from the one creating it.
				Operation:     pbeos.DTrxOp_OPERATION_CREATE,
				ActionIndex:   0,
				Payer:         "eoscanada1",
				Transaction: &pbeos.SignedTransaction{
					Transaction:     nil,
					Signatures:      []string{"signature"},
					ContextFreeData: nil,
				},
			},
		},
	})
	ctx := context.Background()
	require.NoError(t, db.PutBlock(ctx, blk))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
	require.NoError(t, db.Flush(ctx))
}
