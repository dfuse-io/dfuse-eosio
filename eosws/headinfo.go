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

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
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
	initialLIB        string
	subscriptionHub   *hub.SubscriptionHub
}

func NewHeadInfoHub(initialStartBlock string, initialLIB string, subscriptionHub *hub.SubscriptionHub) *HeadInfoHub {
	return &HeadInfoHub{
		CommonHub:         CommonHub{name: "head_info"},
		initialStartBlock: initialStartBlock,
		initialLIB:        initialLIB,
		subscriptionHub:   subscriptionHub,
	}
}

func (h *HeadInfoHub) Launch(ctx context.Context) {
	libRef := bstream.BlockRefFromID(h.initialLIB)
	startBlock := eos.BlockNum(h.initialStartBlock)

	handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		fObj := obj.(*forkable.ForkableObject)
		if fObj.Step == forkable.StepNew {
			blk := block.ToNative().(*pbdeos.Block)

			headInfo := &wsmsg.HeadInfo{}
			headInfo.Data = &wsmsg.HeadInfoData{
				LastIrreversibleBlockNum: uint32(fObj.ForkDB.LIBNum()),
				LastIrreversibleBlockId:  fObj.ForkDB.LIBID(),
				HeadBlockNum:             uint32(block.Num()),
				HeadBlockId:              block.ID(),
				HeadBlockTime:            block.Time(),
				HeadBlockProducer:        blk.Header.Producer,
			}

			metrics.HeadTimeDrift.SetBlockTime(block.Time())
			metrics.HeadBlockNum.SetUint64(block.Num())

			h.SetLast(headInfo)
			h.EmitAll(ctx, headInfo)

			startBlock = headInfo.Data.HeadBlockNum
			libRef = bstream.BlockRefFromID(headInfo.Data.LastIrreversibleBlockId)
		}

		return nil

	})

	gateHandler := bstream.NewBlockNumGate(uint64(startBlock), bstream.GateExclusive, handler)
	forkableHandler := forkable.New(gateHandler, forkable.WithExclusiveLIB(libRef))

	joiningSourceFactory := bstream.SourceFromRefFactory(func(blockRef bstream.BlockRef, handler bstream.Handler) bstream.Source {
		if blockRef.ID() == "" {
			blockRef = libRef
		}

		gate := bstream.NewBlockIDGate(blockRef.ID(), bstream.GateInclusive, forkableHandler)
		return h.subscriptionHub.NewSourceFromBlockRef(blockRef, gate)
	})

	eternalSource := bstream.NewEternalSource(joiningSourceFactory, forkableHandler)

	eternalSource.Run()
	eternalSource.OnTerminating(func(e error) {
		zlog.Error("Head info failed and quit", zap.Error(e))
	})
}
