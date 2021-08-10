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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/streamingfast/dtracing"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	"github.com/dfuse-io/logging"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (ws *WSConn) onGetTableRows(ctx context.Context, msg *wsmsg.GetTableRows) {
	scope := ""
	if msg.Data.Scope != nil {
		scope = string(*msg.Data.Scope)
	}

	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("handling get table rows stream",
		zap.String("account", string(msg.Data.Code)),
		zap.String("scope", string(scope)),
		zap.String("table_name", string(msg.Data.TableName)),
	)
	authReq, ok := ws.AuthorizeRequest(ctx, msg)
	if !ok {
		return
	}

	startBlockNum := authReq.StartBlockNum
	if msg.Fetch && authReq.IsFutureBlock {
		ws.EmitErrorReply(ctx, msg, AppTableRowsCannotFetchInFutureError(ctx, startBlockNum))
		return
	}

	fetchTableRows(ctx, ws, startBlockNum, msg, ws.abiGetter, ws, ws.stateClient, ws.irreversibleFinder)
}

func fetchTableRows(
	ctx context.Context,
	ws *WSConn,
	startBlockNum uint32,
	msg *wsmsg.GetTableRows,
	abiGetter ABIGetter,
	emitter Emitter,
	stateClient pbstatedb.StateClient,
	irrFinder IrreversibleFinder,
) {
	zlogger := logging.Logger(ctx, zlog)

	startBlockID := ""
	if msg.Fetch {
		spanContext, fetchSpan := dtracing.StartSpan(ctx, "fetch table rows")
		if msg.StartBlock == 0 {
			zlogger.Info("user requested start block 0, let statedb turns into head block instead of us doing it to prevent race condition")
			startBlockNum = 0
		}

		request := &pbstatedb.StreamTableRowsRequest{
			BlockNum: uint64(startBlockNum),
			Contract: string(msg.Data.Code),
			Table:    string(msg.Data.TableName),
			Scope:    string(*msg.Data.Scope),
			ToJson:   true,
		}

		zlogger.Info("requesting data from statedb", zap.Any("request", request))
		ref, snapshot, err := fetchStateTableRows(spanContext, stateClient, request)
		if err != nil {
			emitter.EmitErrorReply(ctx, msg, fmt.Errorf("fetch table rows: %w", err))
			fetchSpan.End()
			return
		}

		metrics.DocumentResponseCounter.Inc()
		emitter.EmitReply(spanContext, msg, snapshot)

		if ref.UpToBlock != nil {
			startBlockID = ref.UpToBlock.ID()
			startBlockNum = uint32(ref.UpToBlock.Num())
			zlogger.Info("state client response", zap.Stringer("up_to_block", ref.UpToBlock))
		}
		fetchSpan.End()
	}

	if msg.Listen {
		_, listenSpan := dtracing.StartSpan(ctx, "ws listen table rows")

		var err error

		var abiChangeHandler *ABIChangeHandler
		tableDeltaHandler := newTableDeltaHandler(ctx, msg, emitter, zlog, func() *eos.ABI {
			return abiChangeHandler.CurrentABI()
		})

		abiChangeHandler, err = NewABIChangeHandler(abiGetter, startBlockNum, msg.Data.Code, tableDeltaHandler, ctx)
		if err != nil {
			emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to retrieve abi"))
			return
		}

		var handler bstream.Handler
		handler = abiChangeHandler
		if freq := msg.WithProgress; freq != 0 {
			handler = NewProgressHandler(abiChangeHandler, emitter, msg, ctx)
		}

		var forkablePostGate bstream.Handler
		var irrID string
		if startBlockID != "" { //Flux return this ID
			irrID, err = irrFinder.IrreversibleIDAtBlockID(ctx, startBlockID)
			if err != nil {
				emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to retrieve irreversibility"))
				return
			}
			forkablePostGate = bstream.NewBlockIDGate(startBlockID, bstream.GateExclusive, handler, bstream.GateOptionWithLogger(zlog))
		} else {
			irrID, err = irrFinder.IrreversibleIDAtBlockNum(ctx, startBlockNum)
			if err != nil {
				emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to retrieve irreversibility"))
				return
			}
			forkablePostGate = bstream.NewBlockNumGate(uint64(startBlockNum), bstream.GateInclusive, handler, bstream.GateOptionWithLogger(zlog))
		}

		irrRef := bstream.NewBlockRefFromID(irrID)
		forkableHandler := forkable.New(forkablePostGate, forkable.WithLogger(zlog), forkable.WithExclusiveLIB(irrRef))

		metrics.IncListeners("get_table_rows")

		source := ws.subscriptionHub.NewSourceFromBlockNumWithOpts(irrRef.Num(), forkableHandler, bstream.JoiningSourceTargetBlockID(irrRef.ID()), bstream.JoiningSourceRateLimit(300, ws.filesourceBlockRateLimit))
		source.OnTerminating(func(_ error) {
			metrics.CurrentListeners.Dec("get_table_rows")
			listenSpan.End()
		})

		err = ws.RegisterListener(ctx, msg.ReqID, func() error {
			zlogger.Debug("fetchTableRows: canceller call", zap.String("req_id", msg.ReqID))
			source.Shutdown(nil)
			return nil
		})

		if err != nil {
			source.Shutdown(nil) // important to ensure that OnRunFunc is run
			emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to register listener to ws connection"))
			return
		}

		emitter.EmitReply(ctx, msg, wsmsg.NewListening(startBlockNum+1))
		go source.Run()
	}
}

