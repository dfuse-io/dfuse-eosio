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
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dfuse-io/dauth/authenticator"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/shutter"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

// WSConn represents a single web socket connection.
type WSConn struct {
	*shutter.Shutter

	*WebsocketHandler
	conn *websocket.Conn

	//pipelines         map[string]afterburner.Pipeline
	listenerCancelers map[string]func() error
	listenersLock     sync.Mutex
	emitLock          sync.Mutex

	creds authenticator.Credentials

	Context                  context.Context
	filesourceBlockRateLimit time.Duration
}

func NewWSConn(wshand *WebsocketHandler, conn *websocket.Conn, db DB, creds authenticator.Credentials, filesourceBlockRateLimit time.Duration, ctx context.Context) *WSConn {
	// Each WS conn will have its own SubscribablePipeline ? Hooked into the main pipeline
	// of the process, let's, for now, simply create a Joiner per socket
	ws := &WSConn{
		WebsocketHandler:         wshand,
		conn:                     conn,
		creds:                    creds,
		listenerCancelers:        make(map[string]func() error),
		Context:                  ctx,
		filesourceBlockRateLimit: filesourceBlockRateLimit,
	}

	ws.Shutter = shutter.New()
	ws.Shutter.OnTerminating(func(e error) {
		_ = ws.conn.Close()
		ws.ShutdownAllListeners()

		TrackUserEvent(ws.Context, "ws_conn_close", "error", ws.Err())
	})

	return ws
}

func (ws *WSConn) handleHeartbeats() {
	for {
		time.Sleep(10 * time.Second)

		if ws.IsTerminating() {
			return
		}

		m := wsmsg.NewPing(time.Now().UTC())
		ws.Emit(ws.Context, m)
	}
}

const maxStreamCount = 12

func (ws *WSConn) RegisterListener(ctx context.Context, reqID string, canceler func() error) error {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("registering listener", zap.String("req_id", reqID))

	ws.listenersLock.Lock()
	defer ws.listenersLock.Unlock()

	if canceler == nil {
		return fmt.Errorf("cannot register listener with null canceller")
	}

	if ws.IsTerminating() {
		return WSAlreadyClosedError(ws.Context)
	}

	streamCount := len(ws.listenerCancelers)
	if streamCount > maxStreamCount {
		return WSTooMuchStreamError(ws.Context, streamCount, maxStreamCount)
	}

	if ws.listenerCancelers[reqID] != nil {
		return WSStreamAlreadyExistError(ws.Context, reqID)
	}

	ws.listenerCancelers[reqID] = canceler

	zlogger.Debug("added listener cancelers", zap.Int("new_count", len(ws.listenerCancelers)))
	return nil
}

func (ws *WSConn) ShutdownListener(ctx context.Context, reqID string) error {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("shutting down listener", zap.String("req_id", reqID))

	ws.listenersLock.Lock()
	defer ws.listenersLock.Unlock()

	canceler, ok := ws.listenerCancelers[reqID]
	if !ok {
		return WSStreamNotFoundError(ctx, reqID)
	}

	zlogger.Debug("invoking listener canceler")
	if err := canceler(); err != nil {
		return derr.Wrap(err, "listener canceler failed")
	}

	delete(ws.listenerCancelers, reqID)

	zlogger.Debug("removed listener cancelers", zap.Int("new_count", len(ws.listenerCancelers)))
	return nil
}

func (ws *WSConn) ShutdownAllListeners() {
	ws.listenersLock.Lock()
	defer ws.listenersLock.Unlock()

	zlogger := logging.Logger(ws.Context, zlog)
	for reqID, canceler := range ws.listenerCancelers {
		if err := canceler(); err != nil {
			zlogger.Warn("canceler for listener failed canceling", zap.Error(err))
		}

		delete(ws.listenerCancelers, reqID)
	}
}

func (ws *WSConn) handleWSIncoming() {
	for {
		// msgType ignored, no distinction between binary and text messages
		msgType, rawmsg, err := ws.conn.ReadMessage()
		if err != nil {
			if !ws.IsTerminating() { // not our concern if it is already shut down...
				ws.Shutdown(err)
			}
			return
		}

		if msgType == websocket.BinaryMessage {
			ws.EmitError(ws.Context, "", WSBinaryMessageUnsupportedError(ws.Context))
			continue
		}

		ws.handleMessage(rawmsg)
	}
}

