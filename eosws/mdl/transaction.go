// Copyright 2020 dfuse Platform Inc.
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

package mdl

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/trxdb/mdl"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	v0 "github.com/dfuse-io/eosws-go/mdl/v0"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	"github.com/dfuse-io/opaque"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang-collections/collections/stack"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

// TransactionList represents a list of TransactionLifecycle with some cursor
// information for pagination.
// Candidate for a move in eosws-go once the related REST API are made public.
type TransactionList struct {
	Cursor       string                     `json:"cursor"`
	Transactions []*v1.TransactionLifecycle `json:"transactions"`
}

func ToV1TransactionLifecycle(in *pbcodec.TransactionLifecycle) (*v1.TransactionLifecycle, error) {
	createdBy, err := ToV1ExtDTrxOp(in.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("trx lifecycle: createdBy: %w", err)
	}
	canceledBy, err := ToV1ExtDTrxOp(in.CanceledBy)
	if err != nil {
		return nil, fmt.Errorf("trx lifecycle: canceledBy: %w", err)
	}

	var eosSignedTransaction *eos.SignedTransaction
	if in.Transaction != nil {
		eosSignedTransaction = codec.SignedTransactionToEOS(in.Transaction)
	}

	out := &v1.TransactionLifecycle{
		TransactionStatus: codec.TransactionStatusToEOS(in.TransactionStatus).String(),
		ID:                in.Id,
		Transaction:       eosSignedTransaction,
		// DTrxOps:                 ToV1DTrxOps(in.DtrxOps),
		// CreationTree:            ToV1CreationTree(in.CreationTree),
		// DBOps:                   ToV1DBOps(in.DbOps),
		// RAMOps:                  ToV1RAMOps(in.RamOps),
		// TableOps:                ToV1TableOps(in.TableOps),
		PubKeys:                     codec.PublicKeysToEOS(in.PublicKeys),
		CreatedBy:                   createdBy,
		CanceledBy:                  canceledBy,
		ExecutionIrreversible:       in.ExecutionIrreversible,
		DTrxCreationIrreversible:    in.CreationIrreversible,
		DTrxCancelationIrreversible: in.CancelationIrreversible,
	}

	if in.ExecutionTrace != nil {
		out.ExecutionTrace, err = ToV1TransactionTrace(in.ExecutionTrace)
		if err != nil {
			return nil, fmt.Errorf("trx lifecycle: transaction trace: %w", err)
		}
		// FIXME: Make sure this still works, with reworked `Lifecycle` interface.
		out.DBOps = ToV1DBOps(in.ExecutionTrace.DbOps)
		out.RAMOps = ToV1RAMOps(in.ExecutionTrace.RamOps)
		out.TableOps = ToV1TableOps(in.ExecutionTrace.TableOps)
		out.DTrxOps, err = ToV1DTrxOps(in.ExecutionTrace.DtrxOps)
		if err != nil {
			return nil, fmt.Errorf("transaction lifecycle: %w", err)
		}
		out.CreationTree = ToV1CreationTree(in.ExecutionTrace.CreationTree)
	}

	if in.ExecutionBlockHeader != nil {
		out.ExecutionBlockHeader = codec.BlockHeaderToEOS(in.ExecutionBlockHeader)
	}

	return out, nil
}

func ToV1TableOps(in []*pbcodec.TableOp) []*v1.TableOp {

	var out []*v1.TableOp

	for _, i := range in {

		out = append(out,
			&v1.TableOp{
				Operation:   i.LegacyOperation(),
				ActionIndex: int(i.ActionIndex),
				Payer:       i.Payer,
				Path:        i.Path(),
			})
	}
	return out
}
func ToV0TableOps(in []*pbcodec.TableOp) []*v0.TableOp {

	var out []*v0.TableOp

	for _, i := range in {

		out = append(out,
			&v0.TableOp{
				Operation:   i.LegacyOperation(),
				ActionIndex: int(i.ActionIndex),
				Payer:       i.Payer,
				Path:        i.Path(),
			})
	}
	return out
}

func ToV1CreationTree(in []*pbcodec.CreationFlatNode) v1.CreationFlatTree {
	out := v1.CreationFlatTree{}
	for idx, inNode := range in {
		node := v1.CreationFlatNode([3]int{idx, int(inNode.CreatorActionIndex), int(inNode.ExecutionActionIndex)})
		out = append(out, node)
	}

	return out
}

