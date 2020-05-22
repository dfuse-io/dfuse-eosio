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

	"go.uber.org/zap"
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
			if *seenIrrMark && !inCanonicalChain(evi.BlockId) {
				// if you have seen IRR skip this event
				return true
			}

			if !evi.Irreversible && !inCanonicalChain(evi.BlockId) {
				// if you aren't an event from IRR and you aren't in the longest chian skip it
				return true
			}

			if evi.Irreversible {
				//if you are irr skip futeu
				*seenIrrMark = true
			}

			return false
		}

		switch ev := evi.Event.(type) {
		case *TransactionEvent_Addition:
			zlog.Debug("merging: addition event", zap.String("trx_id", evi.Id))
			if skip(&additionsIrr) {
				zlog.Debug("merging: addition event SKIPPING", zap.String("trx_id", evi.Id))
				continue
			}
			out.TransactionReceipt = ev.Addition.Receipt
			out.PublicKeys = ev.Addition.PublicKeys.PublicKeys
			out.Transaction = ev.Addition.Transaction

		case *TransactionEvent_InternalAddition:
			zlog.Debug("merging: internal addition event", zap.String("trx_id", evi.Id))
			if skip(&intAdditionsIrr) {
				zlog.Debug("merging: internal addition event SKIPPING", zap.String("trx_id", evi.Id))
				continue
			}
			out.Transaction = ev.InternalAddition.Transaction

		case *TransactionEvent_Execution:
			if ev.Execution.Trace.Receipt != nil {
				zlog.Debug("merging: execution event", zap.String("trx_id", evi.Id), zap.String("status", ev.Execution.Trace.Receipt.Status.String()))
			} else {
				zlog.Debug("merging: execution event", zap.String("trx_id", evi.Id))
			}

			if skip(&execIrr) {
				if ev.Execution.Trace.Receipt != nil {
					zlog.Debug("merging: execution event SKIPPING", zap.String("trx_id", evi.Id), zap.String("status", ev.Execution.Trace.Receipt.Status.String()))
				} else {
					zlog.Debug("merging: execution event SKIPPING", zap.String("trx_id", evi.Id))
				}

				continue
			}
			// In the case of a deferred transaction push (using CLI and `--delay-sec`)
			// it will have 2 execution traces, the first one when the delayed transaction got
			// pushed on the chain for later execution (that costs ram...) and the second
			// when the the transaction actually got executed. Thus we must merge the
			// RamOps, DbOps, DtrxOps, etc... to ensure that we have an accurate representation
			// of the execution trace
			mergedExectuionTrace := deepMergeTransactionTrace(out.ExecutionTrace, ev.Execution.Trace)
			out.ExecutionTrace = &mergedExectuionTrace

			out.ExecutionBlockHeader = ev.Execution.BlockHeader
			out.ExecutionIrreversible = evi.Irreversible

		case *TransactionEvent_DtrxScheduling:
			zlog.Debug("merging: dtrx scheduling event", zap.String("trx_id", evi.Id))
			if skip(&dtrxCreateIrr) {
				zlog.Debug("merging: dtrx scheduling event SKIPPING", zap.String("trx_id", evi.Id))
				continue
			}

			out.CreatedBy = ev.DtrxScheduling.CreatedBy
			out.Transaction = ev.DtrxScheduling.Transaction
			out.Transaction = ev.DtrxScheduling.Transaction
			out.CreationIrreversible = evi.Irreversible

		case *TransactionEvent_DtrxCancellation:
			zlog.Debug("merging: dtrx cancellation event", zap.String("trx_id", evi.Id))
			if skip(&dtrxCancelIrr) {
				zlog.Debug("merging: dtrx cancellation event SKIPPING", zap.String("trx_id", evi.Id))
				continue
			}

			if execIrr {
				zlog.Debug("merging: dtrx cancellation event SKIPPING BETA", zap.String("trx_id", evi.Id))
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

		if events[i].BlockNum == events[j].BlockNum {
			return events[i].Irreversible
		}
		return (events[i].BlockNum < events[j].BlockNum)
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

// the way this is use tells us that other can never be nil.
func deepMergeTransactionTrace(base, other *TransactionTrace) TransactionTrace {
	zlog.Debug("deep merging transaction traces",
		zap.String("other_trx_id", other.Id),
	)
	if base == nil {
		zlog.Debug("based not defined returning others")
		return *other

	}
	zlog.Debug("merging transaction traces",
		zap.String("base_trx_id", base.Id),
		zap.String("other_trx_id", other.Id),
	)
	trace := *base

	if trace.Receipt != nil &&
		other.Receipt != nil &&
		trace.Receipt.Status == TransactionStatus_TRANSACTIONSTATUS_DELAYED &&
		other.Receipt.Status != TransactionStatus_TRANSACTIONSTATUS_DELAYED {

		trace.Receipt.Status = other.Receipt.Status
	}

	trace.DbOps = append(base.DbOps, other.DbOps...)
	trace.DtrxOps = append(base.DtrxOps, other.DtrxOps...)
	trace.FeatureOps = append(base.FeatureOps, other.FeatureOps...)
	trace.PermOps = append(base.PermOps, other.PermOps...)
	trace.RamOps = append(base.RamOps, other.RamOps...)
	trace.RamCorrectionOps = append(base.RamCorrectionOps, other.RamCorrectionOps...)
	trace.RlimitOps = append(base.RlimitOps, other.RlimitOps...)
	trace.TableOps = append(base.TableOps, other.TableOps...)
	return trace
}
