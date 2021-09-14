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
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
)

type ProgressHandler struct {
	next         bstream.Handler
	lastProgress time.Time
	emiter       Emitter
	context      context.Context
	message      wsmsg.IncomingMessager
	stepType     forkable.StepType
}

func NewProgressHandler(next bstream.Handler, emiter Emitter, message wsmsg.IncomingMessager, context context.Context) *ProgressHandler {
	return &ProgressHandler{
		next:         next,
		emiter:       emiter,
		lastProgress: time.Now(),
		context:      context,
		message:      message,
		stepType:     forkable.StepNew,
	}
}

func (h *ProgressHandler) SetStepFilter(s forkable.StepType) {
	h.stepType = s
}

func (h *ProgressHandler) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	err := h.next.ProcessBlock(blk, obj)
	if err != nil {
		return err
	}

	if fObj, ok := obj.(*forkable.ForkableObject); ok {
		if fObj.Step != h.stepType {
			return nil
		}
	}

	// TODO: ensure the FIRST block we're processing is also sent.. right now we only send the future blocks..
	blockNum := blk.Num()

	if int64(blockNum)%h.message.GetWithProgress() == 0 && time.Since(h.lastProgress) > 250*time.Millisecond {
		p := &wsmsg.Progress{}
		p.Data.BlockNum = uint32(blockNum)
		p.Data.BlockID = blk.ID()
		h.emiter.EmitReply(h.context, h.message.GetCommon(), p)
		h.lastProgress = time.Now()
	}

	return nil

}
