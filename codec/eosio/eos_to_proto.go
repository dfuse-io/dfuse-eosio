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

package eosio

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func ActivatedProtocolFeaturesToDEOS(in *eos.ProtocolFeatureActivationSet) *pbcodec.ActivatedProtocolFeatures {
	out := &pbcodec.ActivatedProtocolFeatures{}
	out.ProtocolFeatures = checksumsToBytesSlices(in.ProtocolFeatures)
	return out
}

func PendingScheduleToDEOS(in *eos.PendingSchedule) *pbcodec.PendingProducerSchedule {
	out := &pbcodec.PendingProducerSchedule{
		ScheduleLibNum: in.ScheduleLIBNum,
		ScheduleHash:   []byte(in.ScheduleHash),
	}

	/// Specific versions handling

	// Only in EOSIO 1.x
	if in.Schedule.V1 != nil {
		out.ScheduleV1 = ProducerScheduleToDEOS(in.Schedule.V1)
	}

	// Only in EOSIO 2.x
	if in.Schedule.V2 != nil {
		out.ScheduleV2 = ProducerAuthorityScheduleToDEOS(in.Schedule.V2)
	}

	// End (versions)

	return out
}

func ProducerToLastProducedToDEOS(in []eos.PairAccountNameBlockNum) []*pbcodec.ProducerToLastProduced {
	out := make([]*pbcodec.ProducerToLastProduced, len(in))
	for i, elem := range in {
		out[i] = &pbcodec.ProducerToLastProduced{
			Name:                 string(elem.AccountName),
			LastBlockNumProduced: uint32(elem.BlockNum),
		}
	}
	return out
}

func ProducerToLastImpliedIrbToDEOS(in []eos.PairAccountNameBlockNum) []*pbcodec.ProducerToLastImpliedIRB {
	out := make([]*pbcodec.ProducerToLastImpliedIRB, len(in))
	for i, elem := range in {
		out[i] = &pbcodec.ProducerToLastImpliedIRB{
			Name:                 string(elem.AccountName),
			LastBlockNumProduced: uint32(elem.BlockNum),
		}
	}
	return out
}

func BlockrootMerkleToDEOS(merkle *eos.MerkleRoot) *pbcodec.BlockRootMerkle {
	return &pbcodec.BlockRootMerkle{
		NodeCount:   uint32(merkle.NodeCount),
		ActiveNodes: checksumsToBytesSlices(merkle.ActiveNodes),
	}
}

func BlockHeaderToDEOS(blockHeader *eos.BlockHeader) *pbcodec.BlockHeader {
	out := &pbcodec.BlockHeader{
		Timestamp:        mustProtoTimestamp(blockHeader.Timestamp.Time),
		Producer:         string(blockHeader.Producer),
		Confirmed:        uint32(blockHeader.Confirmed),
		Previous:         blockHeader.Previous.String(),
		TransactionMroot: blockHeader.TransactionMRoot,
		ActionMroot:      blockHeader.ActionMRoot,
		ScheduleVersion:  blockHeader.ScheduleVersion,
		HeaderExtensions: ExtensionsToDEOS(blockHeader.HeaderExtensions),
	}

	if blockHeader.NewProducersV1 != nil {
		out.NewProducersV1 = ProducerScheduleToDEOS(blockHeader.NewProducersV1)
	}

	return out
}

func BlockSigningAuthorityToDEOS(authority *eos.BlockSigningAuthority) *pbcodec.BlockSigningAuthority {
	out := &pbcodec.BlockSigningAuthority{}

	switch v := authority.Impl.(type) {
	case *eos.BlockSigningAuthorityV0:
		out.Variant = &pbcodec.BlockSigningAuthority_V0{
			V0: &pbcodec.BlockSigningAuthorityV0{
				Threshold: v.Threshold,
				Keys:      KeyWeightsPToDEOS(v.Keys),
			},
		}
	default:
		panic(fmt.Errorf("unable to convert eos.BlockSigningAuthority to deos: wrong type %T", authority.Impl))
	}

	return out
}

func ProducerScheduleToDEOS(e *eos.ProducerSchedule) *pbcodec.ProducerSchedule {
	return &pbcodec.ProducerSchedule{
		Version:   uint32(e.Version),
		Producers: ProducerKeysToDEOS(e.Producers),
	}
}

