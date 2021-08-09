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
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/logging"
	"github.com/gorilla/websocket"
	"github.com/streamingfast/dauth/authenticator"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
)

type WebsocketHandler struct {
	http.Handler
	abiGetter       ABIGetter
	accountGetter   AccountGetter
	db              DB
	subscriptionHub *hub.SubscriptionHub
	voteTallyHub    *VoteTallyHub
	priceHub        *PriceHub
	headInfoHub     *HeadInfoHub

	connections        int
	connectionsLock    sync.Mutex
	stateClient        pbstatedb.StateClient
	irreversibleFinder IrreversibleFinder

	maxStreamCount int
}

var hostname string
var shortIDGenerator *shortid.Shortid

func init() {
	hostname, _ = os.Hostname()
	shortIDGenerator = shortid.MustNew(1, shortid.DefaultABC, uint64(time.Now().UnixNano()))
}

func NewWebsocketHandler(
	abiGetter ABIGetter,
	accountGetter AccountGetter,
	db DB,
	subscriptionHub *hub.SubscriptionHub,
	stateClient pbstatedb.StateClient,
	voteTallyHub *VoteTallyHub,
	headInfoHub *HeadInfoHub,
	priceHub *PriceHub,
	irrFinder IrreversibleFinder,
	filesourceBlockRateLimit time.Duration,
	maxStreamCount int,
) *WebsocketHandler {
	originChecker := func(r *http.Request) bool {
		if r.Header.Get("Origin") == "" {
			// For now, we do not check the origin. This is easier for our user using Node.js
			// to avoid having to always specify an `Origin` header. Later on, we will implement
			// the origin check based on the actual token (if it's a web token) and only in those
			// cases.
			return true
		}

		return true
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		ctx := r.Context()
		derr.WriteError(ctx, w, "unable to upgrade WebSocket connection", WSUnableToUpgradeConnectionError(ctx, status, reason))
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:       originChecker,
		Error:             errorHandler,
		EnableCompression: true,
	}

	s := &WebsocketHandler{
		abiGetter:          abiGetter,
		accountGetter:      accountGetter,
		db:                 db,
		priceHub:           priceHub,
		subscriptionHub:    subscriptionHub,
		stateClient:        stateClient,
		voteTallyHub:       voteTallyHub,
		headInfoHub:        headInfoHub,
		irreversibleFinder: irrFinder,
		maxStreamCount:     maxStreamCount,
	}

	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		credentials := authenticator.GetCredentials(ctx)
		// r.Trailer = http.Header{
		// 	"X-Client-ID": []string{credentials.Id},
		// }

		// DO NOT remove nor renamed the `ws_conn_id` parameter - mission critical for BigQuery analytics
		zlogger := logging.Logger(ctx, zlog).With(zap.String("ws_conn_id", shortIDGenerator.MustGenerate()))
		zlogger.Debug("handling initial WebSocket connection request")

		wrappedWriter := TurnIntoStatusAwareResponseWriter(w)
		c, err := upgrader.Upgrade(wrappedWriter, r, nil)
		if err != nil {
			// The custom errorHandler configured on the upgrader takes care of logging the error, so no need to do nothing
			return
		}

		defer c.Close()

		s.incConnectionsCounter()
		defer s.decConnectionsCounter()

		childCtx := logging.WithLogger(ctx, zlogger)

		TrackUserEvent(childCtx, "ws_conn_start", "connection_count", s.connections)

		conn := NewWSConn(s, c, credentials, filesourceBlockRateLimit, childCtx)
		go conn.handleWSIncoming()

		go conn.handleHeartbeats()
		select {
		case <-ctx.Done():
			zlog.Info("context done", zap.Error(ctx.Err()))
		case <-conn.Terminating():
			zlog.Info("connection done", zap.Error(ctx.Err()))
		}
		_ = conn.conn.Close()
	})

	return s
}

func (s *WebsocketHandler) incConnectionsCounter() {
	s.connectionsLock.Lock()
	s.connections++
	s.connectionsLock.Unlock()

	metrics.IncCurrentConnections()
}

func (s *WebsocketHandler) decConnectionsCounter() {
	s.connectionsLock.Lock()
	s.connections--
	s.connectionsLock.Unlock()

	metrics.CurrentConnections.Dec()
}
