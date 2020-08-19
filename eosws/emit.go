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

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
)

type Emitter interface {
	Emit(ctx context.Context, msg wsmsg.OutgoingMessager)
	EmitReply(ctx context.Context, originatingMsg wsmsg.IncomingMessager, msg wsmsg.OutgoingMessager)
	EmitErrorReply(ctx context.Context, msg wsmsg.IncomingMessager, err error)
	EmitError(ctx context.Context, reqID string, err error)
}

type TestEmitter struct {
	ctx      context.Context
	messages []wsmsg.OutgoingMessager
	callBack func(wsmsg.OutgoingMessager)
}

func NewTestEmitter(ctx context.Context, callBack func(wsmsg.OutgoingMessager)) *TestEmitter {
	return &TestEmitter{
		ctx:      ctx,
		messages: []wsmsg.OutgoingMessager{},
	}
}

func (e *TestEmitter) Emit(ctx context.Context, msg wsmsg.OutgoingMessager) {

	if e.callBack != nil {
		e.callBack(msg)
	}

	e.messages = append(e.messages, msg)
}

func (e *TestEmitter) EmitReply(ctx context.Context, originatingMsg wsmsg.IncomingMessager, msg wsmsg.OutgoingMessager) {
	msg.SetReqID(originatingMsg.GetReqID())
	e.Emit(ctx, msg)
}

func (e *TestEmitter) EmitErrorReply(ctx context.Context, msg wsmsg.IncomingMessager, err error) {
	e.EmitError(ctx, msg.GetReqID(), err)
}

func (e *TestEmitter) EmitError(ctx context.Context, reqID string, err error) {
	e.Emit(ctx, wsmsg.NewError(reqID, derr.ToErrorResponse(e.ctx, err)))
}
