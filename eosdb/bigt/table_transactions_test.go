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
	"testing"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/stretchr/testify/assert"
)

func alwaysInChain(blockID string) bool    { return true }
func alwaysOutOfChain(blockID string) bool { return false }

func TestTransactionStitch(t *testing.T) {
	tbl := &TransactionsTable{}

	/**
	 * Test Case Name Formatting
	 *
	 * Use flow of rows as the name with the following suffix on the action
	 *  - R: Reversible In-Chain
	 *  - S: Stale Reversible but Out-of-Chain
	 *  - I: Irreversible
	 *
	 * To denote a block link, use `_` and use `/` to denote a fork branch.
	 *
	 * Example:
	 *   Created(I)_Executed(R)/Created(S)_Executed(S)
	 *
	 * This tests a fork with two branches fork. First one having a block with
	 * a `Created` (a deferred) transaction which is irreversible followed
	 * by an `Executed` (execution of the deferred) that is reversible but
	 * considered in-chain for now. The second branch contains a `Created`
	 * that is staled and `Executed` that is also stale (so this branch is
	 * not considered part of the longuest chain of the platform).
	 */
	testCases := []struct {
		name             string
		rows             []*TransactionRow
		inCanonicalChain func(blockID string) bool
		expected         *pbeos.TransactionLifecycle
	}{
		{
			name:     "1_Empty",
			rows:     []*TransactionRow{},
			expected: nil,
		},
		{
			name: "2_Executed(I)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1", true),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "trx_1",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			name: "3_Executed(R)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "trx_1",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: false,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			name: "4_Executed(S)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1", false),
			},
			inCanonicalChain: alwaysOutOfChain,
			expected:         nil,
		},
		{
			name: "5_Executed(I)/Executed(R)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1a", true),
				executedTransactionRow(tbl, "trx_1", "1b", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "trx_1",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			name:             "6_Executed(I)/Executed(S)",
			inCanonicalChain: alwaysOutOfChain,
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1a", true),
				executedTransactionRow(tbl, "trx_1", "1b", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "trx_1",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			name: "7_Created(R)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1", "trx_2", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1",
				},
				ExecutionIrreversible: false,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED,
				PublicKeys:            []string{},
			},
		},
		{
			name: "8_Created(S)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1", "trx_2", false),
			},
			inCanonicalChain: alwaysOutOfChain,
			expected:         nil, //&mdl.TransactionResponse{},
		},
		{
			name: "9_Created(R)/Created(S)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1a", "trx_2", false),
				createdByTransactionRow(tbl, "trx_1", "1b", "trx_2", false),
			},
			inCanonicalChain: func(blockID string) bool { return blockID == "1a" },
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1a",
				},
				TransactionStatus: pbeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED,
				PublicKeys:        []string{},
			},
		},
		{
			name: "10_Created(I)_Canceled(I)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1", "trx_2", true),
				canceledByTransactionRow(tbl, "trx_3", "3", "trx_2", true),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1",
				},
				CanceledBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_3",
					BlockId:             "3",
				},
				CreationIrreversible:    true,
				CancelationIrreversible: true,
				TransactionStatus:       pbeos.TransactionStatus_TRANSACTIONSTATUS_CANCELED,
				PublicKeys:              []string{},
			},
		},
		{
			name: "11_Created(I)_Canceled(R)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1", "trx_2", true),
				canceledByTransactionRow(tbl, "trx_3", "3", "trx_2", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1",
				},
				CanceledBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_3",
					BlockId:             "3",
				},
				CreationIrreversible: true,
				TransactionStatus:    pbeos.TransactionStatus_TRANSACTIONSTATUS_CANCELED,
				PublicKeys:           []string{},
			},
		},
		{
			name: "12_Created(I)/Canceled(S)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1", "trx_2", true),
				canceledByTransactionRow(tbl, "trx_3", "3", "trx_2", false),
			},
			inCanonicalChain: alwaysOutOfChain,
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1",
				},
				CreationIrreversible: true,
				TransactionStatus:    pbeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED,
				PublicKeys:           []string{},
			},
		},
		{
			name: "13_Created(I)/Created(R)_Canceled(R)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1a", "trx_2", true),
				createdByTransactionRow(tbl, "trx_1", "1b", "trx_2", false),
				canceledByTransactionRow(tbl, "trx_3", "3a", "trx_2", false),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1a",
				},
				CanceledBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_3",
					BlockId:             "3a",
				},
				CreationIrreversible: true,
				TransactionStatus:    pbeos.TransactionStatus_TRANSACTIONSTATUS_CANCELED,
				PublicKeys:           []string{},
			},
		},
		{
			name: "14_Created(I)_Executed(I)/Canceled(R)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1a", "trx_2", true),
				{
					Key:         Keys.Transaction("11", "2a"),
					BlockHeader: &pbeos.BlockHeader{},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "2a",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
						},
						DtrxOps: []*pbeos.DTrxOp{},
						DbOps:   []*pbeos.DBOp{},
						RamOps:  []*pbeos.RAMOp{},
					},
					Irreversible: true,
					Written:      true,
				},
				canceledByTransactionRow(tbl, "trx_3", "3a", "trx_2", false),
			},
			inCanonicalChain: func(blockID string) bool { return blockID != "3a" },
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "2a",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1a",
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			// Created by a Smart Contract then , in different blocks
			name: "15_Created(R)_Executed(R)",
			rows: []*TransactionRow{
				{
					Key:         Keys.Transaction("11", "1a"),
					BlockHeader: &pbeos.BlockHeader{},
					Transaction: &pbeos.SignedTransaction{},
					CreatedBy: &pbeos.ExtDTrxOp{
						SourceTransactionId: "10",
						BlockId:             "1a",
					},
					Irreversible: false,
					Written:      true,
				},
				{
					Key:         Keys.Transaction("11", "2a"),
					BlockHeader: &pbeos.BlockHeader{},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "11",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
						},
						Elapsed:      20,
						DtrxOps:      []*pbeos.DTrxOp{},
						DbOps:        []*pbeos.DBOp{{ActionIndex: 1}},
						RamOps:       []*pbeos.RAMOp{{ActionIndex: 2}},
						TableOps:     []*pbeos.TableOp{{ActionIndex: 3}},
						CreationTree: []*pbeos.CreationFlatNode{{CreatorActionIndex: 0, ExecutionActionIndex: 0}},
					},
					Irreversible: false,
					Written:      true,
				},
			},
			inCanonicalChain: func(blockID string) bool { return true },
			expected: &pbeos.TransactionLifecycle{
				Id:          "11",
				Transaction: &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "11",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Elapsed:      20,
					DtrxOps:      []*pbeos.DTrxOp{},
					DbOps:        []*pbeos.DBOp{{ActionIndex: 1}},
					RamOps:       []*pbeos.RAMOp{{ActionIndex: 2}},
					TableOps:     []*pbeos.TableOp{{ActionIndex: 3}},
					CreationTree: []*pbeos.CreationFlatNode{{CreatorActionIndex: 0, ExecutionActionIndex: 0}},
				},
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "10",
					BlockId:             "1a",
				},
				ExecutionIrreversible: false,
				CreationIrreversible:  false,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			// Pushed to chain directly with a delay (I), then executed (R), in different blocks
			name: "16_CreatedDelayed(I)_Executed(R)",
			rows: []*TransactionRow{
				{
					Key:         Keys.Transaction("11", "1a"),
					BlockHeader: &pbeos.BlockHeader{},
					Transaction: &pbeos.SignedTransaction{},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "11",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED,
						},
						Elapsed: 10,
						DtrxOps: []*pbeos.DTrxOp{{
							Operation:     pbeos.DTrxOp_OPERATION_PUSH_CREATE,
							TransactionId: "11",
						}},
						RamOps: []*pbeos.RAMOp{{Operation: pbeos.RAMOp_OPERATION_CREATE_TABLE}},

						// Those are not actually possible in a real case, we put them here to ensure they do NOT cumulate with real execution ones
						DbOps:        []*pbeos.DBOp{{Operation: pbeos.DBOp_OPERATION_INSERT}},
						TableOps:     []*pbeos.TableOp{{Operation: pbeos.TableOp_OPERATION_INSERT}},
						CreationTree: []*pbeos.CreationFlatNode{{CreatorActionIndex: -1, ExecutionActionIndex: 0}},
					},
					CreatedBy: &pbeos.ExtDTrxOp{
						SourceTransactionId: "11",
						BlockId:             "1a",
					},
					Irreversible: true,
					Written:      true,
				},
				{
					Key:         Keys.Transaction("11", "2a"),
					BlockHeader: &pbeos.BlockHeader{},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "11",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
						},
						Elapsed:      20,
						DtrxOps:      []*pbeos.DTrxOp{},
						DbOps:        []*pbeos.DBOp{{ActionIndex: 1}},
						RamOps:       []*pbeos.RAMOp{{ActionIndex: 2}},
						TableOps:     []*pbeos.TableOp{{ActionIndex: 3}},
						CreationTree: []*pbeos.CreationFlatNode{{CreatorActionIndex: 0, ExecutionActionIndex: 0}},
					},
					Irreversible: false,
					Written:      true,
				},
			},
			inCanonicalChain: func(blockID string) bool { return true },
			expected: &pbeos.TransactionLifecycle{
				Id:          "11",
				Transaction: &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "11",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Elapsed:      20,
					DtrxOps:      []*pbeos.DTrxOp{},
					DbOps:        []*pbeos.DBOp{{ActionIndex: 1}},
					RamOps:       []*pbeos.RAMOp{{ActionIndex: 2}},
					TableOps:     []*pbeos.TableOp{{ActionIndex: 3}},
					CreationTree: []*pbeos.CreationFlatNode{{CreatorActionIndex: 0, ExecutionActionIndex: 0}},
				},
				// FIXME: how was this one different from the one in the above `ExecutionTrace`?
				//RamOps:               []*pbeos.RAMOp{{Operation: pbeos.RAMOp_OPERATION_CREATE_TABLE}, {ActionIndex: 2}},
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "11",
					BlockId:             "1a",
				},
				ExecutionIrreversible: false,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			// Pushed to chain directly with a delay (I), then executed (I), in same block
			name: "17_CreatedDelayed(I)_Executed(I)_SameBlock",
			rows: []*TransactionRow{
				{
					Key: Keys.Transaction("11", "1a"),
					BlockHeader: &pbeos.BlockHeader{
						Producer: "eoscanadadad",
					},
					Transaction: &pbeos.SignedTransaction{},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "11",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
						},
						Elapsed: 10,
						DtrxOps: []*pbeos.DTrxOp{
							{
								Operation:     pbeos.DTrxOp_OPERATION_PUSH_CREATE,
								TransactionId: "11",
							},
						},
						RamOps: []*pbeos.RAMOp{{ActionIndex: 1}},
					},
					CreatedBy: &pbeos.ExtDTrxOp{
						SourceTransactionId: "11",
						BlockId:             "1a",
					},
					Irreversible: true,
					Written:      true,
				},
			},
			inCanonicalChain: func(blockID string) bool { return true },
			expected: &pbeos.TransactionLifecycle{
				Id:          "11",
				Transaction: &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "11",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Elapsed: 10,
					DtrxOps: []*pbeos.DTrxOp{
						{
							Operation:     pbeos.DTrxOp_OPERATION_PUSH_CREATE,
							TransactionId: "11",
						},
					},
					RamOps: []*pbeos.RAMOp{{ActionIndex: 1}},
				},
				ExecutionBlockHeader: &pbeos.BlockHeader{
					Producer: "eoscanadadad",
				},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "11",
					BlockId:             "1a",
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			// Pushed to chain directly with a delay (I), then executed (I), in different blocks
			name: "18_CreatedDelayed(I)_Executed(I)_TwoBlocks",
			rows: []*TransactionRow{
				{
					Key: Keys.Transaction("11", "1a"),
					BlockHeader: &pbeos.BlockHeader{
						Producer: "eoscanadadad",
					},
					Transaction: &pbeos.SignedTransaction{
						Transaction: &pbeos.Transaction{
							Header: &pbeos.TransactionHeader{
								RefBlockNum: 1234,
							},
						},
					},
					CreatedBy: &pbeos.ExtDTrxOp{
						SourceTransactionId: "11",
						BlockId:             "1a",
					},
					Irreversible: true,
					Written:      true,
				},
				{
					Key:         Keys.Transaction("11", "2a"),
					Transaction: nil,
					BlockHeader: &pbeos.BlockHeader{
						Producer: "eoscanadacom",
					},
					TransactionTrace: &pbeos.TransactionTrace{
						Id: "11",
						Receipt: &pbeos.TransactionReceiptHeader{
							Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
						},
						Elapsed: 20,
						DtrxOps: []*pbeos.DTrxOp{},
						DbOps:   []*pbeos.DBOp{},
						RamOps:  []*pbeos.RAMOp{},
					},
					Irreversible: true,
					Written:      true,
				},
			},
			inCanonicalChain: func(blockID string) bool { return true },
			expected: &pbeos.TransactionLifecycle{
				Id: "11",
				Transaction: &pbeos.SignedTransaction{
					Transaction: &pbeos.Transaction{
						Header: &pbeos.TransactionHeader{
							RefBlockNum: 1234,
						},
					},
				},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id: "11",
					Receipt: &pbeos.TransactionReceiptHeader{
						Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Elapsed: 20,
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionBlockHeader: &pbeos.BlockHeader{
					Producer: "eoscanadacom",
				},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "11",
					BlockId:             "1a",
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				PublicKeys:            []string{},
			},
		},
		{
			name: "19_Executed(I) (Not Written Ignored)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_123", "1", true, writtenOption(false)),
			},
			expected: nil,
		},
		{
			name: "20_Created(I) (Not Written Ignored)",
			rows: []*TransactionRow{
				createdByTransactionRow(tbl, "trx_1", "1a", "trx_2", true),
				executedTransactionRow(tbl, "trx_2", "2a", true, writtenOption(false)),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:          "trx_2",
				Transaction: &pbeos.SignedTransaction{},
				CreatedBy: &pbeos.ExtDTrxOp{
					SourceTransactionId: "trx_1",
					BlockId:             "1a",
				},
				CreationIrreversible: true,
				TransactionStatus:    pbeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED,
				PublicKeys:           []string{},
			},
		},
		{
			name: "21_Executed(R) (No receipt)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1", false, removeTraceReceipt(true)),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id:      "trx_1",
					Receipt: nil,
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: false,
				CreationIrreversible:  false,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL,
				PublicKeys:            []string{},
			},
		},
		{
			name: "22_Executed(I) (No receipt)",
			rows: []*TransactionRow{
				executedTransactionRow(tbl, "trx_1", "1", true, removeTraceReceipt(true)),
			},
			expected: &pbeos.TransactionLifecycle{
				Id:                   "trx_1",
				ExecutionBlockHeader: &pbeos.BlockHeader{},
				Transaction:          &pbeos.SignedTransaction{},
				ExecutionTrace: &pbeos.TransactionTrace{
					Id:      "trx_1",
					Receipt: nil,
					DtrxOps: []*pbeos.DTrxOp{},
					DbOps:   []*pbeos.DBOp{},
					RamOps:  []*pbeos.RAMOp{},
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				TransactionStatus:     pbeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL,
				PublicKeys:            []string{},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.inCanonicalChain == nil {
				test.inCanonicalChain = alwaysInChain
			}

			response, _ := tbl.stitchTransaction(test.rows, test.inCanonicalChain)
			assert.Equal(t, test.expected, response)
		})
	}
}