func ToV1RAMOps(in []*pbcodec.RAMOp) (out []*v1.RAMOp) {
	for _, inOp := range in {
		out = append(out, ToV1RAMOp(inOp))
	}
	return out
}
func ToV1RAMOp(in *pbcodec.RAMOp) *v1.RAMOp {
	out := &v1.RAMOp{
		Operation:   in.LegacyOperation(),
		ActionIndex: int(in.ActionIndex),
		Payer:       in.Payer,
		Delta:       in.Delta,
		Usage:       in.Usage,
	}
	return out
}

func ToV0RAMOps(in []*pbcodec.RAMOp) (out []*v0.RAMOp) {
	for _, inOp := range in {
		out = append(out, ToV0RAMOp(inOp))
	}
	return out
}
func ToV0RAMOp(in *pbcodec.RAMOp) *v0.RAMOp {
	out := &v0.RAMOp{
		ActionIndex: int(in.ActionIndex),
		EventID:     in.UniqueKey,
		Family:      namespaceToFamily(in.Namespace),
		Action:      strings.ToLower(strings.TrimPrefix(in.Action.String(), "ACTION_")),
		Operation:   in.LegacyOperation(),
		Payer:       in.Payer,
		Delta:       eos.Int64(in.Delta),
		Usage:       eos.Uint64(in.Usage),
	}
	return out
}

func namespaceToFamily(ns pbcodec.RAMOp_Namespace) string {
	return strings.ToLower(strings.TrimPrefix(ns.String(), "NAMESPACE_"))
}

func ToV1DTrxOps(in []*pbcodec.DTrxOp) (out []*v1.DTrxOp, err error) {
	for _, inOp := range in {
		op, err := ToV1DTrxOp(inOp)
		if err != nil {
			return nil, fmt.Errorf("dtrx op: %w", err)
		}
		out = append(out, op)
	}
	return out, nil
}

func ToV1ExtDTrxOp(in *pbcodec.ExtDTrxOp) (*v1.ExtDTrxOp, error) {
	if in == nil {
		return nil, nil
	}
	out := &v1.ExtDTrxOp{
		SourceTransactionID: in.SourceTransactionId,
		BlockNum:            uint32(in.BlockNum),
		BlockID:             in.BlockId,
		BlockTime:           codec.TimestampToBlockTimestamp(in.BlockTime),
	}
	op, err := ToV1DTrxOp(in.DtrxOp)
	if err != nil {
		return nil, fmt.Errorf("ext dtrx op: %w", err)
	}
	out.DTrxOp = *op
	return out, nil
}
func ToV0DTrxOps(in []*pbcodec.DTrxOp) (out []*v0.DTrxOp) {
	for _, inOp := range in {
		out = append(out, ToV0DTrxOp(inOp))
	}
	return out
}

func ToV0DTrxOp(in *pbcodec.DTrxOp) *v0.DTrxOp {
	if in == nil {
		return nil
	}
	trx := codec.SignedTransactionToEOS(in.Transaction)
	out := &v0.DTrxOp{
		Operation:     strings.ToLower(in.LegacyOperation()),
		ActionIndex:   int(in.ActionIndex),
		Sender:        in.Sender,
		SenderID:      in.SenderId,
		Payer:         in.Payer,
		PublishedAt:   in.PublishedAt,
		DelayUntil:    in.DelayUntil,
		ExpirationAt:  in.ExpirationAt,
		TransactionID: in.TransactionId,
		Transaction:   trx.Transaction,
	}

	return out
}
func ToV1DTrxOp(in *pbcodec.DTrxOp) (*v1.DTrxOp, error) {
	if in == nil {
		return nil, nil
	}
	trx := codec.SignedTransactionToEOS(in.Transaction)
	rawTrx, err := json.Marshal(trx)
	if err != nil {

		return nil, fmt.Errorf("couldn't marshal transaction %q: %w", in.TransactionId, err)
	}

	out := &v1.DTrxOp{
		Operation:    in.LegacyOperation(),
		ActionIndex:  int(in.ActionIndex),
		Sender:       in.Sender,
		SenderID:     in.SenderId,
		Payer:        in.Payer,
		PublishedAt:  in.PublishedAt,
		DelayUntil:   in.DelayUntil,
		ExpirationAt: in.ExpirationAt,
		TrxID:        in.TransactionId,
		Trx:          rawTrx,
	}

	return out, nil
}

func ToV1AccountRAMDeltas(in []*pbcodec.AccountRAMDelta) (out []*v1.AccountRAMDelta) {
	for _, inDelta := range in {
		out = append(out, ToV1AccountRAMDelta(inDelta))
	}
	return out
}

