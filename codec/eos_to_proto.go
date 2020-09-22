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

package codec

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/zap"
)

type conversionOption interface{}

type actionConversionOption interface {
	apply(actionTrace *pbcodec.ActionTrace)
}

type actionConversionOptionFunc func(actionTrace *pbcodec.ActionTrace)

func (f actionConversionOptionFunc) apply(actionTrace *pbcodec.ActionTrace) {
	f(actionTrace)
}

func limitConsoleLengthConversionOption(maxCharacterCount int) conversionOption {
	return actionConversionOptionFunc(func(actionTrace *pbcodec.ActionTrace) {
		if len(actionTrace.Console) > maxCharacterCount {
			actionTrace.Console = actionTrace.Console[0:maxCharacterCount]
		}
	})
}

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

func bytesSlicesToHexBytes(in [][]byte) []eos.HexBytes {
	out := make([]eos.HexBytes, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
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

func BlockHeaderToEOS(in *pbcodec.BlockHeader) *eos.BlockHeader {
	stamp, _ := ptypes.Timestamp(in.Timestamp)
	prev, _ := hex.DecodeString(in.Previous)
	out := &eos.BlockHeader{
		Timestamp:        eos.BlockTimestamp{Time: stamp},
		Producer:         eos.AccountName(in.Producer),
		Confirmed:        uint16(in.Confirmed),
		Previous:         prev,
		TransactionMRoot: in.TransactionMroot,
		ActionMRoot:      in.ActionMroot,
		ScheduleVersion:  in.ScheduleVersion,
		HeaderExtensions: ExtensionsToEOS(in.HeaderExtensions),
	}

	if in.NewProducersV1 != nil {
		out.NewProducersV1 = ProducerScheduleToEOS(in.NewProducersV1)
	}

	return out
}

func BlockSigningAuthorityToDEOS(authority *eos.BlockSigningAuthority) *pbcodec.BlockSigningAuthority {
	out := &pbcodec.BlockSigningAuthority{}
	err := authority.DoFor(map[uint32]eos.OnVariant{
		eos.BlockSigningAuthorityV0Type: func(impl interface{}) error {
			v := impl.(*eos.BlockSigningAuthorityV0)

			out.Variant = &pbcodec.BlockSigningAuthority_V0{
				V0: &pbcodec.BlockSigningAuthorityV0{
					Threshold: v.Threshold,
					Keys:      KeyWeightsPToDEOS(v.Keys),
				},
			}

			return nil
		},
	})

	if err != nil {
		panic(fmt.Errorf("unable to convert eos.BlockSigningAuthority to deos: %s", err))
	}

	return out
}

func BlockSigningAuthorityToEOS(in *pbcodec.BlockSigningAuthority) *eos.BlockSigningAuthority {
	out := &eos.BlockSigningAuthority{}
	switch v := in.Variant.(type) {
	case *pbcodec.BlockSigningAuthority_V0:
		out.TypeID = eos.BlockSigningAuthorityV0Type
		out.Impl = &eos.BlockSigningAuthorityV0{
			Threshold: v.V0.Threshold,
		}

		return out
	}

	panic(fmt.Errorf("unknown block signing authority variant %t", in.Variant))
}

func ProducerScheduleToDEOS(e *eos.ProducerSchedule) *pbcodec.ProducerSchedule {
	return &pbcodec.ProducerSchedule{
		Version:   uint32(e.Version),
		Producers: ProducerKeysToDEOS(e.Producers),
	}
}

func ProducerScheduleToEOS(in *pbcodec.ProducerSchedule) *eos.ProducerSchedule {
	return &eos.ProducerSchedule{
		Version:   in.Version,
		Producers: ProducerKeysToEOS(in.Producers),
	}
}

func ProducerAuthorityScheduleToDEOS(e *eos.ProducerAuthoritySchedule) *pbcodec.ProducerAuthoritySchedule {
	return &pbcodec.ProducerAuthoritySchedule{
		Version:   uint32(e.Version),
		Producers: ProducerAuthoritiesToDEOS(e.Producers),
	}
}

func ProducerAuthorityScheduleToEOS(in *pbcodec.ProducerAuthoritySchedule) *eos.ProducerAuthoritySchedule {
	return &eos.ProducerAuthoritySchedule{
		Version:   in.Version,
		Producers: ProducerAuthoritiesToEOS(in.Producers),
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

func ProducerKeysToEOS(in []*pbcodec.ProducerKey) (out []eos.ProducerKey) {
	out = make([]eos.ProducerKey, len(in))
	for i, producer := range in {
		// panic on error instead?
		key, _ := ecc.NewPublicKey(producer.BlockSigningKey)

		out[i] = eos.ProducerKey{
			AccountName:     eos.AccountName(producer.AccountName),
			BlockSigningKey: key,
		}
	}
	return
}

func PublicKeysToEOS(in []string) (out []*ecc.PublicKey) {
	if len(in) <= 0 {
		return nil
	}
	out = make([]*ecc.PublicKey, len(in))
	for i, inkey := range in {
		// panic on error instead?
		key, _ := ecc.NewPublicKey(inkey)

		out[i] = &key
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

func ExtensionsToEOS(in []*pbcodec.Extension) (out []*eos.Extension) {
	if len(in) <= 0 {
		return nil
	}

	out = make([]*eos.Extension, len(in))
	for i, extension := range in {
		out[i] = &eos.Extension{
			Type: uint16(extension.Type),
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

func ProducerAuthoritiesToEOS(producerAuthorities []*pbcodec.ProducerAuthority) (out []*eos.ProducerAuthority) {
	if len(producerAuthorities) <= 0 {
		return nil
	}

	out = make([]*eos.ProducerAuthority, len(producerAuthorities))
	for i, authority := range producerAuthorities {
		out[i] = &eos.ProducerAuthority{
			AccountName:           eos.AccountName(authority.AccountName),
			BlockSigningAuthority: BlockSigningAuthorityToEOS(authority.BlockSigningAuthority),
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

func TransactionReceiptHeaderToEOS(in *pbcodec.TransactionReceiptHeader) *eos.TransactionReceiptHeader {
	return &eos.TransactionReceiptHeader{
		Status:               TransactionStatusToEOS(in.Status),
		CPUUsageMicroSeconds: in.CpuUsageMicroSeconds,
		NetUsageWords:        eos.Varuint32(in.NetUsageWords),
	}
}

func SignaturesToDEOS(in []ecc.Signature) (out []string) {

	out = make([]string, len(in))
	for i, signature := range in {
		out[i] = signature.String()
	}
	return
}

func SignaturesToEOS(in []string) []ecc.Signature {
	out := make([]ecc.Signature, len(in))
	for i, signature := range in {
		sig, err := ecc.NewSignature(signature)
		if err != nil {
			panic(fmt.Sprintf("failed to read signature %q: %s", signature, err))
		}

		out[i] = sig
	}
	return out
}

func TransactionTraceToDEOS(in *eos.TransactionTrace, opts ...conversionOption) *pbcodec.TransactionTrace {
	id := in.ID.String()

	out := &pbcodec.TransactionTrace{
		Id:              id,
		BlockNum:        uint64(in.BlockNum),
		BlockTime:       mustProtoTimestamp(in.BlockTime.Time),
		ProducerBlockId: in.ProducerBlockID.String(),
		Elapsed:         int64(in.Elapsed),
		NetUsage:        uint64(in.NetUsage),
		Scheduled:       in.Scheduled,
		Exception:       ExceptionToDEOS(in.Except),
		ErrorCode:       ErrorCodeToDEOS(in.ErrorCode),
	}

	var someConsoleTruncated bool
	out.ActionTraces, someConsoleTruncated = ActionTracesToDEOS(in.ActionTraces, opts...)
	if someConsoleTruncated {
		zlog.Info("transaction had some of its action trace's console entries truncated", zap.String("id", id))
	}

	if in.FailedDtrxTrace != nil {
		out.FailedDtrxTrace = TransactionTraceToDEOS(in.FailedDtrxTrace, opts...)
	}
	if in.Receipt != nil {
		out.Receipt = TransactionReceiptHeaderToDEOS(in.Receipt)
	}

	return out
}

func TransactionTraceToEOS(in *pbcodec.TransactionTrace) (out *eos.TransactionTrace) {
	out = &eos.TransactionTrace{
		ID:              ChecksumToEOS(in.Id),
		BlockNum:        uint32(in.BlockNum),
		BlockTime:       TimestampToBlockTimestamp(in.BlockTime),
		ProducerBlockID: ChecksumToEOS(in.ProducerBlockId),
		Elapsed:         eos.Int64(in.Elapsed),
		NetUsage:        eos.Uint64(in.NetUsage),
		Scheduled:       in.Scheduled,
		ActionTraces:    ActionTracesToEOS(in.ActionTraces),
		Except:          ExceptionToEOS(in.Exception),
		ErrorCode:       ErrorCodeToEOS(in.ErrorCode),
	}

	if in.FailedDtrxTrace != nil {
		out.FailedDtrxTrace = TransactionTraceToEOS(in.FailedDtrxTrace)
	}
	if in.Receipt != nil {
		out.Receipt = TransactionReceiptHeaderToEOS(in.Receipt)
	}

	return out
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

func AuthoritiesToEOS(authority *pbcodec.Authority) eos.Authority {
	return eos.Authority{
		Threshold: authority.Threshold,
		Keys:      KeyWeightsToEOS(authority.Keys),
		Accounts:  PermissionLevelWeightsToEOS(authority.Accounts),
		Waits:     WaitWeightsToEOS(authority.Waits),
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

func WaitWeightsToEOS(waits []*pbcodec.WaitWeight) (out []eos.WaitWeight) {
	if len(waits) <= 0 {
		return nil
	}

	out = make([]eos.WaitWeight, len(waits))
	for i, o := range waits {
		out[i] = eos.WaitWeight{
			WaitSec: o.WaitSec,
			Weight:  uint16(o.Weight),
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

func PermissionLevelWeightsToEOS(weights []*pbcodec.PermissionLevelWeight) (out []eos.PermissionLevelWeight) {
	if len(weights) == 0 {
		return []eos.PermissionLevelWeight{}
	}

	out = make([]eos.PermissionLevelWeight, len(weights))
	for i, o := range weights {
		out[i] = eos.PermissionLevelWeight{
			Permission: PermissionLevelToEOS(o.Permission),
			Weight:     uint16(o.Weight),
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

func PermissionLevelToEOS(perm *pbcodec.PermissionLevel) eos.PermissionLevel {
	return eos.PermissionLevel{
		Actor:      eos.AccountName(perm.Actor),
		Permission: eos.PermissionName(perm.Permission),
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

func KeyWeightsToEOS(keys []*pbcodec.KeyWeight) (out []eos.KeyWeight) {
	if len(keys) <= 0 {
		return nil
	}

	out = make([]eos.KeyWeight, len(keys))
	for i, o := range keys {
		out[i] = eos.KeyWeight{
			PublicKey: ecc.MustNewPublicKey(o.PublicKey),
			Weight:    uint16(o.Weight),
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

func TransactionToEOS(trx *pbcodec.Transaction) *eos.Transaction {
	var contextFreeActions []*eos.Action
	if len(trx.ContextFreeActions) > 0 {
		contextFreeActions = make([]*eos.Action, len(trx.ContextFreeActions))
		for i, act := range trx.ContextFreeActions {
			contextFreeActions[i] = ActionToEOS(act)
		}
	}

	var actions []*eos.Action
	if len(trx.Actions) > 0 {
		actions = make([]*eos.Action, len(trx.Actions))
		for i, act := range trx.Actions {
			actions[i] = ActionToEOS(act)
		}
	}

	return &eos.Transaction{
		TransactionHeader:  *(TransactionHeaderToEOS(trx.Header)),
		ContextFreeActions: contextFreeActions,
		Actions:            actions,
		Extensions:         ExtensionsToEOS(trx.Extensions),
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

func TransactionHeaderToEOS(trx *pbcodec.TransactionHeader) *eos.TransactionHeader {
	out := &eos.TransactionHeader{
		Expiration:       TimestampToJSONTime(trx.Expiration),
		RefBlockNum:      uint16(trx.RefBlockNum),
		RefBlockPrefix:   uint32(trx.RefBlockPrefix),
		MaxNetUsageWords: eos.Varuint32(trx.MaxNetUsageWords),
		MaxCPUUsageMS:    uint8(trx.MaxCpuUsageMs),
		DelaySec:         eos.Varuint32(trx.DelaySec),
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

func SignedTransactionToEOS(trx *pbcodec.SignedTransaction) *eos.SignedTransaction {
	return &eos.SignedTransaction{
		Transaction:     TransactionToEOS(trx.Transaction),
		Signatures:      SignaturesToEOS(trx.Signatures),
		ContextFreeData: bytesSlicesToHexBytes(trx.ContextFreeData),
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

func ActionTracesToDEOS(actionTraces []eos.ActionTrace, opts ...conversionOption) (out []*pbcodec.ActionTrace, someConsoleTruncated bool) {
	if len(actionTraces) <= 0 {
		return nil, false
	}

	sort.Slice(actionTraces, func(i, j int) bool {
		leftSeq := uint64(math.MaxUint64)
		rightSeq := uint64(math.MaxUint64)

		if leftReceipt := actionTraces[i].Receipt; leftReceipt != nil {
			if seq := leftReceipt.GlobalSequence; seq != 0 {
				leftSeq = uint64(seq)
			}
		}
		if rightReceipt := actionTraces[j].Receipt; rightReceipt != nil {
			if seq := rightReceipt.GlobalSequence; seq != 0 {
				rightSeq = uint64(seq)
			}
		}

		return leftSeq < rightSeq
	})

	out = make([]*pbcodec.ActionTrace, len(actionTraces))
	var consoleTruncated bool
	for idx, actionTrace := range actionTraces {
		out[idx], consoleTruncated = ActionTraceToDEOS(actionTrace, uint32(idx), opts...)
		if consoleTruncated {
			someConsoleTruncated = true
		}
	}

	return
}

func ActionTracesToEOS(actionTraces []*pbcodec.ActionTrace) (out []eos.ActionTrace) {
	if len(actionTraces) <= 0 {
		return nil
	}

	out = make([]eos.ActionTrace, len(actionTraces))
	for i, actionTrace := range actionTraces {
		out[i] = ActionTraceToEOS(actionTrace)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ActionOrdinal < out[j].ActionOrdinal })

	return
}

func AuthSequenceToDEOS(in eos.TransactionTraceAuthSequence) *pbcodec.AuthSequence {
	return &pbcodec.AuthSequence{
		AccountName: string(in.Account),
		Sequence:    uint64(in.Sequence),
	}
}

func AuthSequenceListToEOS(in []*pbcodec.AuthSequence) (out []eos.TransactionTraceAuthSequence) {
	if len(in) == 0 {
		return []eos.TransactionTraceAuthSequence{}
	}

	out = make([]eos.TransactionTraceAuthSequence, len(in))
	for i, seq := range in {
		out[i] = AuthSequenceToEOS(seq)
	}

	return
}

func AuthSequenceToEOS(in *pbcodec.AuthSequence) eos.TransactionTraceAuthSequence {
	return eos.TransactionTraceAuthSequence{
		Account:  eos.AccountName(in.AccountName),
		Sequence: eos.Uint64(in.Sequence),
	}
}

func ActionTraceToDEOS(in eos.ActionTrace, execIndex uint32, opts ...conversionOption) (out *pbcodec.ActionTrace, consoleTruncated bool) {
	out = &pbcodec.ActionTrace{
		Receiver:             string(in.Receiver),
		Action:               ActionToDEOS(in.Action),
		Elapsed:              int64(in.Elapsed),
		Console:              string(in.Console),
		TransactionId:        in.TransactionID.String(),
		ContextFree:          in.ContextFree,
		ProducerBlockId:      in.ProducerBlockID.String(),
		BlockNum:             uint64(in.BlockNum),
		BlockTime:            mustProtoTimestamp(in.BlockTime.Time),
		AccountRamDeltas:     AccountRAMDeltasToDEOS(in.AccountRAMDeltas),
		Exception:            ExceptionToDEOS(in.Except),
		ActionOrdinal:        uint32(in.ActionOrdinal),
		CreatorActionOrdinal: uint32(in.CreatorActionOrdinal),
		ExecutionIndex:       execIndex,
		ErrorCode:            ErrorCodeToDEOS(in.ErrorCode),
	}
	out.ClosestUnnotifiedAncestorActionOrdinal = uint32(in.ClosestUnnotifiedAncestorActionOrdinal) // freaking long line, stay away from me

	if in.Receipt != nil {
		authSequences := in.Receipt.AuthSequence

		var deosAuthSequence []*pbcodec.AuthSequence
		if len(authSequences) > 0 {
			deosAuthSequence = make([]*pbcodec.AuthSequence, len(authSequences))
			for i, seq := range authSequences {
				deosAuthSequence[i] = AuthSequenceToDEOS(seq)
			}
		}

		out.Receipt = &pbcodec.ActionReceipt{
			Receiver:       string(in.Receipt.Receiver),
			Digest:         in.Receipt.ActionDigest.String(),
			GlobalSequence: uint64(in.Receipt.GlobalSequence),
			AuthSequence:   deosAuthSequence,
			RecvSequence:   uint64(in.Receipt.ReceiveSequence),
			CodeSequence:   uint64(in.Receipt.CodeSequence),
			AbiSequence:    uint64(in.Receipt.ABISequence),
		}
	}

	initialConsoleLength := len(in.Console)
	for _, opt := range opts {
		if v, ok := opt.(actionConversionOption); ok {
			v.apply(out)
		}
	}

	return out, initialConsoleLength != len(out.Console)
}

func ErrorCodeToDEOS(in *eos.Uint64) uint64 {
	if in != nil {
		return uint64(*in)
	}
	return 0
}

func ErrorCodeToEOS(in uint64) *eos.Uint64 {
	if in != 0 {
		val := eos.Uint64(in)
		return &val
	}
	return nil
}

func ActionTraceToEOS(in *pbcodec.ActionTrace) (out eos.ActionTrace) {
	out = eos.ActionTrace{
		Receiver:             eos.AccountName(in.Receiver),
		Action:               ActionToEOS(in.Action),
		Elapsed:              eos.Int64(in.Elapsed),
		Console:              eos.SafeString(in.Console),
		TransactionID:        ChecksumToEOS(in.TransactionId),
		ContextFree:          in.ContextFree,
		ProducerBlockID:      ChecksumToEOS(in.ProducerBlockId),
		BlockNum:             uint32(in.BlockNum),
		BlockTime:            TimestampToBlockTimestamp(in.BlockTime),
		AccountRAMDeltas:     AccountRAMDeltasToEOS(in.AccountRamDeltas),
		Except:               ExceptionToEOS(in.Exception),
		ActionOrdinal:        eos.Varuint32(in.ActionOrdinal),
		CreatorActionOrdinal: eos.Varuint32(in.CreatorActionOrdinal),
		ErrorCode:            ErrorCodeToEOS(in.ErrorCode),
	}
	out.ClosestUnnotifiedAncestorActionOrdinal = eos.Varuint32(in.ClosestUnnotifiedAncestorActionOrdinal) // freaking long line, stay away from me

	if in.Receipt != nil {
		receipt := in.Receipt

		out.Receipt = &eos.ActionTraceReceipt{
			Receiver:        eos.AccountName(receipt.Receiver),
			ActionDigest:    ChecksumToEOS(receipt.Digest),
			GlobalSequence:  eos.Uint64(receipt.GlobalSequence),
			AuthSequence:    AuthSequenceListToEOS(receipt.AuthSequence),
			ReceiveSequence: eos.Uint64(receipt.RecvSequence),
			CodeSequence:    eos.Varuint32(receipt.CodeSequence),
			ABISequence:     eos.Varuint32(receipt.AbiSequence),
		}
	}

	return
}

func ChecksumToEOS(in string) eos.Checksum256 {
	out, err := hex.DecodeString(in)
	if err != nil {
		panic(fmt.Sprintf("failed decoding checksum %q: %s", in, err))
	}

	return eos.Checksum256(out)
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

func ActionToEOS(action *pbcodec.Action) (out *eos.Action) {
	d := eos.ActionData{}
	d.SetToServer(false) // rather, what we expect FROM `nodeos` servers

	d.HexData = eos.HexBytes(action.RawData)
	if len(action.JsonData) != 0 {
		err := json.Unmarshal([]byte(action.JsonData), &d.Data)
		if err != nil {
			panic(fmt.Sprintf("unmarshaling action json data %q: %s", action.JsonData, err))
		}
	}

	out = &eos.Action{
		Account:       eos.AccountName(action.Account),
		Name:          eos.ActionName(action.Name),
		Authorization: AuthorizationToEOS(action.Authorization),
		ActionData:    d,
	}

	return out
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

func AuthorizationToEOS(authorization []*pbcodec.PermissionLevel) (out []eos.PermissionLevel) {
	if len(authorization) == 0 {
		return []eos.PermissionLevel{}
	}

	out = make([]eos.PermissionLevel, len(authorization))
	for i, permission := range authorization {
		out[i] = PermissionLevelToEOS(permission)
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

func AccountRAMDeltasToEOS(deltas []*pbcodec.AccountRAMDelta) (out []*eos.AccountRAMDelta) {
	if len(deltas) == 0 {
		return []*eos.AccountRAMDelta{}
	}

	out = make([]*eos.AccountRAMDelta, len(deltas))
	for i, delta := range deltas {
		out[i] = &eos.AccountRAMDelta{
			Account: eos.AccountName(delta.Account),
			Delta:   eos.Int64(delta.Delta),
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

func ExceptionToEOS(in *pbcodec.Exception) *eos.Except {
	if in == nil {
		return nil
	}
	out := &eos.Except{
		Code:    eos.Int64(in.Code),
		Name:    in.Name,
		Message: in.Message,
	}

	if len(in.Stack) > 0 {
		out.Stack = make([]*eos.ExceptLogMessage, len(in.Stack))
		for i, el := range in.Stack {
			msg := &eos.ExceptLogMessage{
				Format: el.Format,
			}

			ctx := LogContextToEOS(el.Context)
			if ctx != nil {
				msg.Context = *ctx
			}

			if len(el.Data) > 0 {
				msg.Data = json.RawMessage(el.Data)
			}

			out.Stack[i] = msg
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

func LogContextToEOS(in *pbcodec.Exception_LogContext) *eos.ExceptLogContext {
	if in == nil {
		return nil
	}

	var exceptLevel eos.ExceptLogLevel
	exceptLevel.FromString(in.Level)

	return &eos.ExceptLogContext{
		Level:      exceptLevel,
		File:       in.File,
		Line:       uint64(in.Line),
		Method:     in.Method,
		Hostname:   in.Hostname,
		ThreadName: in.ThreadName,
		Timestamp:  TimestampToJSONTime(in.Timestamp),
		Context:    LogContextToEOS(in.Context),
	}
}

func TimestampToJSONTime(in *timestamp.Timestamp) eos.JSONTime {
	out, _ := ptypes.Timestamp(in)
	return eos.JSONTime{Time: out}
}

func TimestampToBlockTimestamp(in *timestamp.Timestamp) eos.BlockTimestamp {
	out, _ := ptypes.Timestamp(in)
	return eos.BlockTimestamp{Time: out}
}

func dbOpPathQuad(path string) (code string, scope string, table string, primaryKey string) {
	chunks := strings.Split(path, "/")
	if len(chunks) != 4 {
		panic("received db operation with a path with less than 4 '/'-separated chunks")
	}

	return chunks[0], chunks[1], chunks[2], chunks[3]
}

func tableOpPathQuad(path string) (code string, scope string, table string) {
	chunks := strings.Split(path, "/")
	if len(chunks) != 3 {
		panic("received db operation with a path with less than 3 '/'-separated chunks")
	}

	return chunks[0], chunks[1], chunks[2]
}

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}

func mustTimestamp(in *timestamp.Timestamp) time.Time {
	out, err := ptypes.Timestamp(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
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

func TransactionStatusToEOS(in pbcodec.TransactionStatus) eos.TransactionStatus {
	switch in {
	case pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED:
		return eos.TransactionStatusExecuted
	case pbcodec.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL:
		return eos.TransactionStatusSoftFail
	case pbcodec.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL:
		return eos.TransactionStatusHardFail
	case pbcodec.TransactionStatus_TRANSACTIONSTATUS_DELAYED:
		return eos.TransactionStatusDelayed
	case pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXPIRED:
		return eos.TransactionStatusExpired
	default:
		return eos.TransactionStatusUnknown
	}
}

func ExtractEOSSignedTransactionFromReceipt(trxReceipt *pbcodec.TransactionReceipt) (*eos.SignedTransaction, error) {
	eosPackedTx, err := pbcodecPackedTransactionToEOS(trxReceipt.PackedTransaction)
	if err != nil {
		return nil, fmt.Errorf("pbcodec.PackedTransaction to EOS conversion failed: %s", err)
	}

	signedTransaction, err := eosPackedTx.UnpackBare()
	if err != nil {
		return nil, fmt.Errorf("unable to unpack packed transaction: %s", err)
	}

	return signedTransaction, nil
}

// Best effort to extract public keys from a signed transaction
func GetPublicKeysFromSignedTransaction(chainID eos.Checksum256, signedTransaction *eos.SignedTransaction) []string {
	eccPublicKeys, err := signedTransaction.SignedByKeys(chainID)
	if err != nil {
		// We discard any errors and simply return an empty array
		return nil
	}

	publicKeys := make([]string, len(eccPublicKeys))
	for i, eccPublicKey := range eccPublicKeys {
		publicKeys[i] = eccPublicKey.String()
	}

	return publicKeys
}

func pbcodecPackedTransactionToEOS(packedTrx *pbcodec.PackedTransaction) (*eos.PackedTransaction, error) {
	signatures := make([]ecc.Signature, len(packedTrx.Signatures))
	for i, signature := range packedTrx.Signatures {
		eccSignature, err := ecc.NewSignature(signature)
		if err != nil {
			return nil, err
		}

		signatures[i] = eccSignature
	}

	return &eos.PackedTransaction{
		Signatures:            signatures,
		Compression:           eos.CompressionType(packedTrx.Compression),
		PackedContextFreeData: packedTrx.PackedContextFreeData,
		PackedTransaction:     packedTrx.PackedTransaction,
	}, nil
}
