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
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"go.uber.org/zap"
)

type authStartBlocker interface {
	AuthenticatedStartBlock() int64
}

type AuthorizedRequest struct {
	StartBlockID  string // has precedence over startBlockNum
	StartBlockNum uint32
	IsFutureBlock bool
}

func (ws *WSConn) AuthorizeRequest(ctx context.Context, msg wsmsg.IncomingMessager) (*AuthorizedRequest, bool) {
	headBlockID := ws.subscriptionHub.HeadBlockID()
	authReq, err := ws.authorizeRequest(msg, headBlockID)
	if err != nil {
		ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "request was not authorized"))
		return nil, false
	}

	return authReq, true
}

func (ws *WSConn) authorizeRequest(msg wsmsg.IncomingMessager, headBlock string) (*AuthorizedRequest, error) {
	zlog.Debug("authorizeRequest: creation:", zap.String("head_block", headBlock))
	common := msg.GetCommon()
	reqStartBlock := common.StartBlock

	headBlockNum := eos.BlockNum(headBlock)

	if reqStartBlock == 0 {
		return &AuthorizedRequest{
			StartBlockID:  headBlock,
			StartBlockNum: eos.BlockNum(headBlock),
		}, nil
	}

	var startBlockNum uint32
	if reqStartBlock < 0 && reqStartBlock > -4000000000 /* far in the uint32 realm, avoid overflowing `uint32` */ {
		startBlockNum = headBlockNum - uint32(-reqStartBlock)
	} else if reqStartBlock > 0 {
		startBlockNum = uint32(reqStartBlock)
	}

	var authStartBlock uint32
	if credsStart, ok := ws.creds.(authStartBlocker); ok {
		b := credsStart.AuthenticatedStartBlock()
		if b <= 0 {
			authStartBlock = headBlockNum - uint32(-b)
		} else if b > 0 {
			authStartBlock = uint32(b)
		}
	}

	if startBlockNum < authStartBlock {
		return nil, AuthInvalidStreamingStartBlockError(ws.Context, headBlockNum, startBlockNum, authStartBlock)
	}

	isFutureBlock := false
	// if startBlock is in the future, we'll let it go.. and wait here..
	if startBlockNum > headBlockNum {
		isFutureBlock = true
	}

	return &AuthorizedRequest{
		StartBlockNum: startBlockNum,
		IsFutureBlock: isFutureBlock,
	}, nil
}