func executedTransactionRow(
	tbl *TransactionsTable,
	trxID string,
	blockID string,
	irreversible bool,
	options ...interface{},
) *TransactionRow {
	return configureTransactionRow(&TransactionRow{
		Key:         Keys.Transaction(trxID, blockID),
		Transaction: &pbeos.SignedTransaction{},
		TransactionTrace: &pbeos.TransactionTrace{
			Id: trxID,
			Receipt: &pbeos.TransactionReceiptHeader{
				Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
			},
			DtrxOps: []*pbeos.DTrxOp{},
			DbOps:   []*pbeos.DBOp{},
			RamOps:  []*pbeos.RAMOp{},
		},
		BlockHeader:  &pbeos.BlockHeader{},
		Irreversible: irreversible,
		Written:      true,
	}, options)
}

func createdByTransactionRow(
	tbl *TransactionsTable,
	creatorID string,
	creatorBlockID string,
	trxID string,
	irreversible bool,
	options ...interface{},
) *TransactionRow {
	return configureTransactionRow(&TransactionRow{
		Key:          Keys.Transaction(trxID, creatorBlockID),
		Transaction:  &pbeos.SignedTransaction{},
		Irreversible: irreversible,
		CreatedBy: &pbeos.ExtDTrxOp{
			SourceTransactionId: creatorID,
			BlockId:             creatorBlockID,
		},
		Written: true,
	}, options)
}

func canceledByTransactionRow(
	tbl *TransactionsTable,
	cancelerID string,
	cancelerBlockID string,
	trxID string,
	irreversible bool,
	options ...interface{},
) *TransactionRow {
	return configureTransactionRow(&TransactionRow{
		Key:          Keys.Transaction(trxID, cancelerBlockID),
		Irreversible: irreversible,
		CanceledBy: &pbeos.ExtDTrxOp{
			SourceTransactionId: cancelerID,
			BlockId:             cancelerBlockID,
		},
		Written: true,
	}, options)
}

func configureTransactionRow(row *TransactionRow, options []interface{}) *TransactionRow {
	for _, option := range options {
		switch v := option.(type) {
		case writtenOption:
			row.Written = bool(v)

		case removeTraceReceipt:
			if v {
				row.TransactionTrace.Receipt = nil
			}
		}
	}

	return row
}

type writtenOption bool
type removeTraceReceipt bool