func ProducerAuthorityScheduleToDEOS(e *eos.ProducerAuthoritySchedule) *pbcodec.ProducerAuthoritySchedule {
	return &pbcodec.ProducerAuthoritySchedule{
		Version:   uint32(e.Version),
		Producers: ProducerAuthoritiesToDEOS(e.Producers),
	}
}

func ProducerKeysToDEOS(in []eos.ProducerKey) (out []*pbcodec.ProducerKey) {
	out = make([]*pbcodec.ProducerKey, len(in))
	for i, key := range in {
		out[i] = &pbcodec.ProducerKey{
			AccountName:     string(key.AccountName),
			BlockSigningKey: key.BlockSigningKey.String(),
		}
	}
	return
}

func ExtensionsToDEOS(in []*eos.Extension) (out []*pbcodec.Extension) {
	out = make([]*pbcodec.Extension, len(in))
	for i, extension := range in {
		out[i] = &pbcodec.Extension{
			Type: uint32(extension.Type),
			Data: extension.Data,
		}
	}

	return
}

func ProducerAuthoritiesToDEOS(producerAuthorities []*eos.ProducerAuthority) (out []*pbcodec.ProducerAuthority) {
	if len(producerAuthorities) <= 0 {
		return nil
	}

	out = make([]*pbcodec.ProducerAuthority, len(producerAuthorities))
	for i, authority := range producerAuthorities {
		out[i] = &pbcodec.ProducerAuthority{
			AccountName:           string(authority.AccountName),
			BlockSigningAuthority: BlockSigningAuthorityToDEOS(authority.BlockSigningAuthority),
		}
	}
	return
}

func TransactionReceiptToDEOS(txReceipt *eos.TransactionReceipt) *pbcodec.TransactionReceipt {
	receipt := &pbcodec.TransactionReceipt{
		Status:               TransactionStatusToDEOS(txReceipt.Status),
		CpuUsageMicroSeconds: txReceipt.CPUUsageMicroSeconds,
		NetUsageWords:        uint32(txReceipt.NetUsageWords),
	}

	receipt.Id = txReceipt.Transaction.ID.String()
	if txReceipt.Transaction.Packed != nil {
		receipt.PackedTransaction = &pbcodec.PackedTransaction{
			Signatures:            SignaturesToDEOS(txReceipt.Transaction.Packed.Signatures),
			Compression:           uint32(txReceipt.Transaction.Packed.Compression),
			PackedContextFreeData: txReceipt.Transaction.Packed.PackedContextFreeData,
			PackedTransaction:     txReceipt.Transaction.Packed.PackedTransaction,
		}
	}

	return receipt
}

func TransactionReceiptHeaderToDEOS(in *eos.TransactionReceiptHeader) *pbcodec.TransactionReceiptHeader {
	return &pbcodec.TransactionReceiptHeader{
		Status:               TransactionStatusToDEOS(in.Status),
		CpuUsageMicroSeconds: in.CPUUsageMicroSeconds,
		NetUsageWords:        uint32(in.NetUsageWords),
	}
}

func SignaturesToDEOS(in []ecc.Signature) (out []string) {
	out = make([]string, len(in))
	for i, signature := range in {
		out[i] = signature.String()
	}
	return
}

func PermissionToDEOS(perm *eos.Permission) *pbcodec.Permission {
	return &pbcodec.Permission{
		Name:         perm.PermName,
		Parent:       perm.Parent,
		RequiredAuth: AuthoritiesToDEOS(&perm.RequiredAuth),
	}
}

func AuthoritiesToDEOS(authority *eos.Authority) *pbcodec.Authority {
	return &pbcodec.Authority{
		Threshold: authority.Threshold,
		Keys:      KeyWeightsToDEOS(authority.Keys),
		Accounts:  PermissionLevelWeightsToDEOS(authority.Accounts),
		Waits:     WaitWeightsToDEOS(authority.Waits),
	}
}

func WaitWeightsToDEOS(waits []eos.WaitWeight) (out []*pbcodec.WaitWeight) {
	if len(waits) <= 0 {
		return nil
	}

	out = make([]*pbcodec.WaitWeight, len(waits))
	for i, o := range waits {
		out[i] = &pbcodec.WaitWeight{
			WaitSec: o.WaitSec,
			Weight:  uint32(o.Weight),
		}
	}
	return out
}

