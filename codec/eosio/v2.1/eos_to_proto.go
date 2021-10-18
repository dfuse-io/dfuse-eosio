package eosio

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/dfuse-io/dfuse-eosio/codec/eosio"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/zap"
)

func TransactionReceiptToDEOS(txReceipt *TransactionReceipt) *pbcodec.TransactionReceipt {
	receipt := &pbcodec.TransactionReceipt{
		Status:               eosio.TransactionStatusToDEOS(txReceipt.Status),
		CpuUsageMicroSeconds: txReceipt.CPUUsageMicroSeconds,
		NetUsageWords:        uint32(txReceipt.NetUsageWords),
	}

	switch txReceipt.Transaction.TypeID {
	case TransactionVariant.TypeID("transaction_id"):
		receipt.Id = txReceipt.Transaction.Impl.(eos.Checksum256).String()

	case TransactionVariant.TypeID("packed_transaction"):
		packed := txReceipt.Transaction.Impl.(*PackedTransaction)

		receipt.PackedTransaction = PackedTransactionToDEOS(packed)

	default:
		id, name, _ := txReceipt.Transaction.Obtain(TransactionVariant)
		panic(fmt.Errorf("Transaction variant %q (%d) is unknown", name, id))
	}

	return receipt
}

func PackedTransactionToDEOS(in *PackedTransaction) *pbcodec.PackedTransaction {
	out := &pbcodec.PackedTransaction{
		Compression:       uint32(in.Compression),
		PackedTransaction: in.PackedTransaction,
	}

	switch in.PrunableData.TypeID {
	case PrunableDataVariant.TypeID("full_legacy"):
		fullLegacy := in.PrunableData.Impl.(*PackedTransactionPrunableFullLegacy)

		out.Signatures = eosio.SignaturesToDEOS(fullLegacy.Signatures)
		out.PackedContextFreeData = fullLegacy.PackedContextFreeData

	case PrunableDataVariant.TypeID("full"):
		panic(fmt.Errorf("Only full_legacy pruning state is supported right now, got full"))
		// full := in.PrunableData.Impl.(*PackedTransactionPrunableFull)
		// out.Signatures = eosio.SignaturesToDEOS(full.Signatures)

	case PrunableDataVariant.TypeID("partial"):
		panic(fmt.Errorf("Only full_legacy pruning state is supported right now, got partial"))
		// partial := in.PrunableData.Impl.(*PackedTransactionPrunablePartial)
		// out.Signatures = eosio.SignaturesToDEOS(partial.Signatures)

	case PrunableDataVariant.TypeID("none"):
		panic(fmt.Errorf("Only full_legacy pruning state is supported right now, got none"))

	default:
		id, name, _ := in.PrunableData.Obtain(PrunableDataVariant)
		panic(fmt.Errorf("PrunableData variant %q (%d) is unknown", name, id))
	}

	return out
}

func TransactionTraceToDEOS(logger *zap.Logger, in *TransactionTrace, opts ...eosio.ConversionOption) *pbcodec.TransactionTrace {
	id := in.ID.String()

	out := &pbcodec.TransactionTrace{
		Id:              id,
		BlockNum:        uint64(in.BlockNum),
		BlockTime:       mustProtoTimestamp(in.BlockTime.Time),
		ProducerBlockId: in.ProducerBlockID.String(),
		Elapsed:         int64(in.Elapsed),
		NetUsage:        uint64(in.NetUsage),
		Scheduled:       in.Scheduled,
		Exception:       eosio.ExceptionToDEOS(in.Except),
		ErrorCode:       eosio.ErrorCodeToDEOS(in.ErrorCode),
	}

	var someConsoleTruncated bool
	out.ActionTraces, someConsoleTruncated = ActionTracesToDEOS(in.ActionTraces, opts...)
	if someConsoleTruncated {
		logger.Info("transaction had some of its action trace's console entries truncated", zap.String("id", id))
	}

	if in.FailedDtrxTrace != nil {
		out.FailedDtrxTrace = TransactionTraceToDEOS(logger, in.FailedDtrxTrace, opts...)
	}
	if in.Receipt != nil {
		out.Receipt = eosio.TransactionReceiptHeaderToDEOS(in.Receipt)
	}

	return out
}

func ActionTracesToDEOS(actionTraces []*ActionTrace, opts ...eosio.ConversionOption) (out []*pbcodec.ActionTrace, someConsoleTruncated bool) {
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

func ActionTraceToDEOS(in *ActionTrace, execIndex uint32, opts ...eosio.ConversionOption) (out *pbcodec.ActionTrace, consoleTruncated bool) {
	out = &pbcodec.ActionTrace{
		Receiver:             string(in.Receiver),
		Action:               eosio.ActionToDEOS(in.Action),
		Elapsed:              int64(in.ElapsedUs),
		Console:              string(in.Console),
		TransactionId:        in.TransactionID.String(),
		ContextFree:          in.ContextFree,
		ProducerBlockId:      in.ProducerBlockID.String(),
		BlockNum:             uint64(in.BlockNum),
		BlockTime:            mustProtoTimestamp(in.BlockTime.Time),
		AccountRamDeltas:     AccountRAMDeltasToDEOS(in.AccountRAMDeltas),
		AccountDiskDeltas:    AccountDeltasToDEOS(in.AccountDiskDeltas),
		Exception:            eosio.ExceptionToDEOS(in.Except),
		ActionOrdinal:        uint32(in.ActionOrdinal),
		CreatorActionOrdinal: uint32(in.CreatorActionOrdinal),
		ExecutionIndex:       execIndex,
		ErrorCode:            eosio.ErrorCodeToDEOS(in.ErrorCode),
	}
	out.ClosestUnnotifiedAncestorActionOrdinal = uint32(in.ClosestUnnotifiedAncestorActionOrdinal) // freaking long line, stay away from me

	if in.Receipt != nil {
		out.Receipt = eosio.ActionTraceReceiptToDEOS(in.Receipt)
	}

	initialConsoleLength := len(in.Console)
	for _, opt := range opts {
		if v, ok := opt.(eosio.ActionConversionOption); ok {
			v.Apply(out)
		}
	}

	return out, initialConsoleLength != len(out.Console)
}

func AccountDeltasToDEOS(deltas []AccountDelta) (out []*pbcodec.AccountDelta) {
	if len(deltas) <= 0 {
		return nil
	}

	out = make([]*pbcodec.AccountDelta, len(deltas))
	for i, delta := range deltas {
		out[i] = &pbcodec.AccountDelta{
			Account: string(delta.Account),
			Delta:   int64(delta.Delta),
		}
	}
	return
}

func AccountRAMDeltasToDEOS(deltas []AccountDelta) (out []*pbcodec.AccountRAMDelta) {
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

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}
