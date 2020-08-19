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

package pbcodec

import (
	"fmt"
	"os"
	"testing"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	evs := []*TransactionEvent{
		&TransactionEvent{Id: "trx1", Irreversible: false},
		&TransactionEvent{Id: "trx2", Irreversible: true},
		&TransactionEvent{Id: "trx3", Irreversible: false},
		&TransactionEvent{Id: "trx4", Irreversible: true},
		&TransactionEvent{Id: "trx5", Irreversible: false},
		&TransactionEvent{Id: "trx6", Irreversible: true},
		&TransactionEvent{Id: "trx7", Irreversible: true},
	}

	evs = sortEvents(evs)

	assert.True(t, evs[0].Irreversible)
	assert.True(t, evs[1].Irreversible)
	assert.True(t, evs[2].Irreversible)
	assert.True(t, evs[3].Irreversible)
	assert.False(t, evs[4].Irreversible)
	assert.False(t, evs[5].Irreversible)
	assert.False(t, evs[6].Irreversible)
}

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func TestMergeTransactionEvents(t *testing.T) {

	tests := []struct {
		name           string
		events         []*TransactionEvent
		canonicalChain func(t *testing.T, id string) bool
		expect         *TransactionLifecycle
	}{
		{
			name: "single, irreversible event",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "abc", Irreversible: true, Event: NewTestAddEvent(1)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				fmt.Println("CHECKED", id)
				return true
			},
			expect: &TransactionLifecycle{
				Id:                 "trx1",
				TransactionReceipt: &TransactionReceipt{Index: 1},
			},
		},
		{
			name: "two additions, none irr, check canonical chain",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "a", Irreversible: false, Event: NewTestAddEvent(1)}, // skip this one since it isn't IRR and is NOT in the longest chain
				{Id: "trx1", BlockId: "b", Irreversible: false, Event: NewTestAddEvent(2)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				return id == "b"
			},
			expect: &TransactionLifecycle{
				Id:                 "trx1",
				TransactionReceipt: &TransactionReceipt{Index: 2},
			},
		},
		{
			name: "multiple, select the irr of each kind, never call canonical chain",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "a", Irreversible: false, Event: NewTestAddEvent(1)},
				{Id: "trx1", BlockId: "b", Irreversible: false, Event: NewTestAddEvent(2)},
				{Id: "trx1", BlockId: "c", Irreversible: true, Event: NewTestAddEvent(3)},

				{Id: "trx1", BlockId: "d", Irreversible: false, Event: NewTestExecEvent(4)},
				{Id: "trx1", BlockId: "c", Irreversible: true, Event: NewTestExecEvent(6)},
				{Id: "trx1", BlockId: "e", Irreversible: false, Event: NewTestExecEvent(5)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				t.Error("we said never call canonicalChain!")
				return true
			},
			expect: &TransactionLifecycle{
				Id:                 "trx1",
				TransactionStatus:  TransactionStatus_TRANSACTIONSTATUS_EXECUTED, // no receipt, ignore
				TransactionReceipt: &TransactionReceipt{Index: 3},
				ExecutionTrace: &TransactionTrace{
					Receipt: &TransactionReceiptHeader{
						Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Index: 6,
				},
				ExecutionIrreversible: true,
			},
		},
		{
			name: "multiple, select one of each, ignore dtrx cancels if execution irreversible",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "2a", Irreversible: false, Event: NewTestDtrxCreateEvent("1")},
				{Id: "trx1", BlockId: "2b", Irreversible: true, Event: NewTestDtrxCreateEvent("2")},
				{Id: "trx1", BlockId: "2c", Irreversible: false, Event: NewTestDtrxCreateEvent("3")},

				// delay trx thus execution could be at a later block
				{Id: "trx1", BlockId: "4a", Irreversible: false, Event: NewTestExecEvent(4)},
				{Id: "trx1", BlockId: "4b", Irreversible: true, Event: NewTestExecEvent(5)},
				{Id: "trx1", BlockId: "4c", Irreversible: false, Event: NewTestExecEvent(6)},

				{Id: "trx1", BlockId: "5b", Irreversible: false, Event: NewTestDtrxCancelEvent("1")},
				{Id: "trx1", BlockId: "5d", Irreversible: false, Event: NewTestDtrxCancelEvent("2")},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				if id == "5b" {
					return true
				}
				return false
			},
			expect: &TransactionLifecycle{
				Id:                "trx1",
				TransactionStatus: TransactionStatus_TRANSACTIONSTATUS_EXECUTED, // no receipt, ignore
				ExecutionTrace: &TransactionTrace{
					Receipt: &TransactionReceiptHeader{
						Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Index: 5,
				},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				CreatedBy:             &ExtDTrxOp{SourceTransactionId: "2"}},
		},
		{
			name: "cancellation arrives before irreversible execution, should not show cancelled at all",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "d", BlockNum: 4, Irreversible: false, Event: NewTestDtrxCancelEvent("1")},
				{Id: "trx1", BlockId: "f", BlockNum: 5, Irreversible: true, Event: NewTestExecEvent(6)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				return true
			},
			expect: &TransactionLifecycle{
				Id:                "trx1",
				TransactionStatus: TransactionStatus_TRANSACTIONSTATUS_EXECUTED, // no receipt, ignore
				ExecutionTrace: &TransactionTrace{
					Receipt: &TransactionReceiptHeader{
						Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Index: 6,
				},
				ExecutionIrreversible: true,
			},
		},
		{
			name: "dev1: deferred transaction push, has multiple execution traces, execution succeeded",
			events: []*TransactionEvent{
				{
					Id:           "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					BlockId:      "0000002ca98c204ec8f93e91332baf2a0eea5edd4e9f262bfb3e9b997d6f2415",
					Irreversible: true,
					Event: &TransactionEvent_Addition{Addition: &TransactionEvent_Added{
						Receipt: &TransactionReceipt{Index: uint64(1)},
						Transaction: &SignedTransaction{
							Transaction: &Transaction{
								Actions: []*Action{
									{
										Account:  "eosio.token",
										Name:     "transfer",
										JsonData: "",
									},
								},
							},
						},
						PublicKeys: &PublicKeys{
							PublicKeys: []string{"EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP"},
						},
					}},
				},
				{
					Id:           "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					BlockId:      "0000002ca98c204ec8f93e91332baf2a0eea5edd4e9f262bfb3e9b997d6f2415",
					Irreversible: true,
					Event: &TransactionEvent_Execution{Execution: &TransactionEvent_Executed{
						BlockHeader: &BlockHeader{
							ActionMroot: []byte("1"),
						},
						Trace: &TransactionTrace{
							Receipt: &TransactionReceiptHeader{
								Status: TransactionStatus_TRANSACTIONSTATUS_DELAYED,
							},
							Id:    "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
							Index: uint64(2),
							RamOps: []*RAMOp{
								{
									Operation:   RAMOp_OPERATION_DEFERRED_TRX_PUSHED,
									ActionIndex: 0,
									Payer:       "eosio",
									Delta:       371,
									Usage:       1182644,
									Namespace:   RAMOp_NAMESPACE_DEFERRED_TRX,
									UniqueKey:   "9",
									Action:      RAMOp_ACTION_PUSH,
								},
							},
						},
					}},
				},
				{
					Id:           "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					BlockId:      "0000002ca98c204ec8f93e91332baf2a0eea5edd4e9f262bfb3e9b997d6f2415",
					Irreversible: true,
					Event: &TransactionEvent_DtrxScheduling{DtrxScheduling: &TransactionEvent_DtrxScheduled{
						Transaction: &SignedTransaction{
							Transaction: &Transaction{
								Actions: []*Action{
									{
										Account:  "eosio.token",
										Name:     "transfer",
										JsonData: "{\"from\":\"eosio\",\"to\":\"battlefield1\",\"quantity\":\"1.0000 EOS\",\"memo\":\"push delayed trx\"}",
									},
								},
							},
						},
						CreatedBy: &ExtDTrxOp{
							SourceTransactionId: "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
							DtrxOp: &DTrxOp{
								Operation: DTrxOp_OPERATION_PUSH_CREATE,
							},
						},
					}},
				},
				{
					Id:           "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					BlockId:      "0000002e7936c0363549d9f1bfc737e75a42a02d9e6cf72437e347f36580cfd8",
					Irreversible: true,
					Event: &TransactionEvent_Execution{Execution: &TransactionEvent_Executed{
						BlockHeader: &BlockHeader{
							ActionMroot: []byte("2"),
						},
						Trace: &TransactionTrace{
							Receipt: &TransactionReceiptHeader{
								Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
							},
							Id:    "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
							Index: uint64(1),
							RamOps: []*RAMOp{
								{
									Operation:   RAMOp_OPERATION_DEFERRED_TRX_REMOVED,
									ActionIndex: 0,
									Payer:       "eosio",
									Delta:       -371,
									Usage:       1182273,
									Namespace:   RAMOp_NAMESPACE_DEFERRED_TRX,
									UniqueKey:   "9",
									Action:      RAMOp_ACTION_REMOVE,
								},
							},
						},
					}},
				},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				return true
			},
			expect: &TransactionLifecycle{
				Id:                 "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
				TransactionStatus:  TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				TransactionReceipt: &TransactionReceipt{Index: uint64(1)},
				PublicKeys:         []string{"EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP"},
				Transaction: &SignedTransaction{
					Transaction: &Transaction{
						Actions: []*Action{
							{
								Account:  "eosio.token",
								Name:     "transfer",
								JsonData: "{\"from\":\"eosio\",\"to\":\"battlefield1\",\"quantity\":\"1.0000 EOS\",\"memo\":\"push delayed trx\"}",
							},
						},
					},
				},
				CreatedBy: &ExtDTrxOp{
					SourceTransactionId: "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					DtrxOp: &DTrxOp{
						Operation: DTrxOp_OPERATION_PUSH_CREATE,
					},
				},
				ExecutionBlockHeader: &BlockHeader{
					ActionMroot: []byte("2"),
				},
				ExecutionTrace: &TransactionTrace{
					Receipt: &TransactionReceiptHeader{
						Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
					},
					Id:    "480a4adde14100097abec586d1dec805b3bfdb48c9efed5695ca02c61ea043bd",
					Index: 1,
					RamOps: []*RAMOp{
						{
							Operation:   RAMOp_OPERATION_DEFERRED_TRX_PUSHED,
							ActionIndex: 0,
							Payer:       "eosio",
							Delta:       371,
							Usage:       1182644,
							Namespace:   RAMOp_NAMESPACE_DEFERRED_TRX,
							UniqueKey:   "9",
							Action:      RAMOp_ACTION_PUSH,
						},
						{
							Operation:   RAMOp_OPERATION_DEFERRED_TRX_REMOVED,
							ActionIndex: 0,
							Payer:       "eosio",
							Delta:       -371,
							Usage:       1182273,
							Namespace:   RAMOp_NAMESPACE_DEFERRED_TRX,
							UniqueKey:   "9",
							Action:      RAMOp_ACTION_REMOVE,
						},
					},
				},
				CreationIrreversible:  true,
				ExecutionIrreversible: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := MergeTransactionEvents(test.events, func(id string) bool { return test.canonicalChain(t, id) })
			assert.Equal(t, test.expect, res)
		})
	}
}