func ToV1AccountRAMDelta(in *pbcodec.AccountRAMDelta) *v1.AccountRAMDelta {
	if in == nil {
		return nil
	}
	out := &v1.AccountRAMDelta{
		Account: eos.AccountName(in.Account),
		Delta:   in.Delta,
	}
	return out
}

func ToV1TransactionTrace(in *pbcodec.TransactionTrace) (*v1.TransactionTrace, error) {
	if in == nil {
		return nil, nil
	}
	rawExcept, _ := json.Marshal(codec.ExceptionToEOS(in.Exception))
	actionTrace, err := ToV1ActionTraces(in.ActionTraces)
	if err != nil {
		return nil, fmt.Errorf("transaction trace: action trace: %w", err)
	}
	out := &v1.TransactionTrace{
		ID:              in.Id,
		BlockNum:        uint32(in.BlockNum),
		BlockTime:       codec.TimestampToBlockTimestamp(in.BlockTime),
		ProducerBlockID: in.ProducerBlockId,
		Elapsed:         in.Elapsed,
		NetUsage:        in.NetUsage,
		Scheduled:       in.Scheduled,
		ActionTraces:    actionTrace,
		Except:          json.RawMessage(rawExcept),
	}
	if in.Receipt != nil {
		out.Receipt = *(codec.TransactionReceiptHeaderToEOS(in.Receipt))
	}

	if in.FailedDtrxTrace != nil {
		out.FailedDTrxTrace, err = ToV1TransactionTrace(in.FailedDtrxTrace)
		if err != nil {
			return nil, fmt.Errorf("failed transaction trace: action trace: %w", err)
		}
	}
	return out, nil
}

func ToV1ActionTraces(in []*pbcodec.ActionTrace) (out []*v1.ActionTrace, err error) {
	executionOrdinal := func(act *pbcodec.ActionTrace) uint64 {
		if act.Receipt == nil || act.Receipt.GlobalSequence == 0 {
			return math.MaxUint64
		}

		return uint64(act.Receipt.GlobalSequence)
	}

	inByExecutionOrder := make([]*pbcodec.ActionTrace, len(in))
	copy(inByExecutionOrder, in)
	sort.Slice(inByExecutionOrder, func(i, j int) bool {
		return executionOrdinal(inByExecutionOrder[i]) < executionOrdinal(inByExecutionOrder[j])
	})

	var lastMDLActionTrace *pbcodec.ActionTrace
	var lastV1ActionTrace *v1.ActionTrace
	parentStack := &ExtendedStack{}

	for _, inActionTrace := range inByExecutionOrder {
		trace, err := ToV1ActionTrace(inActionTrace)
		if err != nil {
			return nil, fmt.Errorf("action traces: %w", err)
		}
		cuaOrdinal := inActionTrace.ClosestUnnotifiedAncestorActionOrdinal

		// We are a top-level actions
		if cuaOrdinal == 0 {
			out = append(out, trace)
			parentStack.Push(trace)
		} else {
			lastCUAOrdinal := lastMDLActionTrace.ClosestUnnotifiedAncestorActionOrdinal

			// We are starting a new children sequence
			if cuaOrdinal > lastCUAOrdinal {
				parentStack.Push(lastV1ActionTrace)
			}

			// We are returning to the previous children sequence
			if cuaOrdinal < lastCUAOrdinal {
				parentStack.MustPop()
			}

			parentTrace := parentStack.MustPeek().(*v1.ActionTrace)
			parentTrace.InlineTraces = append(parentTrace.InlineTraces, trace)
		}

		lastMDLActionTrace = inActionTrace
		lastV1ActionTrace = trace
	}

	return out, nil
}

