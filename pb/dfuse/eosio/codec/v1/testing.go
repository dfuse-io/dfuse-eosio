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

func NewTestAddEvent(idx int) *TransactionEvent_Addition {
	return &TransactionEvent_Addition{Addition: &TransactionEvent_Added{
		Receipt:    &TransactionReceipt{Index: uint64(idx)},
		PublicKeys: &PublicKeys{},
	}}
}

func NewTestIntAddEvent(prefix int) *TransactionEvent_InternalAddition {
	return &TransactionEvent_InternalAddition{InternalAddition: &TransactionEvent_AddedInternally{
		Transaction: &SignedTransaction{
			Transaction: &Transaction{
				Header: &TransactionHeader{
					RefBlockPrefix: uint32(prefix),
				},
			},
		},
	}}
}

func NewSimpleTestExecEvent(idx int) *TransactionEvent_Execution {
	return &TransactionEvent_Execution{Execution: &TransactionEvent_Executed{
		Trace: &TransactionTrace{
			Index: uint64(idx),
		},
	}}
}

func NewTestExecEvent(idx int) *TransactionEvent_Execution {
	return &TransactionEvent_Execution{Execution: &TransactionEvent_Executed{
		Trace: &TransactionTrace{
			Receipt: &TransactionReceiptHeader{
				Status: TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
			},
			Index: uint64(idx),
		},
	}}
}

func NewTestDtrxCreateEvent(src string) *TransactionEvent_DtrxScheduling {
	return &TransactionEvent_DtrxScheduling{DtrxScheduling: &TransactionEvent_DtrxScheduled{
		CreatedBy: &ExtDTrxOp{
			SourceTransactionId: src,
		},
	}}
}

func NewTestDtrxCancelEvent(src string) *TransactionEvent_DtrxCancellation {
	return &TransactionEvent_DtrxCancellation{DtrxCancellation: &TransactionEvent_DtrxCanceled{
		CanceledBy: &ExtDTrxOp{
			SourceTransactionId: src,
		},
	}}
}