func PermissionLevelWeightsToDEOS(weights []eos.PermissionLevelWeight) (out []*pbcodec.PermissionLevelWeight) {
	if len(weights) <= 0 {
		return nil
	}

	out = make([]*pbcodec.PermissionLevelWeight, len(weights))
	for i, o := range weights {
		out[i] = &pbcodec.PermissionLevelWeight{
			Permission: PermissionLevelToDEOS(o.Permission),
			Weight:     uint32(o.Weight),
		}
	}
	return
}

func PermissionLevelToDEOS(perm eos.PermissionLevel) *pbcodec.PermissionLevel {
	return &pbcodec.PermissionLevel{
		Actor:      string(perm.Actor),
		Permission: string(perm.Permission),
	}
}

func KeyWeightsToDEOS(keys []eos.KeyWeight) (out []*pbcodec.KeyWeight) {
	if len(keys) <= 0 {
		return nil
	}

	out = make([]*pbcodec.KeyWeight, len(keys))
	for i, o := range keys {
		out[i] = &pbcodec.KeyWeight{
			PublicKey: o.PublicKey.String(),
			Weight:    uint32(o.Weight),
		}
	}
	return
}

func KeyWeightsPToDEOS(keys []*eos.KeyWeight) (out []*pbcodec.KeyWeight) {
	if len(keys) <= 0 {
		return nil
	}

	out = make([]*pbcodec.KeyWeight, len(keys))
	for i, o := range keys {
		out[i] = &pbcodec.KeyWeight{
			PublicKey: o.PublicKey.String(),
			Weight:    uint32(o.Weight),
		}
	}
	return
}

func TransactionToDEOS(trx *eos.Transaction) *pbcodec.Transaction {
	var contextFreeActions []*pbcodec.Action
	if len(trx.ContextFreeActions) > 0 {
		contextFreeActions = make([]*pbcodec.Action, len(trx.ContextFreeActions))
		for i, act := range trx.ContextFreeActions {
			contextFreeActions[i] = ActionToDEOS(act)
		}
	}

	var actions []*pbcodec.Action
	if len(trx.Actions) > 0 {
		actions = make([]*pbcodec.Action, len(trx.Actions))
		for i, act := range trx.Actions {
			actions[i] = ActionToDEOS(act)
		}
	}

	return &pbcodec.Transaction{
		Header:             TransactionHeaderToDEOS(&trx.TransactionHeader),
		ContextFreeActions: contextFreeActions,
		Actions:            actions,
		Extensions:         ExtensionsToDEOS(trx.Extensions),
	}
}

func TransactionHeaderToDEOS(trx *eos.TransactionHeader) *pbcodec.TransactionHeader {
	out := &pbcodec.TransactionHeader{
		Expiration:       mustProtoTimestamp(trx.Expiration.Time),
		RefBlockNum:      uint32(trx.RefBlockNum),
		RefBlockPrefix:   trx.RefBlockPrefix,
		MaxNetUsageWords: uint32(trx.MaxNetUsageWords),
		MaxCpuUsageMs:    uint32(trx.MaxCPUUsageMS),
		DelaySec:         uint32(trx.DelaySec),
	}

	return out
}

func SignedTransactionToDEOS(trx *eos.SignedTransaction) *pbcodec.SignedTransaction {
	return &pbcodec.SignedTransaction{
		Transaction:     TransactionToDEOS(trx.Transaction),
		Signatures:      SignaturesToDEOS(trx.Signatures),
		ContextFreeData: hexBytesToBytesSlices(trx.ContextFreeData),
	}
}

func CreationTreeToDEOS(tree CreationFlatTree) []*pbcodec.CreationFlatNode {
	if len(tree) <= 0 {
		return nil
	}

	out := make([]*pbcodec.CreationFlatNode, len(tree))
	for i, node := range tree {
		out[i] = &pbcodec.CreationFlatNode{
			CreatorActionIndex:   int32(node[1]),
			ExecutionActionIndex: uint32(node[2]),
		}
	}
	return out
}

func AuthSequenceToDEOS(in eos.TransactionTraceAuthSequence) *pbcodec.AuthSequence {
	return &pbcodec.AuthSequence{
		AccountName: string(in.Account),
		Sequence:    uint64(in.Sequence),
	}
}

