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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/golang/protobuf/ptypes"
	"github.com/tidwall/gjson"
)

// TODO: We should probably memoize all fields that requires computation
//       like Time() and likes.

func (b *Block) ID() string {
	return b.Id
}

func (b *Block) Num() uint64 {
	return uint64(b.Number)
}

func (b *Block) PreviousID() string {
	return b.Header.Previous
}

func (b *Block) Time() (time.Time, error) {
	timestamp, err := ptypes.Timestamp(b.Header.Timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to turn google proto Timestamp into time.Time: %s", err)
	}

	return timestamp, nil
}

func (b *Block) MustTime() time.Time {
	timestamp, err := b.Time()
	if err != nil {
		panic(err)
	}

	return timestamp
}

func (b *Block) LIBNum() uint64 {
	return uint64(b.DposIrreversibleBlocknum)
}

func (b *Block) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.Id, uint64(b.Number))
}

func (b *Block) Transactions() []*TransactionReceipt {
	if b.FilteringApplied {
		return b.FilteredTransactions
	}

	return b.UnfilteredTransactions
}

func (b *Block) TransactionTraces() []*TransactionTrace {
	if b.FilteringApplied {
		return b.FilteredTransactionTraces
	}

	return b.UnfilteredTransactionTraces
}

func (b *Block) ImplicitTransactionOps() []*TrxOp {
	if b.FilteringApplied {
		return b.FilteredImplicitTransactionOps
	}

	return b.UnfilteredImplicitTransactionOps
}

func (b *Block) CanceledDTrxIDs() (out []string) {
	seen := make(map[string]bool)
	for _, trx := range b.TransactionTraces() {
		for _, dtrxOp := range trx.DtrxOps {
			if dtrxOp.IsCancelOperation() {
				if !seen[dtrxOp.TransactionId] {
					out = append(out, dtrxOp.TransactionId)
					seen[dtrxOp.TransactionId] = true
				}
			}
		}
	}

	return
}

func (b *Block) CreatedDTrxIDs() (out []string) {
	seen := make(map[string]bool)
	for _, trx := range b.TransactionTraces() {
		for _, dtrxOp := range trx.DtrxOps {
			if dtrxOp.IsCreateOperation() {
				if !seen[dtrxOp.TransactionId] {
					out = append(out, dtrxOp.TransactionId)
					seen[dtrxOp.TransactionId] = true
				}
			}
		}
	}

	return
}

// MigrateV0ToV1 will compute block stats
// for the total number of transaction, transacation trace,
// input action and execute action.
//
// This is actual used on in `pbcodec.Block.ToNative` function to
// re-hydrate the value after decompression until we do a full
// reprocessing. at which time this will not be needed anymore.
func (b *Block) MigrateV0ToV1() {

	if b.Version != 0 {
		return
	}
	b.Version = 1

	b.UnfilteredTransactionCount = uint32(len(b.UnfilteredTransactions))
	b.UnfilteredTransactionTraceCount = uint32(len(b.UnfilteredTransactionTraces))
	b.UnfilteredExecutedInputActionCount = 0
	b.UnfilteredExecutedTotalActionCount = 0

	for _, t := range b.UnfilteredTransactionTraces {
		for _, actionTrace := range t.ActionTraces {
			b.UnfilteredExecutedTotalActionCount++
			if actionTrace.IsInput() {
				b.UnfilteredExecutedInputActionCount++
			}
		}
	}
}

func (t *TransactionTrace) HasBeenReverted() bool {
	// This is an abnormal case, `Receipt` should always be present, but let's assume it's been reverted if no present to play safe
	if t.Receipt == nil {
		return true
	}

	status := t.Receipt.Status

	// Any executed transaction is definitely NOT reverted
	if status == TransactionStatus_TRANSACTIONSTATUS_EXECUTED {
		return false
	}

	// The `eosio::onerror` transaction when in `soft_fail` status means it was actually correctly executed, so it's NOT reverted
	if status == TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL && len(t.ActionTraces) >= 1 && t.ActionTraces[0].FullName() == "eosio:eosio:onerror" {
		return false
	}

	// Even if there is other transaction status, a transaction trace, the object
	// we are in, can only be executed, expired, soft_fail or hard_fail. So at this
	// point, it must have been reverted
	return true
}

func (t *TransactionTrace) DBOpsForAction(idx uint32) (ops []*DBOp) {
	for _, op := range t.DbOps {
		if op.ActionIndex == idx {
			ops = append(ops, op)
		}
	}
	return
}

func (t *TransactionTrace) DtrxOpsForAction(idx uint32) (ops []*DTrxOp) {
	for _, op := range t.DtrxOps {
		if op.ActionIndex == idx {
			ops = append(ops, op)
		}
	}
	return
}

func (t *TransactionTrace) FeatureOpsForAction(idx uint32) (ops []*FeatureOp) {
	for _, op := range t.FeatureOps {
		if op.ActionIndex < 0 { // means not attached to any action
			continue
		}
		if uint32(op.ActionIndex) == idx {
			ops = append(ops, op)
		}
	}
	return
}

