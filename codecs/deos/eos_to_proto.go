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

package deos

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func ActivatedProtocolFeaturesToDEOS(in map[string][]eos.HexBytes) *pbdeos.ActivatedProtocolFeatures {
	out := &pbdeos.ActivatedProtocolFeatures{}
	out.ProtocolFeatures = hexBytesToBytesSlices(in["protocol_features"])
	return out
}

func PendingScheduleToDEOS(in *eos.PendingSchedule) *pbdeos.PendingProducerSchedule {
	out := &pbdeos.PendingProducerSchedule{
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

func ProducerToLastProducedToDEOS(in [][2]eos.EOSNameOrUint32) []*pbdeos.ProducerToLastProduced {
	out := []*pbdeos.ProducerToLastProduced{}
	for _, elem := range in {
		out = append(out, &pbdeos.ProducerToLastProduced{
			Name:                 elem[0].(string),
			LastBlockNumProduced: uint32(elem[1].(float64)),
		})
	}
	return out
}

func ProducerToLastImpliedIrbToDEOS(in [][2]eos.EOSNameOrUint32) []*pbdeos.ProducerToLastImpliedIRB {
	out := []*pbdeos.ProducerToLastImpliedIRB{}
	for _, elem := range in {
		out = append(out, &pbdeos.ProducerToLastImpliedIRB{
			Name:                 elem[0].(string),
			LastBlockNumProduced: uint32(elem[1].(float64)),
		})
	}
	return out
}

func BlockrootMerkleToDEOS(merkle *eos.MerkleRoot) *pbdeos.BlockRootMerkle {
	return &pbdeos.BlockRootMerkle{
		NodeCount:   merkle.NodeCount,
		ActiveNodes: mustHexStringArrayToBytesArray(merkle.ActiveNodes),
	}
}

func hexBytesToBytesSlices(in []eos.HexBytes) [][]byte {
	out := [][]byte{}
	for _, s := range in {
		out = append(out, []byte(s))
	}
	return out
}

func bytesSlicesToHexBytes(in [][]byte) []eos.HexBytes {
	out := []eos.HexBytes{}
	for _, s := range in {
		out = append(out, []byte(s))
	}
	return out
}

func mustHexStringArrayToBytesArray(in []string) [][]byte {
	out := [][]byte{}
	for _, s := range in {
		b, err := hex.DecodeString(s)
		if err != nil {
			panic("invalid hex string")
		}
		out = append(out, b)
	}
	return out
}

func BlockHeaderToDEOS(blockHeader *eos.BlockHeader) *pbdeos.BlockHeader {
	out := &pbdeos.BlockHeader{
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

func BlockHeaderToEOS(in *pbdeos.BlockHeader) *eos.BlockHeader {
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

func BlockSigningAuthorityToDEOS(authority *eos.BlockSigningAuthority) *pbdeos.BlockSigningAuthority {
	out := &pbdeos.BlockSigningAuthority{}
	err := authority.DoFor(map[uint32]eos.OnVariant{
		eos.BlockSigningAuthorityV0Type: func(impl interface{}) error {
			v := impl.(*eos.BlockSigningAuthorityV0)

			out.Variant = &pbdeos.BlockSigningAuthority_V0{
				V0: &pbdeos.BlockSigningAuthorityV0{
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

func BlockSigningAuthorityToEOS(in *pbdeos.BlockSigningAuthority) *eos.BlockSigningAuthority {
	out := &eos.BlockSigningAuthority{}
	switch v := in.Variant.(type) {
	case *pbdeos.BlockSigningAuthority_V0:
		out.TypeID = eos.BlockSigningAuthorityV0Type
		out.Impl = &eos.BlockSigningAuthorityV0{
			Threshold: v.V0.Threshold,
		}

		return out
	}

	panic(fmt.Errorf("unknown block signing authority variant %t", in.Variant))
}

func ProducerScheduleToDEOS(e *eos.ProducerSchedule) *pbdeos.ProducerSchedule {
	return &pbdeos.ProducerSchedule{
		Version:   uint32(e.Version),
		Producers: ProducerKeysToDEOS(e.Producers),
	}
}

func ProducerScheduleToEOS(in *pbdeos.ProducerSchedule) *eos.ProducerSchedule {
	return &eos.ProducerSchedule{
		Version:   in.Version,
		Producers: ProducerKeysToEOS(in.Producers),
	}
}

func ProducerAuthorityScheduleToDEOS(e *eos.ProducerAuthoritySchedule) *pbdeos.ProducerAuthoritySchedule {
	return &pbdeos.ProducerAuthoritySchedule{
		Version:   uint32(e.Version),
		Producers: ProducerAuthoritiesToDEOS(e.Producers),
	}
}

func ProducerAuthorityScheduleToEOS(in *pbdeos.ProducerAuthoritySchedule) *eos.ProducerAuthoritySchedule {
	return &eos.ProducerAuthoritySchedule{
		Version:   in.Version,
		Producers: ProducerAuthoritiesToEOS(in.Producers),
	}
}

func ProducerKeysToDEOS(in []eos.ProducerKey) (out []*pbdeos.ProducerKey) {
	for _, key := range in {
		out = append(out, &pbdeos.ProducerKey{
			AccountName:     string(key.AccountName),
			BlockSigningKey: key.BlockSigningKey.String(),
		})
	}
	return
}

func ProducerKeysToEOS(in []*pbdeos.ProducerKey) (out []eos.ProducerKey) {
	for _, producer := range in {
		key, _ := ecc.NewPublicKey(producer.BlockSigningKey)
		// panic?
		eosProducer := eos.ProducerKey{
			AccountName:     eos.AccountName(producer.AccountName),
			BlockSigningKey: key,
		}
		out = append(out, eosProducer)
	}
	return
}

func PublicKeysToEOS(in []string) (out []*ecc.PublicKey) {
	for _, inkey := range in {
		key, _ := ecc.NewPublicKey(inkey)
		out = append(out, &key)
	}
	return
}

func ExtensionsToDEOS(in []*eos.Extension) (out []*pbdeos.Extension) {
	for _, extension := range in {
		out = append(out, &pbdeos.Extension{
			Type: uint32(extension.Type),
			Data: extension.Data,
		})
	}

	return
}

func ExtensionsToEOS(in []*pbdeos.Extension) (out []*eos.Extension) {
	for _, extension := range in {
		out = append(out, &eos.Extension{
			Type: uint16(extension.Type),
			Data: extension.Data,
		})
	}
	return
}

func ProducerAuthoritiesToDEOS(producerAuthorities []*eos.ProducerAuthority) (out []*pbdeos.ProducerAuthority) {
	for _, authority := range producerAuthorities {
		deosProducer := pbdeos.ProducerAuthority{
			AccountName:           string(authority.AccountName),
			BlockSigningAuthority: BlockSigningAuthorityToDEOS(authority.BlockSigningAuthority),
		}
		out = append(out, &deosProducer)
	}
	return
}

func ProducerAuthoritiesToEOS(producerAuthorities []*pbdeos.ProducerAuthority) (out []*eos.ProducerAuthority) {
	for _, authority := range producerAuthorities {
		out = append(out, &eos.ProducerAuthority{
			AccountName:           eos.AccountName(authority.AccountName),
			BlockSigningAuthority: BlockSigningAuthorityToEOS(authority.BlockSigningAuthority),
		})
	}
	return
}

func TransactionReceiptToDEOS(txReceipt *eos.TransactionReceipt) *pbdeos.TransactionReceipt {
	receipt := &pbdeos.TransactionReceipt{
		Status:               TransactionStatusToDEOS(txReceipt.Status),
		CpuUsageMicroSeconds: txReceipt.CPUUsageMicroSeconds,
		NetUsageWords:        uint32(txReceipt.NetUsageWords),
	}

	receipt.Id = txReceipt.Transaction.ID.String()
	if txReceipt.Transaction.Packed != nil {
		receipt.PackedTransaction = &pbdeos.PackedTransaction{
			Signatures:            SignaturesToDEOS(txReceipt.Transaction.Packed.Signatures),
			Compression:           uint32(txReceipt.Transaction.Packed.Compression),
			PackedContextFreeData: txReceipt.Transaction.Packed.PackedContextFreeData,
			PackedTransaction:     txReceipt.Transaction.Packed.PackedTransaction,
		}
	}

	return receipt
}

func TransactionReceiptHeaderToDEOS(in *eos.TransactionReceiptHeader) *pbdeos.TransactionReceiptHeader {
	return &pbdeos.TransactionReceiptHeader{
		Status:               TransactionStatusToDEOS(in.Status),
		CpuUsageMicroSeconds: in.CPUUsageMicroSeconds,
		NetUsageWords:        uint32(in.NetUsageWords),
	}
}

func TransactionReceiptHeaderToEOS(in *pbdeos.TransactionReceiptHeader) *eos.TransactionReceiptHeader {
	return &eos.TransactionReceiptHeader{
		Status:               TransactionStatusToEOS(in.Status),
		CPUUsageMicroSeconds: in.CpuUsageMicroSeconds,
		NetUsageWords:        eos.Varuint32(in.NetUsageWords),
	}
}

func SignaturesToDEOS(in []ecc.Signature) (out []string) {
	for _, signature := range in {
		out = append(out, signature.String())
	}
	return
}

func SignaturesToEOS(in []string) []ecc.Signature {
	out := []ecc.Signature{}
	for _, signature := range in {
		sig, err := ecc.NewSignature(signature)
		if err != nil {
			panic(fmt.Sprintf("failed to read signature %q: %s", signature, err))
		}
		out = append(out, sig)
	}
	return out
}

func TransactionTraceToDEOS(in *eos.TransactionTrace) *pbdeos.TransactionTrace {
	id := in.ID.String()

	out := &pbdeos.TransactionTrace{
		Id:              id,
		BlockNum:        uint64(in.BlockNum),
		BlockTime:       mustProtoTimestamp(in.BlockTime.Time),
		ProducerBlockId: in.ProducerBlockID.String(),
		Elapsed:         int64(in.Elapsed),
		NetUsage:        uint64(in.NetUsage),
		Scheduled:       in.Scheduled,
		ActionTraces:    ActionTracesToDEOS(in.ActionTraces),
		Exception:       ExceptionToDEOS(in.Except),
		ErrorCode:       ErrorCodeToDEOS(in.ErrorCode),
	}

	if in.FailedDtrxTrace != nil {
		out.FailedDtrxTrace = TransactionTraceToDEOS(in.FailedDtrxTrace)
	}
	if in.Receipt != nil {
		out.Receipt = TransactionReceiptHeaderToDEOS(in.Receipt)
	}

	return out
}

func TransactionTraceToEOS(in *pbdeos.TransactionTrace) (out *eos.TransactionTrace) {
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

func PermissionToDEOS(perm *eos.Permission) *pbdeos.Permission {
	return &pbdeos.Permission{
		Name:         perm.PermName,
		Parent:       perm.Parent,
		RequiredAuth: AuthoritiesToDEOS(&perm.RequiredAuth),
	}
}

func AuthoritiesToDEOS(authority *eos.Authority) *pbdeos.Authority {
	return &pbdeos.Authority{
		Threshold: authority.Threshold,
		Keys:      KeyWeightToDEOS(authority.Keys),
		Accounts:  PermissionLevelWeightsToDEOS(authority.Accounts),
		Waits:     WaitWeightsToDEOS(authority.Waits),
	}
}

func WaitWeightsToDEOS(waits []eos.WaitWeight) (out []*pbdeos.WaitWeight) {
	for _, o := range waits {
		out = append(out, &pbdeos.WaitWeight{
			WaitSec: o.WaitSec,
			Weight:  uint32(o.Weight),
		})
	}
	return
}

func PermissionLevelWeightsToDEOS(weights []eos.PermissionLevelWeight) (out []*pbdeos.PermissionLevelWeight) {
	for _, o := range weights {
		out = append(out, &pbdeos.PermissionLevelWeight{
			Permission: PermissionLevelToDEOS(o.Permission),
			Weight:     uint32(o.Weight),
		})
	}
	return
}

func PermissionLevelWeightsToEOS(weights []*pbdeos.PermissionLevelWeight) (out []eos.PermissionLevelWeight) {
	if len(weights) == 0 {
		return []eos.PermissionLevelWeight{}
	}

	for _, o := range weights {
		out = append(out, eos.PermissionLevelWeight{
			Permission: PermissionLevelToEOS(o.Permission),
			Weight:     uint16(o.Weight),
		})
	}
	return
}

func PermissionLevelToDEOS(perm eos.PermissionLevel) *pbdeos.PermissionLevel {
	return &pbdeos.PermissionLevel{
		Actor:      string(perm.Actor),
		Permission: string(perm.Permission),
	}
}

func PermissionLevelToEOS(perm *pbdeos.PermissionLevel) eos.PermissionLevel {
	return eos.PermissionLevel{
		Actor:      eos.AccountName(perm.Actor),
		Permission: eos.PermissionName(perm.Permission),
	}
}

func KeyWeightToDEOS(keys []eos.KeyWeight) (out []*pbdeos.KeyWeight) {
	for _, o := range keys {
		out = append(out, &pbdeos.KeyWeight{
			PublicKey: o.PublicKey.String(),
			Weight:    uint32(o.Weight),
		})
	}
	return
}

func KeyWeightsPToDEOS(keys []*eos.KeyWeight) (out []*pbdeos.KeyWeight) {
	for _, o := range keys {
		out = append(out, &pbdeos.KeyWeight{
			PublicKey: o.PublicKey.String(),
			Weight:    uint32(o.Weight),
		})
	}
	return
}

func TransactionToDEOS(trx *eos.Transaction) *pbdeos.Transaction {

	var contextFreeActions []*pbdeos.Action
	for _, act := range trx.ContextFreeActions {
		contextFreeActions = append(contextFreeActions, ActionToDEOS(act))
	}
	var actions []*pbdeos.Action
	for _, act := range trx.Actions {
		actions = append(actions, ActionToDEOS(act))
	}

	return &pbdeos.Transaction{
		Header:             TransactionHeaderToDEOS(&trx.TransactionHeader),
		ContextFreeActions: contextFreeActions,
		Actions:            actions,
		Extensions:         ExtensionsToDEOS(trx.Extensions),
	}
}

func TransactionToEOS(trx *pbdeos.Transaction) *eos.Transaction {
	var contextFreeActions []*eos.Action
	for _, act := range trx.ContextFreeActions {
		contextFreeActions = append(contextFreeActions, ActionToEOS(act))
	}

	var actions []*eos.Action
	for _, act := range trx.Actions {
		actions = append(actions, ActionToEOS(act))
	}

	return &eos.Transaction{
		TransactionHeader:  *(TransactionHeaderToEOS(trx.Header)),
		ContextFreeActions: contextFreeActions,
		Actions:            actions,
		Extensions:         ExtensionsToEOS(trx.Extensions),
	}
}

func TransactionHeaderToDEOS(trx *eos.TransactionHeader) *pbdeos.TransactionHeader {
	out := &pbdeos.TransactionHeader{
		Expiration:       mustProtoTimestamp(trx.Expiration.Time),
		RefBlockNum:      uint32(trx.RefBlockNum),
		RefBlockPrefix:   trx.RefBlockPrefix,
		MaxNetUsageWords: uint32(trx.MaxNetUsageWords),
		MaxCpuUsageMs:    uint32(trx.MaxCPUUsageMS),
		DelaySec:         uint32(trx.DelaySec),
	}

	return out
}

func TransactionHeaderToEOS(trx *pbdeos.TransactionHeader) *eos.TransactionHeader {
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

func SignedTransactionToDEOS(trx *eos.SignedTransaction) *pbdeos.SignedTransaction {
	return &pbdeos.SignedTransaction{
		Transaction:     TransactionToDEOS(trx.Transaction),
		Signatures:      SignaturesToDEOS(trx.Signatures),
		ContextFreeData: hexBytesToBytesSlices(trx.ContextFreeData),
	}
}

func SignedTransactionToEOS(trx *pbdeos.SignedTransaction) *eos.SignedTransaction {
	return &eos.SignedTransaction{
		Transaction:     TransactionToEOS(trx.Transaction),
		Signatures:      SignaturesToEOS(trx.Signatures),
		ContextFreeData: bytesSlicesToHexBytes(trx.ContextFreeData),
	}
}

func CreationTreeToDEOS(tree CreationFlatTree) []*pbdeos.CreationFlatNode {
	var out []*pbdeos.CreationFlatNode
	for _, node := range tree {
		out = append(out, &pbdeos.CreationFlatNode{
			CreatorActionIndex:   int32(node[1]),
			ExecutionActionIndex: uint32(node[2]),
		})
	}
	return out
}

func ActionTracesToDEOS(actionTraces []eos.ActionTrace) (out []*pbdeos.ActionTrace) {
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

	for idx, actionTrace := range actionTraces {
		out = append(out, ActionTraceToDEOS(actionTrace, uint32(idx)))
	}

	return
}

func ActionTracesToEOS(actionTraces []*pbdeos.ActionTrace) (out []eos.ActionTrace) {
	sort.Slice(actionTraces, func(i, j int) bool { return actionTraces[i].ActionOrdinal < actionTraces[j].ActionOrdinal })

	for _, actionTrace := range actionTraces {
		out = append(out, ActionTraceToEOS(actionTrace))
	}

	return
}

func AuthSequenceToDEOS(in eos.TransactionTraceAuthSequence) *pbdeos.AuthSequence {
	return &pbdeos.AuthSequence{
		AccountName: string(in.Account),
		Sequence:    uint64(in.Sequence),
	}
}

func AuthSequenceListToEOS(in []*pbdeos.AuthSequence) (out []eos.TransactionTraceAuthSequence) {
	if len(in) == 0 {
		return []eos.TransactionTraceAuthSequence{}
	}

	for _, seq := range in {
		out = append(out, AuthSequenceToEOS(seq))
	}

	return
}

func AuthSequenceToEOS(in *pbdeos.AuthSequence) eos.TransactionTraceAuthSequence {
	return eos.TransactionTraceAuthSequence{
		Account:  eos.AccountName(in.AccountName),
		Sequence: eos.Uint64(in.Sequence),
	}
}

func ActionTraceToDEOS(in eos.ActionTrace, execIndex uint32) (out *pbdeos.ActionTrace) {
	out = &pbdeos.ActionTrace{
		Receiver:             string(in.Receiver),
		Action:               ActionToDEOS(in.Action),
		Elapsed:              int64(in.Elapsed),
		Console:              in.Console,
		TransactionId:        in.TransactionID.String(),
		ContextFree:          in.ContextFree,
		ProducerBlockId:      in.ProducerBlockID.String(),
		BlockNum:             uint64(in.BlockNum),
		BlockTime:            mustProtoTimestamp(in.BlockTime.Time),
		AccountRamDeltas:     AccountRAMDeltasToDEOS(in.AccountRAMDeltas),
		Exception:            ExceptionToDEOS(in.Except),
		ActionOrdinal:        in.ActionOrdinal,
		CreatorActionOrdinal: in.CreatorActionOrdinal,
		ExecutionIndex:       execIndex,
		ErrorCode:            ErrorCodeToDEOS(in.ErrorCode),
	}
	out.ClosestUnnotifiedAncestorActionOrdinal = in.ClosestUnnotifiedAncestorActionOrdinal // freaking long line, stay away from me

	if in.Receipt != nil {
		var deosAuthSequence []*pbdeos.AuthSequence
		for _, seq := range in.Receipt.AuthSequence {
			deosAuthSequence = append(deosAuthSequence, AuthSequenceToDEOS(seq))
		}
		out.Receipt = &pbdeos.ActionReceipt{
			Receiver:       string(in.Receipt.Receiver),
			Digest:         in.Receipt.ActionDigest,
			GlobalSequence: uint64(in.Receipt.GlobalSequence),
			AuthSequence:   deosAuthSequence,
			RecvSequence:   uint64(in.Receipt.ReceiveSequence),
			CodeSequence:   uint64(in.Receipt.CodeSequence),
			AbiSequence:    uint64(in.Receipt.ABISequence),
		}
	}

	return out
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

func ActionTraceToEOS(in *pbdeos.ActionTrace) (out eos.ActionTrace) {
	out = eos.ActionTrace{
		Receiver:             eos.AccountName(in.Receiver),
		Action:               ActionToEOS(in.Action),
		Elapsed:              eos.Int64(in.Elapsed),
		Console:              in.Console,
		TransactionID:        ChecksumToEOS(in.TransactionId),
		ContextFree:          in.ContextFree,
		ProducerBlockID:      ChecksumToEOS(in.ProducerBlockId),
		BlockNum:             uint32(in.BlockNum),
		BlockTime:            TimestampToBlockTimestamp(in.BlockTime),
		AccountRAMDeltas:     AccountRAMDeltasToEOS(in.AccountRamDeltas),
		Except:               ExceptionToEOS(in.Exception),
		ActionOrdinal:        uint32(in.ActionOrdinal),
		CreatorActionOrdinal: uint32(in.CreatorActionOrdinal),
		ErrorCode:            ErrorCodeToEOS(in.ErrorCode),
	}
	out.ClosestUnnotifiedAncestorActionOrdinal = uint32(in.ClosestUnnotifiedAncestorActionOrdinal) // freaking long line, stay away from me

	if in.Receipt != nil {
		receipt := in.Receipt

		out.Receipt = &eos.ActionTraceReceipt{
			Receiver:        eos.AccountName(receipt.Receiver),
			ActionDigest:    receipt.Digest,
			GlobalSequence:  eos.Uint64(receipt.GlobalSequence),
			AuthSequence:    AuthSequenceListToEOS(receipt.AuthSequence),
			ReceiveSequence: eos.Uint64(receipt.RecvSequence),
			CodeSequence:    eos.Uint64(receipt.CodeSequence),
			ABISequence:     eos.Uint64(receipt.AbiSequence),
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

func ActionToDEOS(action *eos.Action) *pbdeos.Action {
	deosAction := &pbdeos.Action{
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

func ActionToEOS(action *pbdeos.Action) (out *eos.Action) {
	d := eos.ActionData{}
	d.SetToServer(false) // rather, what we expect FROM `nodeos` servers

	if len(action.JsonData) != 0 {
		err := json.Unmarshal([]byte(action.JsonData), &d.Data)
		if err != nil {
			panic(fmt.Sprintf("unmarshaling action json data %q: %s", action.JsonData, err))
		}
		d.HexData = eos.HexBytes(action.RawData)
	} else {
		d.HexData = eos.HexBytes(action.RawData)
	}

	out = &eos.Action{
		Account:       eos.AccountName(action.Account),
		Name:          eos.ActionName(action.Name),
		Authorization: AuthorizationToEOS(action.Authorization),
		ActionData:    d,
	}

	return out
}

func AuthorizationToDEOS(authorization []eos.PermissionLevel) (out []*pbdeos.PermissionLevel) {
	for _, permission := range authorization {
		out = append(out, PermissionLevelToDEOS(permission))
	}
	return
}

func AuthorizationToEOS(authorization []*pbdeos.PermissionLevel) (out []eos.PermissionLevel) {
	if len(authorization) == 0 {
		return []eos.PermissionLevel{}
	}

	for _, permission := range authorization {
		out = append(out, PermissionLevelToEOS(permission))
	}
	return
}

func AccountRAMDeltasToDEOS(deltas []*eos.AccountRAMDelta) (out []*pbdeos.AccountRAMDelta) {
	for _, delta := range deltas {
		out = append(out, &pbdeos.AccountRAMDelta{
			Account: string(delta.Account),
			Delta:   int64(delta.Delta),
		})
	}
	return
}

func AccountRAMDeltasToEOS(deltas []*pbdeos.AccountRAMDelta) (out []*eos.AccountRAMDelta) {
	if len(deltas) == 0 {
		return []*eos.AccountRAMDelta{}
	}

	for _, delta := range deltas {
		out = append(out, &eos.AccountRAMDelta{
			Account: eos.AccountName(delta.Account),
			Delta:   eos.Int64(delta.Delta),
		})
	}
	return
}

func ExceptionToDEOS(in *eos.Except) *pbdeos.Exception {
	if in == nil {
		return nil
	}
	out := &pbdeos.Exception{
		Code:    int32(in.Code),
		Name:    in.Name,
		Message: in.Message,
	}

	for _, el := range in.Stack {
		msg := &pbdeos.Exception_LogMessage{
			Context: LogContextToDEOS(el.Context),
			Format:  el.Format,
			Data:    el.Data,
		}

		out.Stack = append(out.Stack, msg)
	}

	return out
}

func ExceptionToEOS(in *pbdeos.Exception) *eos.Except {
	if in == nil {
		return nil
	}
	out := &eos.Except{
		Code:    int(in.Code),
		Name:    in.Name,
		Message: in.Message,
	}

	for _, el := range in.Stack {
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

		out.Stack = append(out.Stack, msg)
	}

	return out
}

func LogContextToDEOS(in eos.ExceptLogContext) *pbdeos.Exception_LogContext {

	out := &pbdeos.Exception_LogContext{
		Level:      in.Level,
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

func LogContextToEOS(in *pbdeos.Exception_LogContext) *eos.ExceptLogContext {
	if in == nil {
		return nil
	}

	return &eos.ExceptLogContext{
		Level:      in.Level,
		File:       in.File,
		Line:       int(in.Line),
		Method:     in.Method,
		Hostname:   in.Hostname,
		ThreadName: in.ThreadName,
		Timestamp:  TimestampToJSONTime(in.Timestamp),
		Context:    LogContextToEOS(in.Context),
	}
}

func TimestampToJSONTime(in *timestamp.Timestamp) eos.JSONTime {
	out, _ := ptypes.Timestamp(in)
	// if err != nil {
	// 	panic(fmt.Sprintf("invalid timestamp JSONTime conversion %v: %s", in, err))
	// }
	return eos.JSONTime{Time: out}
}

func TimestampToBlockTimestamp(in *timestamp.Timestamp) eos.BlockTimestamp {
	out, _ := ptypes.Timestamp(in)
	// if err != nil {
	// 	panic(fmt.Sprintf("invalid timestamp BlockTimestamp conversion %v: %s", in, err))
	// }
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

func TransactionStatusToDEOS(in eos.TransactionStatus) pbdeos.TransactionStatus {
	switch in {
	case eos.TransactionStatusExecuted:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED
	case eos.TransactionStatusSoftFail:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL
	case eos.TransactionStatusHardFail:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL
	case eos.TransactionStatusDelayed:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED
	case eos.TransactionStatusExpired:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_EXPIRED
	default:
		return pbdeos.TransactionStatus_TRANSACTIONSTATUS_UNKNOWN
	}
}

func TransactionStatusToEOS(in pbdeos.TransactionStatus) eos.TransactionStatus {
	switch in {
	case pbdeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED:
		return eos.TransactionStatusExecuted
	case pbdeos.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL:
		return eos.TransactionStatusSoftFail
	case pbdeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL:
		return eos.TransactionStatusHardFail
	case pbdeos.TransactionStatus_TRANSACTIONSTATUS_DELAYED:
		return eos.TransactionStatusDelayed
	case pbdeos.TransactionStatus_TRANSACTIONSTATUS_EXPIRED:
		return eos.TransactionStatusExpired
	default:
		return eos.TransactionStatusUnknown
	}
}

func ExtractEOSSignedTransactionFromReceipt(trxReceipt *pbdeos.TransactionReceipt) (*eos.SignedTransaction, error) {
	eosPackedTx, err := pbdeosPackedTransactionToEOS(trxReceipt.PackedTransaction)
	if err != nil {
		return nil, fmt.Errorf("pbdeos.PackedTransaction to EOS conversion failed: %s", err)
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
		//zlog.Warn("unable to extract public keys from transaction: %s", zap.Error(err))
		return nil
	}

	publicKeys := make([]string, len(eccPublicKeys))
	for i, eccPublicKey := range eccPublicKeys {
		publicKeys[i] = eccPublicKey.String()
	}

	return publicKeys
}

func pbdeosPackedTransactionToEOS(packedTrx *pbdeos.PackedTransaction) (*eos.PackedTransaction, error) {
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
