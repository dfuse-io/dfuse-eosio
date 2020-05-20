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
	"sort"
)

func MergeTransactionEvents(events []*TransactionEvent, inCanonicalChain func(blockID string) bool) *TransactionLifecycle {
	if len(events) == 0 {
		return nil
	}

	sortEvents(events)

	out := &TransactionLifecycle{}

	var additionsIrr, intAdditionsIrr, execIrr, dtrxCreateIrr, dtrxCancelIrr bool
	var trxID string
	for _, evi := range events {
		if trxID == "" {
			trxID = evi.Id
		} else {
			if trxID != evi.Id {
				panic(fmt.Errorf("transaction events passed to MergeTransactionEvents are not all from the same transaction id %q and %q", trxID, evi.Id))
			}
		}

		skip := func(seenIrrMark *bool) bool {
			if *seenIrrMark {
				return true
			}

			if !evi.Irreversible && !inCanonicalChain(evi.BlockId) {
				return true
			}

			if evi.Irreversible {
				*seenIrrMark = true
			}

			return false
		}

		switch ev := evi.Event.(type) {
		case *TransactionEvent_Addition:
			if skip(&additionsIrr) {
				continue
			}

			out.TransactionReceipt = ev.Addition.Receipt
			out.Transaction = ev.Addition.Transaction
			out.PublicKeys = ev.Addition.PublicKeys.PublicKeys

		case *TransactionEvent_InternalAddition:
			if skip(&intAdditionsIrr) {
				continue
			}

			out.Transaction = ev.InternalAddition.Transaction

		case *TransactionEvent_Execution:
			if skip(&execIrr) {
				continue
			}

			out.ExecutionTrace = ev.Execution.Trace
			out.ExecutionBlockHeader = ev.Execution.BlockHeader
			out.ExecutionIrreversible = evi.Irreversible

		case *TransactionEvent_DtrxScheduling:
			if skip(&dtrxCreateIrr) {
				continue
			}

			out.CreatedBy = ev.DtrxScheduling.CreatedBy
			out.Transaction = ev.DtrxScheduling.Transaction
			out.CreationIrreversible = evi.Irreversible

		case *TransactionEvent_DtrxCancellation:
			if skip(&dtrxCancelIrr) {
				continue
			}

			if execIrr {
				continue
			}

			out.CanceledBy = ev.DtrxCancellation.CanceledBy
			out.CancelationIrreversible = evi.Irreversible

		default:
			panic("what's that type anyway?")
		}
	}

	out.Id = trxID
	out.TransactionStatus = getTransactionLifeCycleStatus(out)
	// TODO: REplace by a function call on `TransactionLifecycle` to get it..
	// response.TransactionStatus = getTransactionLifeCycleStatus(response)

	// FIXME: previous implementation returned `nil, nil` when in the
	// end, there were no TransactionRow that passed the in-chain
	// tests.
	// * Is that what we want? Is it okay to do it this way now?
	//   We could simply check that we arrived at the `switch` statement
	//   at least once, if not, we'd return `nil`

	return out
}

func sortEvents(events []*TransactionEvent) []*TransactionEvent {
	sort.Slice(events, func(i, j int) bool {
		// TEST that this does INDEED sort
		return events[i].Irreversible
	})
	return events
}

// This should replace, or we assign it at the end inside the Lifecycle.TransactionStatus
func getTransactionLifeCycleStatus(lifeCycle *TransactionLifecycle) TransactionStatus {
	// FIXME: this function belongs to the sample place as the stitcher, probably in `pbcodec`
	// alongside the rest.
	if lifeCycle.CanceledBy != nil {
		return TransactionStatus_TRANSACTIONSTATUS_CANCELED
	}

	if lifeCycle.ExecutionTrace == nil {
		if lifeCycle.CreatedBy != nil {
			return TransactionStatus_TRANSACTIONSTATUS_DELAYED
		}

		// FIXME: It was `pending` before but not present anymore, what should we do?
		return TransactionStatus_TRANSACTIONSTATUS_NONE
	}

	if lifeCycle.ExecutionTrace.Receipt == nil {
		// That happen strangely on EOS Kylin where `eosio:onblock` started to fail and exhibit no Receipt
		return TransactionStatus_TRANSACTIONSTATUS_HARDFAIL
	}

	// Expired Failed Executed
	return lifeCycle.ExecutionTrace.Receipt.Status
}