func (t *TransactionTrace) PermOpsForAction(idx uint32) (ops []*PermOp) {
	for _, op := range t.PermOps {
		if op.ActionIndex == idx {
			ops = append(ops, op)
		}
	}
	return
}

func (t *TransactionTrace) TableOpsForAction(idx uint32) (ops []*TableOp) {
	for _, op := range t.TableOps {
		if op.ActionIndex == idx {
			ops = append(ops, op)
		}
	}
	return
}

func (t *TransactionTrace) RAMOpsForAction(idx uint32) (ops []*RAMOp) {
	for _, op := range t.RamOps {
		if op.ActionIndex == idx {
			ops = append(ops, op)
		}
	}
	return
}

// CreatorMap creates a mapping between execution trace indexes and
// their parent's execution trace index
func (t *TransactionTrace) CreatorMap() map[uint32]int32 {
	creatorMap := map[uint32]int32{}
	for _, el := range t.CreationTree {
		// formerly idx 2 = Execution
		// formerly idx 1 = Parent
		creatorMap[el.ExecutionActionIndex] = el.CreatorActionIndex
	}
	return creatorMap
}

//
/// ActionTrace
//

func (a *ActionTrace) Name() string {
	return a.Action.Name
}

func (a *ActionTrace) Account() string {
	return a.Action.Account
}

func (a *ActionTrace) SimpleName() string {
	return a.Action.SimpleName()
}

func (a *ActionTrace) FullName() string {
	return a.Receiver + ":" + a.Action.SimpleName()
}

func (a *ActionTrace) GetData(gjsonPath string) gjson.Result {
	// TODO: take that out, to remove the `gjson` dependency in this package.
	return gjson.Get(a.Action.JsonData, gjsonPath)
}

func (a *ActionTrace) IsInput() bool {
	return a.GetCreatorActionOrdinal() == 0
}

//
/// Action
//

func (a *Action) SimpleName() string {
	return a.Account + ":" + a.Name
}

func (a *Action) UnmarshalData(into interface{}) error {
	return json.Unmarshal([]byte(a.JsonData), into)
}

//
/// DTrxOp
//
func (op *DTrxOp) IsCreateOperation() bool {
	return op.Operation == DTrxOp_OPERATION_MODIFY_CREATE ||
		op.Operation == DTrxOp_OPERATION_CREATE ||
		op.Operation == DTrxOp_OPERATION_PUSH_CREATE
}

func (op *DTrxOp) IsCancelOperation() bool {
	return op.Operation == DTrxOp_OPERATION_MODIFY_CANCEL || op.Operation == DTrxOp_OPERATION_CANCEL
}

func (op *DTrxOp) IsFailedOperation() bool {
	return op.Operation == DTrxOp_OPERATION_FAILED
}

func (op *DTrxOp) LegacyOperation() string {
	return strings.Replace(op.Operation.String(), "OPERATION_", "", 1)
}

func (op *DTrxOp) ToExtDTrxOp(block *Block, trxTrace *TransactionTrace) *ExtDTrxOp {
	return &ExtDTrxOp{
		BlockId:             block.Id,
		BlockNum:            uint64(block.Number),
		BlockTime:           block.Header.Timestamp,
		SourceTransactionId: trxTrace.Id,
		DtrxOp:              op,
	}
}

//
/// DBOp
//

func (op *DBOp) LegacyOperation() string {
	switch op.Operation {
	case DBOp_OPERATION_INSERT:
		return "INS"
	case DBOp_OPERATION_UPDATE:
		return "UPD"
	case DBOp_OPERATION_REMOVE:
		return "REM"
	}

	// Impossible to reach, we cover all options above
	return ""
}

//
/// TableOp
//

func (op *TableOp) Path() string {
	return strings.Join([]string{op.Code, op.Scope, op.TableName}, "/")
}

func (op *TableOp) LegacyOperation() string {
	switch op.Operation {
	case TableOp_OPERATION_INSERT:
		return "INS"
	case TableOp_OPERATION_REMOVE:
		return "REM"
	}

	// Impossible to reach, we cover all options above
	return ""
}

//
/// RAMop
//

// LegacyOperation returns the RAMOp tag value that was previously used in eosws/...
func (op *RAMOp) LegacyOperation() string {
	parts := strings.SplitN(op.Operation.String(), "_", 2)

	return strings.ToLower(parts[1])
}

//
/// RlimitOp
//

func (r *RlimitOp) IsGlobalKind() bool {
	_, isConfig := r.Kind.(*RlimitOp_Config)
	_, isState := r.Kind.(*RlimitOp_State)

	return isConfig || isState
}

func (r *RlimitOp) IsLocalKind() bool {
	_, isAccountUsage := r.Kind.(*RlimitOp_AccountUsage)
	_, isAccountLimits := r.Kind.(*RlimitOp_AccountLimits)
	return isAccountUsage || isAccountLimits
}

//
/// PermissionLevel
//

// Authorization returns the concatenation of `Actor`@`Permission` which is
// the standard way to print permission level in string format in EOSIO world.
func (l *PermissionLevel) Authorization() string {
	return l.Actor + "@" + l.Permission
}
