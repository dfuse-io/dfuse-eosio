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
	"testing"

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
				{Id: "trx1", BlockId: "a", Irreversible: false, Event: NewTestAddEvent(1)},
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
				{Id: "trx1", BlockId: "e", Irreversible: false, Event: NewTestExecEvent(5)},
				{Id: "trx1", BlockId: "f", Irreversible: true, Event: NewTestExecEvent(6)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				t.Error("we said never call canonicalChain!")
				return true
			},
			expect: &TransactionLifecycle{
				Id:                    "trx1",
				TransactionStatus:     TransactionStatus_TRANSACTIONSTATUS_HARDFAIL, // no receipt, ignore
				TransactionReceipt:    &TransactionReceipt{Index: 3},
				ExecutionTrace:        &TransactionTrace{Index: 6},
				ExecutionIrreversible: true,
			},
		},
		{
			name: "multiple, select one of each, ignore dtrx cancels if execution irreversible",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "a", Irreversible: false, Event: NewTestDtrxCreateEvent("1")},
				{Id: "trx1", BlockId: "b", Irreversible: true, Event: NewTestDtrxCreateEvent("2")},
				{Id: "trx1", BlockId: "c", Irreversible: false, Event: NewTestDtrxCreateEvent("3")},

				{Id: "trx1", BlockId: "d", Irreversible: false, Event: NewTestExecEvent(4)},
				{Id: "trx1", BlockId: "e", Irreversible: false, Event: NewTestExecEvent(5)},
				{Id: "trx1", BlockId: "f", Irreversible: true, Event: NewTestExecEvent(6)},

				{Id: "trx1", BlockId: "call1", Irreversible: false, Event: NewTestDtrxCancelEvent("1")},
				{Id: "trx1", BlockId: "call2", Irreversible: false, Event: NewTestDtrxCancelEvent("2")},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				if id == "call1" || id == "call2" {
					return true
				}
				t.Error("don't call canonicalChain otherwise")
				return true
			},
			expect: &TransactionLifecycle{
				Id:                    "trx1",
				TransactionStatus:     TransactionStatus_TRANSACTIONSTATUS_HARDFAIL, // no receipt, ignore
				ExecutionTrace:        &TransactionTrace{Index: 6},
				ExecutionIrreversible: true,
				CreationIrreversible:  true,
				CreatedBy:             &ExtDTrxOp{SourceTransactionId: "2"}},
		},
		{
			name: "cancellation arrives before irreversible execution, should not show cancelled at all",
			events: []*TransactionEvent{
				{Id: "trx1", BlockId: "d", Irreversible: false, Event: NewTestDtrxCancelEvent("1")},
				{Id: "trx1", BlockId: "f", Irreversible: true, Event: NewTestExecEvent(6)},
			},
			canonicalChain: func(t *testing.T, id string) bool {
				return true
			},
			expect: &TransactionLifecycle{
				Id:                    "trx1",
				TransactionStatus:     TransactionStatus_TRANSACTIONSTATUS_HARDFAIL, // no receipt, ignore
				ExecutionTrace:        &TransactionTrace{Index: 6},
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
