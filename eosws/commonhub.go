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
	"fmt"
	"sync"

	"github.com/dfuse-io/logging"

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"github.com/streamingfast/shutter"
)

type CommonHub struct {
	subscribersLock     sync.Mutex
	subscribers         []*bufferedEmitter
	lastOutgoingMsgLock sync.Mutex
	lastOutgoingMsg     wsmsg.OutgoingMessager
	name                string
}

func (c *CommonHub) SetLast(msg wsmsg.OutgoingMessager) {
	c.lastOutgoingMsgLock.Lock()
	c.lastOutgoingMsg = msg
	c.lastOutgoingMsgLock.Unlock()
}

func (c *CommonHub) Last() wsmsg.OutgoingMessager {
	c.lastOutgoingMsgLock.Lock()
	defer c.lastOutgoingMsgLock.Unlock()
	return c.lastOutgoingMsg
}

func (c *CommonHub) Subscribe(ctx context.Context, msg wsmsg.IncomingMessager, ws *WSConn) {
	emitter := newBufferedEmitter(msg.GetReqID(), ws)
	err := ws.RegisterListener(ctx, msg.GetReqID(), func() error {
		c.Unsubscribe(ctx, emitter)
		metrics.CurrentListeners.Dec(c.name)
		emitter.Shutdown(nil)
		return nil
	})

	if err != nil {
		ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "registration of listener failed"))
		return
	}

	c.subscribersLock.Lock()
	c.subscribers = append(c.subscribers, emitter)
	c.subscribersLock.Unlock()

	logging.Logger(ctx, zlog).Debug("subscribing emitter to hub")
	out := c.Last()
	if out != nil {
		emitter.Emit(out)
	}
	metrics.IncListeners(c.name)
	go emitter.Launch(ctx)
}

func (c *CommonHub) EmitAll(ctx context.Context, msg wsmsg.OutgoingMessager) {
	c.subscribersLock.Lock()
	subscribers := c.subscribers
	c.subscribersLock.Unlock()

	for _, emitter := range subscribers {
		emitter.Emit(msg)
	}
}

func (c *CommonHub) Unsubscribe(ctx context.Context, removeEmitter *bufferedEmitter) {
	c.subscribersLock.Lock()
	defer c.subscribersLock.Unlock()

	var newSubscribers []*bufferedEmitter
	for _, emitter := range c.subscribers {
		if emitter != removeEmitter {
			newSubscribers = append(newSubscribers, emitter)
		} else {
			logging.Logger(ctx, zlog).Debug("removing emitter from hub")
		}
	}

	c.subscribers = newSubscribers
}

type bufferedEmitter struct {
	*shutter.Shutter
	reqID string
	ws    *WSConn
	c     chan wsmsg.OutgoingMessager
}

func (e *bufferedEmitter) Launch(ctx context.Context) {
	logging.Logger(ctx, zlog).Debug("launching buffered emitter")

	for {
		select {
		case <-e.Terminating():
			return
		case msg := <-e.c:
			if e.IsTerminating() {
				return
			}
			msg.SetReqID(e.reqID)
			e.ws.Emit(ctx, msg)
		}
	}
}

func newBufferedEmitter(reqID string, ws *WSConn) *bufferedEmitter {
	e := &bufferedEmitter{
		Shutter: shutter.New(),
		reqID:   reqID,
		ws:      ws,
		c:       make(chan wsmsg.OutgoingMessager, 200),
	}
	return e
}

func (e *bufferedEmitter) Emit(msg wsmsg.OutgoingMessager) {
	if e.IsTerminating() {
		return
	}
	if len(e.c) > 199 {
		e.ws.Shutdown(fmt.Errorf("buffer is too full in buffered emitter"))
		return
	}
	e.c <- msg
}
