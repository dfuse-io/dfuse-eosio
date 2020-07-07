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

package eosws

import (
	"context"
	"encoding/json"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	_ "github.com/eoscanada/eos-go/forum"
	"github.com/golang/protobuf/ptypes"
)

func (ws *WSConn) onGetActionTraces(ctx context.Context, msg *wsmsg.GetActionTraces) {
	authReq, ok := ws.AuthorizeRequest(ctx, msg)
	if !ok {
		return
	}

	// backwards compatibility:
	if msg.Data.Account != "" {
		msg.Data.Accounts = string(msg.Data.Account)
	}
	if msg.Data.Receiver != "" {
		msg.Data.Receivers = string(msg.Data.Receiver)
	}
	if msg.Data.ActionName != "" {
		msg.Data.ActionNames = string(msg.Data.ActionName)
	}

	// Support multiple things
	targetAccounts := mapString(msg.Data.Accounts)
	targetReceivers := targetAccounts
	if msg.Data.Receivers != "" {
		targetReceivers = mapString(msg.Data.Receivers)
	}
	targetActions := mapString(msg.Data.ActionNames)

	handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		blk := block.ToNative().(*pbcodec.Block)

		for _, trx := range blk.TransactionTraces {
			if trx.Receipt == nil || trx.Receipt.Status != pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED {
				// We do **not** stream transaction for that are not properly executed
				continue
			}

			allActions := trx.ActionTraces
			for actIdx, act := range allActions {
				if !targetReceivers[string(act.Receiver)] {
					continue
				}

				if !targetAccounts[string(act.Action.Account)] {
					continue
				}

				if msg.Data.ActionNames != "" && !targetActions[string(act.Action.Name)] {
					continue
				}

				rawTrace, err := mdl.ToV1ActionTraceRaw(act, allActions, msg.Data.WithInlineTraces)
				if err != nil {
					return err
				}

				out := wsmsg.NewActionTrace(trx.Id, actIdx /* , act.Depth() */, json.RawMessage(rawTrace))
				out.Data.BlockNum = blk.Number
				out.Data.BlockID = blk.Id
				stamp, _ := ptypes.Timestamp(blk.Header.Timestamp)
				out.Data.BlockTime = stamp

				if msg.Data.WithRAMOps {
					out.Data.RAMOps = mdl.ToV0RAMOps(trx.RAMOpsForAction(act.ExecutionIndex))
				}
				// TODO: we want that to reflect `eosws-go`'s `DTrxOp`
				if msg.Data.WithDTrxOps {
					out.Data.DTrxOps = mdl.ToV0DTrxOps(trx.DtrxOpsForAction(act.ExecutionIndex))
				}
				// TODO: we need to do JSON encoding with the ABI
				// valid here (?)  TODO: we need to rewrite the
				// `DBOps` because we don't want to send them in the
				// `hlog.DBOp` format, but in the `eosws-go:v1.DBOp`
				// format.
				if msg.Data.WithDBOps {
					out.Data.DBOps = mdl.ToV0DBOps(trx.DBOpsForAction(act.ExecutionIndex))
				}

				if msg.Data.WithTableOps {
					out.Data.TableOps = mdl.ToV0TableOps(trx.TableOpsForAction(act.ExecutionIndex))
				}

				metrics.DocumentResponseCounter.Inc()
				ws.EmitReply(ctx, msg, out)
			}
		}

		return nil
	})

	irrID, err := ws.irreversibleFinder.IrreversibleIDAtBlockNum(ctx, authReq.StartBlockNum)
	if err != nil {
		ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to retrieve irreversibility"))
		return
	}

	if freq := msg.WithProgress; freq != 0 {
		progHandler := NewProgressHandler(handler, ws, msg, ctx)
		if msg.IrreversibleOnly {
			progHandler.SetStepFilter(forkable.StepIrreversible)
		}
		handler = progHandler.ProcessBlock
	}

	if msg.IrreversibleOnly {
		forkableHandler := forkable.New(handler, forkable.WithLogger(zlog), forkable.WithFilters(forkable.StepIrreversible))
		handler = forkableHandler.ProcessBlock
	}

	blocknumGate := bstream.NewBlockNumGate(uint64(authReq.StartBlockNum), bstream.GateInclusive, handler, bstream.GateOptionWithLogger(zlog))
	metrics.IncListeners("get_action_traces")
	irrRef := bstream.BlockRefFromID(irrID)
	source := ws.subscriptionHub.NewSourceFromBlockNumWithOpts(irrRef.Num(), blocknumGate, bstream.JoiningSourceTargetBlockID(irrRef.ID()), bstream.JoiningSourceRateLimit(300, ws.filesourceBlockRateLimit))

	source.OnTerminating(func(_ error) {
		metrics.CurrentListeners.Dec("get_action_traces")
	})
	err = ws.RegisterListener(ctx, msg.ReqID, func() error {
		source.Shutdown(nil)
		return nil
	})
	if err != nil {
		source.Shutdown(nil) // important to ensure that OnRunFunc is run
		ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to register listener to ws connection"))
		return
	}

	ws.EmitReply(ctx, msg, wsmsg.NewListening(authReq.StartBlockNum))
	go source.Run()
}
