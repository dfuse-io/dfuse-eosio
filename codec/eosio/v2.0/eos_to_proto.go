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

func TransactionTraceToDEOS(logger *zap.Logger, in *eos.TransactionTrace, opts ...eosio.ConversionOption) *pbcodec.TransactionTrace {
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

func ActionTracesToDEOS(actionTraces []eos.ActionTrace, opts ...eosio.ConversionOption) (out []*pbcodec.ActionTrace, someConsoleTruncated bool) {
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

func ActionTraceToDEOS(in eos.ActionTrace, execIndex uint32, opts ...eosio.ConversionOption) (out *pbcodec.ActionTrace, consoleTruncated bool) {
	out = &pbcodec.ActionTrace{
		Receiver:             string(in.Receiver),
		Action:               eosio.ActionToDEOS(in.Action),
		Elapsed:              int64(in.Elapsed),
		Console:              string(in.Console),
		TransactionId:        in.TransactionID.String(),
		ContextFree:          in.ContextFree,
		ProducerBlockId:      in.ProducerBlockID.String(),
		BlockNum:             uint64(in.BlockNum),
		BlockTime:            mustProtoTimestamp(in.BlockTime.Time),
		AccountRamDeltas:     eosio.AccountRAMDeltasToDEOS(in.AccountRAMDeltas),
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