func ActionTraceReceiptToDEOS(in *eos.ActionTraceReceipt) *pbcodec.ActionReceipt {
	authSequences := in.AuthSequence

	var deosAuthSequence []*pbcodec.AuthSequence
	if len(authSequences) > 0 {
		deosAuthSequence = make([]*pbcodec.AuthSequence, len(authSequences))
		for i, seq := range authSequences {
			deosAuthSequence[i] = AuthSequenceToDEOS(seq)
		}
	}

	return &pbcodec.ActionReceipt{
		Receiver:       string(in.Receiver),
		Digest:         in.ActionDigest.String(),
		GlobalSequence: uint64(in.GlobalSequence),
		AuthSequence:   deosAuthSequence,
		RecvSequence:   uint64(in.ReceiveSequence),
		CodeSequence:   uint64(in.CodeSequence),
		AbiSequence:    uint64(in.ABISequence),
	}
}

func ErrorCodeToDEOS(in *eos.Uint64) uint64 {
	if in != nil {
		return uint64(*in)
	}
	return 0
}

func ActionToDEOS(action *eos.Action) *pbcodec.Action {
	deosAction := &pbcodec.Action{
		Account:       string(action.Account),
		Name:          string(action.Name),
		Authorization: AuthorizationToDEOS(action.Authorization),
		RawData:       action.HexData,
	}

	if action.Data != nil {
		v, dataIsString := action.Data.(string)
		if dataIsString && len(action.HexData) == 0 {
			// When the action.Data is actually a string, and the HexData field is not set, we assume data sould be rawData instead
			rawData, err := hex.DecodeString(v)
			if err != nil {
				panic(fmt.Errorf("unable to unmarshal action data %q as hex: %s", v, err))
			}

			deosAction.RawData = rawData
		} else {
			serializedData, err := json.Marshal(action.Data)
			if err != nil {
				panic(fmt.Errorf("unable to unmarshal action data JSON: %s", err))
			}

			deosAction.JsonData = string(serializedData)
		}
	}

	return deosAction
}

func AuthorizationToDEOS(authorization []eos.PermissionLevel) (out []*pbcodec.PermissionLevel) {
	if len(authorization) <= 0 {
		return nil
	}

	out = make([]*pbcodec.PermissionLevel, len(authorization))
	for i, permission := range authorization {
		out[i] = PermissionLevelToDEOS(permission)
	}
	return
}

func AccountRAMDeltasToDEOS(deltas []*eos.AccountRAMDelta) (out []*pbcodec.AccountRAMDelta) {
	if len(deltas) <= 0 {
		return nil
	}

	out = make([]*pbcodec.AccountRAMDelta, len(deltas))
	for i, delta := range deltas {
		out[i] = &pbcodec.AccountRAMDelta{
			Account: string(delta.Account),
			Delta:   int64(delta.Delta),
		}
	}
	return
}

func ExceptionToDEOS(in *eos.Except) *pbcodec.Exception {
	if in == nil {
		return nil
	}
	out := &pbcodec.Exception{
		Code:    int32(in.Code),
		Name:    in.Name,
		Message: in.Message,
	}

	if len(in.Stack) > 0 {
		out.Stack = make([]*pbcodec.Exception_LogMessage, len(in.Stack))
		for i, el := range in.Stack {
			out.Stack[i] = &pbcodec.Exception_LogMessage{
				Context: LogContextToDEOS(el.Context),
				Format:  el.Format,
				Data:    el.Data,
			}
		}
	}

	return out
}

func LogContextToDEOS(in eos.ExceptLogContext) *pbcodec.Exception_LogContext {
	out := &pbcodec.Exception_LogContext{
		Level:      in.Level.String(),
		File:       in.File,
		Line:       int32(in.Line),
		Method:     in.Method,
		Hostname:   in.Hostname,
		ThreadName: in.ThreadName,
		Timestamp:  mustProtoTimestamp(in.Timestamp.Time),
	}
	if in.Context != nil {
		out.Context = LogContextToDEOS(*in.Context)
	}
	return out
}

func TransactionStatusToDEOS(in eos.TransactionStatus) pbcodec.TransactionStatus {
	switch in {
	case eos.TransactionStatusExecuted:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED
	case eos.TransactionStatusSoftFail:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL
	case eos.TransactionStatusHardFail:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL
	case eos.TransactionStatusDelayed:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_DELAYED
	case eos.TransactionStatusExpired:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXPIRED
	default:
		return pbcodec.TransactionStatus_TRANSACTIONSTATUS_UNKNOWN
	}
}

func checksumsToBytesSlices(in []eos.Checksum256) [][]byte {
	out := make([][]byte, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

func hexBytesToBytesSlices(in []eos.HexBytes) [][]byte {
	out := make([][]byte, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}