// ToV1ActionTraceRaw converts betwen `pbcodec.ActionWrap` format to the old v1 action traces format
// containing `inline_traces` object.
//
// **Warning** The `actions` parameter is expected to be sorted by execution order (the
// default) for the algorithm to work correctly.
func ToV1ActionTraceRaw(parentAction *pbcodec.ActionTrace, actions []*pbcodec.ActionTrace, withInlines bool) (json.RawMessage, error) {
	// Only elements that directly follow us on the flat list can be our children, limit to that
	potentialChildren := actions[parentAction.ExecutionIndex+1:]

	var rawChildTraces []byte
	if withInlines {
		rawChildTraces = []byte{'['}
		for _, child := range potentialChildren {
			// As soon as the potential child is not our child, there is no more child to visit
			if child.ClosestUnnotifiedAncestorActionOrdinal != parentAction.ActionOrdinal {
				break
			}

			if len(rawChildTraces) != 1 {
				// It's not the first child, otherwise length would be 1, add coma before the new element
				rawChildTraces = append(rawChildTraces, ',')
			}

			actionTraceRaw, err := ToV1ActionTraceRaw(child, actions, true)
			if err != nil {
				return nil, fmt.Errorf("failed to convert action raw: %w", err)
			}
			rawChildTraces = append(rawChildTraces, actionTraceRaw...)
		}
		rawChildTraces = append(rawChildTraces, ']')
	}

	trace := codec.ActionTraceToEOS(parentAction)
	rawTrace, err := json.Marshal(trace)
	if err != nil {
		return nil, fmt.Errorf("marshaling action traces %v: %w", trace, err)
	}

	if withInlines {
		//fmt.Println("INLINE", string(rawChildTraces))
		rawTrace, err = sjson.SetRawBytes(rawTrace, "inline_traces", rawChildTraces)
		if err != nil {
			return nil, fmt.Errorf("set raw byte: %w", err)
		}
	} else {
		rawTrace, err = sjson.SetRawBytes(rawTrace, "inline_traces", []byte("[]"))
		if err != nil {
			return nil, fmt.Errorf("set raw byte no inline: %w", err)
		}
	}

	return rawTrace, nil
}

func ToV1ActionTrace(in *pbcodec.ActionTrace) (*v1.ActionTrace, error) {
	if in == nil {
		return nil, nil
	}

	rawExcept, err := json.Marshal(codec.ExceptionToEOS(in.Exception))
	if err != nil {
		return nil, fmt.Errorf("marshaling action exception %v: %w", in.Exception, err)
	}

	out := &v1.ActionTrace{}
	out.Receipt = ToV1ActionReceipt(string(in.Receiver), in.Receipt)
	if in.Action != nil {
		out.Action = *(codec.ActionToEOS(in.Action))
	}
	out.ContextFree = in.ContextFree
	out.Elapsed = in.Elapsed
	out.Console = in.Console
	out.TransactionID = in.TransactionId
	out.BlockNum = uint32(in.BlockNum)
	out.BlockTime = codec.TimestampToBlockTimestamp(in.BlockTime)
	out.ProducerBlockID = &in.ProducerBlockId
	out.AccountRAMDeltas = ToV1AccountRAMDeltas(in.AccountRamDeltas)
	out.Except = json.RawMessage(rawExcept)

	return out, nil
}

func ToV1ActionReceipt(receiver string, in *pbcodec.ActionReceipt) v1.ActionReceipt {
	if in == nil {
		return v1.ActionReceipt{
			Receiver:     receiver,
			AuthSequence: []json.RawMessage{},
		}
	}

	authSeqs := make([]json.RawMessage, 0)
	for _, authSeq := range in.AuthSequence {
		cnt, _ := json.Marshal(codec.AuthSequenceToEOS(authSeq))
		authSeqs = append(authSeqs, json.RawMessage(cnt))
	}
	out := v1.ActionReceipt{
		Receiver:       in.Receiver,
		Digest:         in.Digest,
		GlobalSequence: eos.Uint64(in.GlobalSequence),
		AuthSequence:   authSeqs,
		RecvSequence:   eos.Uint64(in.RecvSequence),
		CodeSequence:   eos.Uint64(in.CodeSequence),
		ABISequence:    eos.Uint64(in.AbiSequence),
	}
	return out
}

func ToV1TransactionList(list *mdl.TransactionList) (*TransactionList, error) {
	opaqueNextCursor, err := opaque.ToOpaque(list.NextCursor)
	if err != nil {
		zlog.Error("converting cursor", zap.Error(err))
		return nil, err
	}

	out := &TransactionList{
		Cursor: opaqueNextCursor,
	}

	var outList []*v1.TransactionLifecycle
	for _, tx := range list.Transactions {
		// FIXME: which Chain Discriminator should we be using here?
		lifecycle := pbcodec.MergeTransactionEvents(tx, func(id string) bool { return true })
		v1Lifecycle, err := ToV1TransactionLifecycle(lifecycle)
		if err != nil {
			return nil, fmt.Errorf("transaction list: %w", err)
		}
		outList = append(outList, v1Lifecycle)
	}

	out.Transactions = outList

	return out, nil
}

type ExtendedStack struct {
	stack.Stack
}

func (s *ExtendedStack) MustPeek() interface{} {
	peek := s.Peek()
	if peek == nil {
		panic("at least one parent must exist in stack at this point")
	}

	return peek
}

func (s *ExtendedStack) MustPop() interface{} {
	popped := s.Pop()
	if popped == nil {
		panic("at least one parent must exist in stack at this point")
	}

	return popped
}