func Test_deepMergeTransactionTrace(t *testing.T) {
	a := &TransactionTrace{
		Id:        "trx1",
		RamOps:    []*RAMOp{{Operation: RAMOp_OPERATION_DEFERRED_TRX_REMOVED}},
		RlimitOps: []*RlimitOp{{Operation: RlimitOp_OPERATION_INSERT}},
	}
	b := &TransactionTrace{
		Id:        "trx1",
		RamOps:    []*RAMOp{{Operation: RAMOp_OPERATION_DEFERRED_TRX_PUSHED}},
		RlimitOps: []*RlimitOp{{Operation: RlimitOp_OPERATION_UPDATE}},
	}
	r1 := mergeTransactionTrace(a, b)
	assert.Equal(t, []*RAMOp{{Operation: RAMOp_OPERATION_DEFERRED_TRX_PUSHED}, {Operation: RAMOp_OPERATION_DEFERRED_TRX_REMOVED}}, r1.RamOps)
	assert.Equal(t, []*RlimitOp{{Operation: RlimitOp_OPERATION_UPDATE}, {Operation: RlimitOp_OPERATION_INSERT}}, r1.RlimitOps)

	c := &TransactionTrace{
		Id:        "trx1",
		RamOps:    []*RAMOp{{Operation: RAMOp_OPERATION_DEFERRED_TRX_REMOVED}},
		RlimitOps: []*RlimitOp{{Operation: RlimitOp_OPERATION_INSERT}},
	}

	d := &TransactionTrace{
		Id: "trx1",
	}
	r2 := mergeTransactionTrace(c, d)
	assert.Equal(t, []*RAMOp{{Operation: RAMOp_OPERATION_DEFERRED_TRX_REMOVED}}, r2.RamOps)
	assert.Equal(t, []*RlimitOp{{Operation: RlimitOp_OPERATION_INSERT}}, r2.RlimitOps)

}