func (ws *WSConn) handleMessage(rawMsg []byte) {
	var inspect struct {
		Type   string      `json:"type"`
		ReqID  string      `json:"req_id,omitempty"`
		Listen *bool       `json:"listen,omitempty"`
		Data   interface{} `json:"data,omitempty"`
	}
	ctx := ws.Context
	err := json.Unmarshal(rawMsg, &inspect)
	if err != nil {
		ws.EmitError(ctx, "", WSInvalidJSONMessageError(ctx, err))
		return
	}

	if inspect.Type == "pong" {
		return
	}

	objType := wsmsg.IncomingMessageMap[inspect.Type]
	if objType == nil {
		ws.EmitError(ctx, inspect.ReqID, WSUnknownMessageError(ctx, inspect.Type))
		return
	}

	obj := reflect.New(objType).Interface()
	err = json.Unmarshal(rawMsg, &obj)
	if err != nil {
		ws.EmitError(ctx, inspect.ReqID, WSInvalidJSONMessageDataError(ctx, inspect.Type, err))
		return
	}

	if validatorObj, ok := obj.(wsmsg.Validator); ok {
		logging.Logger(ctx, zlog).Info("validating message")
		if err := validatorObj.Validate(ctx); err != nil {
			ws.EmitError(ctx, inspect.ReqID, WSMessageDataValidationError(ctx, err))
			return
		}
	}

	childCtx := ctx
	if inMsg, ok := obj.(wsmsg.IncomingMessage); ok {
		var span *trace.Span
		childCtx, span = dtracing.StartSpan(childCtx, fmt.Sprintf("Recv. ws/%s", inMsg.GetType()))
		span.AddAttributes(trace.StringAttribute("req_id", inMsg.GetReqID()))
		defer span.End()

		commonIn := inMsg.GetCommon()

		dataStr := gjson.GetBytes(rawMsg, "data").Raw
		if len(dataStr) > 15000 { // sometimes it is insanely big
			dataStr = "[TRUNCATED]" + dataStr[:15000]
		}
		var currentHeadNum uint64
		if headBlock := ws.subscriptionHub.HeadBlock(); headBlock != nil {
			currentHeadNum = headBlock.Num()
		}
		var relToHead uint64
		if commonIn.StartBlock <= 0 {
			relToHead = uint64(-commonIn.StartBlock)
		} else {
			relToHead = currentHeadNum - uint64(commonIn.StartBlock)
		}

		//////////////////////////////////////////////////////////////////////
		// Billable event on Websocket inbound
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithCredentials(dmetering.Event{
			Source:        "eosws",
			Kind:          "Websocket Message",
			Method:        inMsg.GetType(),
			RequestsCount: 1,
			IngressBytes:  int64(len(rawMsg)),
		}, ws.creds)
		//////////////////////////////////////////////////////////////////////

		TrackUserEvent(childCtx, "ws_conn_handle_message",
			"data", dataStr,
			"req_id", commonIn.ReqID,
			"msg_type", inMsg.GetType(),
			"start_block", commonIn.StartBlock,
			"with_progress", commonIn.WithProgress,
			"fetch", commonIn.Fetch,
			"listen", commonIn.Listen,
			"start_rel_to_head", relToHead,
		)

		childLogger := logging.Logger(childCtx, zlog).With(zap.String("req", inMsg.GetID()))
		childCtx = logging.WithLogger(childCtx, childLogger)
	}

	switch msg := obj.(type) {
	case *wsmsg.Unlisten:
		err := ws.ShutdownListener(childCtx, msg.Data.ReqID)
		if err != nil {
			ws.EmitErrorReply(childCtx, msg, derr.Wrap(err, "unable to unlisten listener"))
		} else {
			ws.EmitReply(childCtx, msg, wsmsg.NewUnlistened())
		}

	case *wsmsg.GetTransaction:
		ws.onGetTransaction(childCtx, msg)

	case *wsmsg.GetVoteTally:
		ws.onGetVoteTally(childCtx, msg)

	case *wsmsg.GetTableRows:
		ws.onGetTableRows(childCtx, msg)

	case *wsmsg.GetActionTraces:
		ws.onGetActionTraces(childCtx, msg)

	case *wsmsg.GetPrice:
		ws.onGetPrice(childCtx, msg)

	case *wsmsg.GetHeadInfo:
		ws.onGetHeadInfo(childCtx, msg)

	case *wsmsg.GetAccount:
		ws.onAccount(childCtx, msg)

	}
}

func (ws *WSConn) Emit(ctx context.Context, msg wsmsg.OutgoingMessager) {
	zlogger := logging.Logger(ctx, zlog)

	msgType, err := wsmsg.GetType(msg)
	if err != nil {
		zlogger.Error("error getting message type", zap.Error(err))
		ws.Shutdown(err)
	}

	msg.SetType(msgType)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		zlogger.Error("error marshalling message", zap.Error(err))
		ws.Shutdown(err)
	}

	//////////////////////////////////////////////////////////////////////
	// Billable event on Websocket outbound
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "eosws",
		Kind:           "Websocket Message",
		Method:         msgType,
		ResponsesCount: 1,
		EgressBytes:    int64(len(msgBytes)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	ws.emitLock.Lock()
	_ = ws.conn.SetWriteDeadline(time.Now().Add(1 * time.Minute))
	if err := ws.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		zlogger.Info("unable to write message back to client", zap.Error(err))
		ws.Shutdown(err)
	}
	ws.emitLock.Unlock()
}

func (ws *WSConn) EmitReply(ctx context.Context, originatingMsg wsmsg.IncomingMessager, msg wsmsg.OutgoingMessager) {
	msg.SetReqID(originatingMsg.GetReqID())
	ws.Emit(ctx, msg)
}

func (ws *WSConn) EmitErrorReply(ctx context.Context, msg wsmsg.IncomingMessager, err error) {
	ws.EmitError(ctx, msg.GetReqID(), err)
}

func (ws *WSConn) EmitError(ctx context.Context, reqID string, err error) {
	if ctx.Err() == context.Canceled {
		return
	}

	response := derr.ToErrorResponse(ctx, err)
	if response.Status >= 500 {
		ws.emitServerErrorResponse(ctx, reqID, response)
	} else {
		ws.emitClientErrorResponse(ctx, reqID, response)
	}
}

func (ws *WSConn) emitClientErrorResponse(ctx context.Context, reqID string, response *derr.ErrorResponse) {
	TrackUserEvent(ctx, "ws_conn_client_error", "error", response)

	logging.Logger(ctx, zlog).Info("emiting client error reply", zap.Error(response))
	ws.Emit(ctx, wsmsg.NewError(reqID, response))
}

func (ws *WSConn) emitServerErrorResponse(ctx context.Context, reqID string, response *derr.ErrorResponse) {
	TrackUserEvent(ctx, "ws_conn_server_error", "error", response)

	logging.Logger(ctx, zlog).Error("emiting server error reply", zap.Error(response))
	ws.Emit(ctx, wsmsg.NewError(reqID, response))
}
