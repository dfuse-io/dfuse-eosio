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

	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dtracing"
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	fluxdb "github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/logging"
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

	fetchTableRows(ws, startBlockNum, msg, ws.abiGetter, ws, ws.fluxClient, ws.irreversibleFinder, ctx, ws.subscriptionHub)
}

func fetchTableRows(
	ws *WSConn,
	startBlockNum uint32,
	msg *wsmsg.GetTableRows,
	abiGetter ABIGetter,
	emitter Emitter,
	fluxClient fluxdb.Client,
	irrFinder IrreversibleFinder,
	ctx context.Context,
	hub *hub.SubscriptionHub,
) {
	zlogger := logging.Logger(ctx, zlog)

	startBlockID := ""
	if msg.Fetch {
		spanContext, fetchSpan := dtracing.StartSpan(ctx, "fetch table rows")
		if msg.StartBlock == 0 {
			zlogger.Info("user requested start block 0, let fluxdb turns into head block instead of us doing it to prevent race condition")
			startBlockNum = 0
		}

		request := fluxdb.NewGetTableRequest(msg.Data.Code, *msg.Data.Scope, msg.Data.TableName, "name")
		zlogger.Info("requesting data from fluxdb", zap.Uint32("start_block_num", startBlockNum), zap.Any("request", request))

		response, err := fluxClient.GetTable(spanContext, startBlockNum, request)
		if err != nil {
			emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "fluxdb client request failed"))
			fetchSpan.End()
			return
		}
		metrics.DocumentResponseCounter.Inc()
		emitter.EmitReply(ctx, msg, wsmsg.NewTableSnapshot(response.Rows))

		if response.UpToBlockNum != 0 {
			startBlockID = response.UpToBlockID
			startBlockNum = eos.BlockNum(startBlockID)
			zlogger.Info("Flux response", zap.Uint32("up_to_block_num", startBlockNum), zap.String("up_to_block_id", startBlockID))
		}
		fetchSpan.End()
	}

	if msg.Listen {
		_, listenSpan := dtracing.StartSpan(ctx, "ws listen table rows")

		var err error

		var abiChangeHandler *ABIChangeHandler
		tableDeltaHandler := NewTableDeltaHandler(msg, emitter, ctx, zlog, func() *eos.ABI {
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
			forkablePostGate = bstream.NewBlockIDGate(startBlockID, bstream.GateExclusive, handler)
		} else {
			irrID, err = irrFinder.IrreversibleIDAtBlockNum(ctx, startBlockNum)
			if err != nil {
				emitter.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to retrieve irreversibility"))
				return
			}
			forkablePostGate = bstream.NewBlockNumGate(uint64(startBlockNum), bstream.GateInclusive, handler)
		}

		forkableHandler := forkable.New(forkablePostGate, forkable.WithExclusiveLIB(bstream.BlockRefFromID(irrID)))

		metrics.IncListeners("get_table_rows")

		irrRef := bstream.BlockRefFromID(irrID)
		source := ws.subscriptionHub.NewSourceFromBlockNumWithOpts(irrRef.Num(), forkableHandler, bstream.JoiningSourceTargetBlockID(irrRef.ID()), bstream.JoiningSourceRateLimit(300, ws.filesourceBlockRateLimit))
		source.OnTerminating(func(e error) {
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
	zlog.Debug("about to stream table deltas from block", zap.Stringer("block", block), zap.String("step", step.String()))
	var deltas []*wsmsg.TableDelta

	blk := block.ToNative().(*pbdeos.Block)
	for _, trxTrace := range blk.TransactionTraces {
		for _, dbOp := range trxTrace.DbOps {
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

type TableDeltaHandler struct {
	msg        *wsmsg.GetTableRows
	emitter    Emitter
	ctx        context.Context
	zlog       *zap.Logger
	getABIFunc func() *eos.ABI
}

func NewTableDeltaHandler(msg *wsmsg.GetTableRows, emitter Emitter, ctx context.Context, zlog *zap.Logger, getABIFunc func() *eos.ABI) *TableDeltaHandler {
	return &TableDeltaHandler{msg: msg, emitter: emitter, ctx: ctx, zlog: zlog, getABIFunc: getABIFunc}
}

func (h *TableDeltaHandler) ProcessBlock(block *bstream.Block, obj interface{}) error {
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