func tableDeltasFromBlock(block *bstream.Block, msg *wsmsg.GetTableRows, abi *eos.ABI, step forkable.StepType, zlog *zap.Logger) []*wsmsg.TableDelta {
	zlog.Debug("about to stream table deltas from block", zap.Stringer("block", block), zap.Stringer("step", step))
	var deltas []*wsmsg.TableDelta

	blk := block.ToNative().(*pbcodec.Block)
	for _, trxTrace := range blk.TransactionTraces() {
		actionMatcher := blk.FilteringActionMatcher(trxTrace)

		for _, dbOp := range trxTrace.DbOps {
			if !actionMatcher.Matched(dbOp.ActionIndex) {
				continue
			}

			if dbOp.Code != string(msg.Data.Code) || dbOp.TableName != string(msg.Data.TableName) || dbOp.Scope != string(*msg.Data.Scope) {
				continue
			}

			v1DBOp := &v1.DBOp{
				Op:          strings.ToLower(dbOp.LegacyOperation()),
				Key:         dbOp.PrimaryKey,
				ActionIndex: int(dbOp.ActionIndex),
				Scope:       dbOp.Scope,
				Account:     dbOp.Code,
				Table:       dbOp.TableName,
			}

			if len(dbOp.OldData) != 0 {
				v1DBOp.Old = newDBRow(dbOp.OldData, msg.Data.TableName, abi, dbOp.OldPayer, msg.Data.JSON, zlog)
			}

			if len(dbOp.NewData) != 0 {
				v1DBOp.New = newDBRow(dbOp.NewData, msg.Data.TableName, abi, dbOp.NewPayer, msg.Data.JSON, zlog)
			}

			if step == forkable.StepUndo {
				oldRow := v1DBOp.Old
				v1DBOp.Old = v1DBOp.New
				v1DBOp.New = oldRow
				switch v1DBOp.Op {
				case "ins":
					v1DBOp.Op = "rem"
				case "rem":
					v1DBOp.Op = "ins"
				}
			}

			deltas = append(deltas,
				wsmsg.NewTableDelta(
					uint32(blk.Num()),
					v1DBOp,
					step,
				),
			)
		}
	}

	return deltas
}

func newDBRow(data []byte, tableName eos.TableName, abi *eos.ABI, payer string, needJSON bool, zlog *zap.Logger) *v1.DBRow {
	row := &v1.DBRow{
		Payer: payer,
	}

	if needJSON {
		rowData, err := abi.DecodeTableRow(tableName, data)
		if err == nil {
			row.JSON = json.RawMessage(rowData)
		} else {
			zlog.Error("couldn't decode row", zap.Error(err))
			row.Error = fmt.Sprintf("Couldn't json decode ROW: %s", err)
		}
	} else {
		row.Hex = hex.EncodeToString(data)
	}
	return row
}

func fetchStateTableRows(ctx context.Context, stateClient pbstatedb.StateClient, request *pbstatedb.StreamTableRowsRequest) (ref *pbstatedb.StreamReference, out *wsmsg.TableSnapshot, err error) {
	out = new(wsmsg.TableSnapshot)
	ref, err = pbstatedb.ForEachTableRows(ctx, stateClient, request, func(row *pbstatedb.TableRowResponse) error {
		out.Data.Rows = append(out.Data.Rows, []byte(row.Json))
		return nil
	})

	return
}

type tableDeltaHandler struct {
	msg        *wsmsg.GetTableRows
	emitter    Emitter
	ctx        context.Context
	zlog       *zap.Logger
	getABIFunc func() *eos.ABI
}

func newTableDeltaHandler(ctx context.Context, msg *wsmsg.GetTableRows, emitter Emitter, zlog *zap.Logger, getABIFunc func() *eos.ABI) *tableDeltaHandler {
	return &tableDeltaHandler{msg: msg, emitter: emitter, ctx: ctx, zlog: zlog, getABIFunc: getABIFunc}
}

func (h *tableDeltaHandler) ProcessBlock(block *bstream.Block, obj interface{}) error {
	fObj := obj.(*forkable.ForkableObject)

	if fObj.Step == forkable.StepNew || fObj.Step == forkable.StepUndo || fObj.Step == forkable.StepRedo {
		abi := h.getABIFunc()
		if abi == nil {
			return fmt.Errorf("expected a none nil abi")
		}

		deltas := tableDeltasFromBlock(block, h.msg, h.getABIFunc(), fObj.Step, h.zlog)

		for _, d := range deltas {
			metrics.DocumentResponseCounter.Inc()
			h.emitter.EmitReply(h.ctx, h.msg, d)
		}
	}

	return nil
}
