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

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/streamingfast/bstream/hub"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

func (ws *WSConn) onGetHeadInfo(ctx context.Context, msg *wsmsg.GetHeadInfo) {
	if msg.Listen {
		ws.headInfoHub.Subscribe(ctx, msg, ws)
		return
	}

	if msg.Fetch {
		out := ws.headInfoHub.Last()
		if out == nil {
			ws.EmitErrorReply(ctx, msg, AppHeadInfoNotReadyError(ctx))
			return
		}
		metrics.DocumentResponseCounter.Inc()
		ws.EmitReply(ctx, msg, out)
	}
}

type HeadInfoHub struct {
	CommonHub
	initialStartBlock string
	currentLIB        *atomic.String
	subscriptionHub   *hub.SubscriptionHub
}

func NewHeadInfoHub(initialStartBlock string, initialLIB string, subscriptionHub *hub.SubscriptionHub) *HeadInfoHub {
	return &HeadInfoHub{
		CommonHub:         CommonHub{name: "head_info"},
		initialStartBlock: initialStartBlock,
		currentLIB:        atomic.NewString(initialLIB),
		subscriptionHub:   subscriptionHub,
	}
}

func (h *HeadInfoHub) Launch(ctx context.Context) {
	libRef := bstream.NewBlockRefFromID(h.currentLIB.Load())
	startBlock := eos.BlockNum(h.initialStartBlock)

	handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		fObj := obj.(*forkable.ForkableObject)
		if fObj.Step == forkable.StepNew {
			blk := block.ToNative().(*pbcodec.Block)

			headInfo := &wsmsg.HeadInfo{}
			headInfo.Data = &wsmsg.HeadInfoData{
				LastIrreversibleBlockNum: uint32(fObj.ForkDB.LIBNum()),
				LastIrreversibleBlockId:  fObj.ForkDB.LIBID(),
				HeadBlockNum:             uint32(block.Num()),
				HeadBlockId:              block.ID(),
				HeadBlockTime:            block.Time(),
				HeadBlockProducer:        blk.Header.Producer,
			}
			h.currentLIB.Store(fObj.ForkDB.LIBID())

			metrics.HeadTimeDrift.SetBlockTime(block.Time())
			metrics.HeadBlockNum.SetUint64(block.Num())

			h.SetLast(headInfo)
			h.EmitAll(ctx, headInfo)

			startBlock = headInfo.Data.HeadBlockNum
			libRef = bstream.NewBlockRef(headInfo.Data.LastIrreversibleBlockId, uint64(headInfo.Data.LastIrreversibleBlockNum))
		}

		return nil

	})

	gateHandler := bstream.NewBlockNumGate(uint64(startBlock), bstream.GateExclusive, handler, bstream.GateOptionWithLogger(zlog))
	forkableHandler := forkable.New(gateHandler, forkable.WithLogger(zlog), forkable.WithExclusiveLIB(libRef))

	joiningSourceFactory := bstream.SourceFromRefFactory(func(blockRef bstream.BlockRef, handler bstream.Handler) bstream.Source {
		if blockRef.ID() == "" {
			blockRef = libRef
		}

		gate := bstream.NewBlockIDGate(blockRef.ID(), bstream.GateInclusive, forkableHandler)
		return h.subscriptionHub.NewSourceFromBlockRef(blockRef, gate)
	})

	eternalSource := bstream.NewEternalSource(joiningSourceFactory, forkableHandler, bstream.EternalSourceWithLogger(zlog))

	eternalSource.Run()
	eternalSource.OnTerminating(func(e error) {
		zlog.Error("head info failed and quit", zap.Error(e))
	})
}

func (h *HeadInfoHub) LibID() string {
	return h.currentLIB.Load()
}
