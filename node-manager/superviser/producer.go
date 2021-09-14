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
	"fmt"
	"regexp"
	"strconv"
	"time"

	nodeManager "github.com/streamingfast/node-manager"
	"go.uber.org/zap"
)

func (s *NodeosSuperviser) ResumeProduction() error {
	s.Logger.Info("sending API call to resume nodeos producer")
	err := s.api.ProducerResume(context.Background())
	if err != nil {
		return err
	}

	s.producerHostname = s.options.Hostname

	return nil
}

func (s *NodeosSuperviser) PauseProduction() error {
	s.Logger.Info("sending API call to pause nodeos producer")
	err := s.api.ProducerPause(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (s *NodeosSuperviser) IsProducing() (bool, error) {
	if !s.IsActiveProducer() {
		return false, nil
	}

	isPaused, err := s.api.IsProducerPaused(context.Background())
	if err != nil {
		return false, err
	}

	return !isPaused, nil
}

func (s *NodeosSuperviser) IsActiveProducer() bool {
	return s.forceProduction || (s.options.Hostname != "" && s.producerHostname == s.options.Hostname)
}

func (s *NodeosSuperviser) WaitUntilEndOfNextProductionRound(timeout time.Duration) error {
	delay := 0 * time.Second
	for {
		time.Sleep(delay)
		delay = 1 * time.Second

		select {
		case <-time.After(timeout):
			return fmt.Errorf("timed out waiting for AfterProduction state")
		default:
		}

		if s.isPostProdOrStaleState() {
			return nil
		}
	}
}

var reReceivedBlock = regexp.MustCompile(`info\s+\S+\s+thread-\d+\s+producer_plugin.cpp:\d+\s+on_incoming_block\s+\] Received block ([0-9a-z]+)... #(\d+) @ .* signed by ([a-z1-5.]+) \[trxs: (\d+), lib: (\d+), conf: (\d+), latency: (-?\d+) ms\]`)
var reProducedBlock = regexp.MustCompile(`info\s+\S+\s+thread-\d+\s+producer_plugin.cpp:\d+\s+produce_block\s+\] Produced block ([0-9a-z]+)... #(\d+) @ .* signed by ([a-z1-5.]+) \[trxs: (\d+), lib: (\d+), confirmed: (\d+)\]`)

func (s *NodeosSuperviser) isPostProdOrStaleState() bool {
	s.productionStateLock.Lock()
	defer s.productionStateLock.Unlock()

	switch s.productionState {
	case nodeManager.StatePost, nodeManager.StateStale:
		return true
	case nodeManager.StateProducing, nodeManager.StatePre:
		return false
	default:
		s.Logger.Info("invalid production state", zap.Any("production_state", s.productionState))
		return false
	}

}

func (s *NodeosSuperviser) analyzeLogLineForStateChange(in string) {
	if len(in) < 5 || in[0:4] != "info" {
		return
	}

	if match := reReceivedBlock.FindStringSubmatch(in); match != nil {
		blockNumber, _ := strconv.ParseInt(match[2], 10, 64)
		s.updateProductionState(blockNumber, nodeManager.EventReceived)
	} else if match := reProducedBlock.FindStringSubmatch(in); match != nil {
		blockNumber, _ := strconv.ParseInt(match[2], 10, 64)
		s.updateProductionState(blockNumber, nodeManager.EventProduced)
	}
}

func (s *NodeosSuperviser) updateProductionState(blockNum int64, event nodeManager.ProductionEvent) {
	s.productionStateLock.Lock()
	defer s.productionStateLock.Unlock()

	switch event {
	case nodeManager.EventProduced:
		s.changeProductionState(nodeManager.StateProducing)
		s.productionStateLastProduced = time.Now()

	case nodeManager.EventReceived:
		lastProd := s.productionStateLastProduced
		if lastProd.After(time.Now().Add(-2 * time.Second)) {
			s.changeProductionState(nodeManager.StateProducing) // still mark as producing...
		} else if lastProd.After(time.Now().Add(-60 * time.Second)) {
			s.changeProductionState(nodeManager.StatePost)
		} else if lastProd.After(time.Now().Add(-12 * time.Minute)) {
			s.changeProductionState(nodeManager.StatePre)
		} else {
			s.changeProductionState(nodeManager.StateStale)
		}

	default:
		panic("invalid message")
	}
}

func (s *NodeosSuperviser) changeProductionState(newState nodeManager.ProductionState) {
	// Call with the `productionStateLock` already acquired.
	if s.productionState != newState {
		s.Logger.Info("changing production state", zap.Any("new_state", newState))
	}

	s.productionState = newState
}
