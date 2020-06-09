// Copyright 2019 dfuse Platform Inc.
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

package superviser

import (
	"context"
	"time"

	"github.com/dfuse-io/manageos/metrics"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type connectionWatchdog struct {
	superviser     *NodeosSuperviser
	reconnectDelay time.Duration
	pollDelay      time.Duration
	connections    map[string]*outboundConnectionStatus
	terminating    <-chan struct{}
}

type outboundConnectionStatus struct {
	reconnectInProgress bool
	failureCount        int
}

func (s *NodeosSuperviser) LaunchConnectionWatchdog(terminating <-chan struct{}) {
	s.Logger.Info("launching nodeos connection whatchdog")
	s.newConnectionWatchdog(terminating).Start()
}

func (s *NodeosSuperviser) newConnectionWatchdog(done <-chan struct{}) *connectionWatchdog {
	return &connectionWatchdog{
		connections:    make(map[string]*outboundConnectionStatus),
		pollDelay:      5 * time.Second,
		reconnectDelay: 30 * time.Second,
		superviser:     s,
		terminating:    done,
	}
}

func Fields() []zap.Field {
	return []zap.Field{zap.String("name", "value")}
}

func (cw *connectionWatchdog) Start() {
	logger := cw.superviser.Superviser.Logger

	for {
		select {
		case <-cw.terminating:
			return
		case <-time.After(cw.pollDelay):
		}

		if !cw.superviser.IsRunning() {
			continue
		}

		connections, err := cw.superviser.api.GetNetConnections(context.Background())
		if err != nil {
			logger.Info("couldn't get connections, will retry", zap.Error(err))
			continue
		}

		metrics.NodeosConnectedPeers.SetFloat64(float64(len(connections)))
		for _, cnx := range connections {
			cw.processConnection(cnx, logger)
		}
	}
}

func (cw *connectionWatchdog) processConnection(conn *eos.NetConnectionsResp, logger *zap.Logger) {
	if conn.Peer == "" {
		return
	}
	if _, ok := cw.connections[conn.Peer]; !ok {
		logger.Info("adding peer to watchlist", zap.String("conn_peer", conn.Peer))
		cw.connections[conn.Peer] = &outboundConnectionStatus{}
		return
	}

	if allZero(conn.LastHandshake.ChainID) {
		cs := cw.connections[conn.Peer]
		cs.failureCount++
		if cs.failureCount >= 3 {
			// We only reconnect if there is no reconnect in progress and when ready.
			// By not re-connecting when not ready, we give a better chance for the
			// process to catch-up.
			if !cs.reconnectInProgress && cw.superviser.IsRunning() {
				cs.reconnectInProgress = true
				go cw.reconnect(conn.Peer, logger)
			}
		}
	} else {
		cs := cw.connections[conn.Peer]
		cs.failureCount = 0
	}
}

func (cw *connectionWatchdog) SetDelays(pollDelay, reconnectDelay time.Duration) {
	cw.pollDelay = pollDelay
	cw.reconnectDelay = reconnectDelay
}

func (cw *connectionWatchdog) reconnect(peer string, logger *zap.Logger) {
	delay := cw.reconnectDelay
	zlogger := logger.With(zap.String("peer", peer))

	zlogger.Info("reconnection with peer", zap.Duration("delay", delay))
	disconResp, err := cw.superviser.api.NetDisconnect(context.Background(), peer)
	if err != nil {
		zlogger.Debug("got error while trying to disconnect, ignoring", zap.Any("response", disconResp), zap.Error(err))
	}

	zlogger.Debug("performing actual re-connection")
	for {
		time.Sleep(delay)
		if !cw.superviser.IsRunning() {
			continue
		}

		conResp, err := cw.superviser.api.NetConnect(context.Background(), peer)
		if err == nil {
			break
		}

		zlogger.Info("error while re-connecting peer, retrying forever", zap.Any("response", conResp), zap.Error(err))
		_, _ = cw.superviser.api.NetDisconnect(context.Background(), peer) //prevent looping on already connect peer
	}

	time.Sleep(delay)
	con, ok := cw.connections[peer]
	if !ok {
		zlogger.Info("peer still not connected after reconnect routine")
		return
	}

	zlogger.Info("peer re-connected successfully, resetting reconnect in progress state")
	con.reconnectInProgress = false
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
