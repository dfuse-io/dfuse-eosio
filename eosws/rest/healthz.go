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

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/logging"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

var maxSecondsLatencyHubStatus = 120
var maxBlockLatencyStreams = 24
var maxBlockLatencyArchives = 500
var maxSubsystemResponseTime = 350 * time.Millisecond

type Hub struct {
	HeadBlockNum   uint32    `json:"head_block_num"`
	HeadBlockTime  time.Time `json:"head_block_time"`
	TimeLatencySec int64     `json:"time_latency_sec"`
}
type TRXDB struct {
	HeadBlock    uint32 `json:"head_block"`
	BlockLatency int    `json:"block_latency"`
}
type StateDB struct {
	HeadBlock    uint32 `json:"head_block"`
	BlockLatency int    `json:"block_latency"`
}
type Search struct {
	HeadBlock    uint32 `json:"head_block"`
	BlockLatency int    `json:"block_latency"`
}
type Merger struct {
	LastMergedBlock uint32 `json:"last_merged_block"`
	BlockLatency    int    `json:"block_latency"`
}

type Healthz struct {
	Errors  []string `json:"errors"`
	Hub     *Hub     `json:"hub,omitempty"`
	TRXDB   *TRXDB   `json:"trxdb,omitempty"`
	StateDB *StateDB `json:"statedb,omitempty"`
	Search  *Search  `json:"search,omitempty"`
	Merger  *Merger  `json:"merger,omitempty"`
	Healthy struct {
		Hub     *bool `json:"hub,omitempty"`
		Merger  *bool `json:"merger,omitempty"`
		TRXDB   *bool `json:"trxdb,omitempty"`
		SatetDB *bool `json:"statedb,omitempty"`
		Search  *bool `json:"search,omitempty"`
	} `json:"healthy"`
}

func SearchNotStuckHandler(searchEngine *eosws.SearchEngine) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchRequest := &pbsearch.RouterRequest{
			Query:               "action:onblock",
			Limit:               1,
			WithReversible:      true,
			Descending:          true,
			UseLegacyBoundaries: true,
			BlockCount:          1,
		}
		_, _, err := searchEngine.DoRequest(r.Context(), searchRequest)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("search error: %s", err.Error())))
			return
		}
		w.Write([]byte("ok"))
		return
	})
}

func HealthzHandler(hub *hub.SubscriptionHub, api *eos.API, blocksStore dstore.Store, db eosws.DB, stateClient pbstatedb.StateClient, searchEngine *eosws.SearchEngine, expectedSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlogger := logging.Logger(r.Context(), zlog)
		h := &Healthz{
			Errors: []string{},
		}

		keys := r.URL.Query()
		secret := keys.Get("secret")
		if secret != expectedSecret {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 page not found"))
			return
		}
		full, _ := strconv.ParseBool(keys.Get("full")) // full is for Checkly
		if !full && derr.IsShuttingDown() {            // k8s healthcheck needs to know we're unavailable when shutting down, Checkly does NOT
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		customMaxSecondsLatencyHubStatus, err := strconv.ParseInt(keys.Get("max_seconds_latency_hub_status"), 10, 64)
		if err == nil && customMaxSecondsLatencyHubStatus > 0 {
			maxSecondsLatencyHubStatus = int(customMaxSecondsLatencyHubStatus)
		}
		customMaxBlockLatencyStreams, err := strconv.ParseInt(keys.Get("max_block_latency_streams"), 10, 64)
		if err == nil && customMaxBlockLatencyStreams > 0 {
			maxBlockLatencyStreams = int(customMaxBlockLatencyStreams)
		}
		customMaxBlockLatencyArchives, err := strconv.ParseInt(keys.Get("max_block_latency_archives"), 10, 64)
		if err == nil && customMaxBlockLatencyArchives > 0 {
			maxBlockLatencyArchives = int(customMaxBlockLatencyArchives)
		}
		customMaxSubsystemResponseTimeMS, err := strconv.ParseInt(keys.Get("max_subsystem_response_time_ms"), 10, 64)
		if err == nil && customMaxSubsystemResponseTimeMS > 0 {
			maxSubsystemResponseTime = time.Duration(customMaxSubsystemResponseTimeMS) * time.Millisecond
		}

		var wg sync.WaitGroup

		refHeadBlockNum := healthCheckHub(h, hub, zlogger)

		if full {
			wg.Add(4)
			go healthCheckBigTable(h, &wg, db, r, refHeadBlockNum)
			go healthCheckStateDB(r.Context(), h, &wg, stateClient, refHeadBlockNum)
			go healthCheckSearch(r.Context(), h, &wg, searchEngine, refHeadBlockNum, zlogger)
			go healthCheckMerger(h, &wg, refHeadBlockNum, blocksStore, zlogger)
			if h.Hub.TimeLatencySec > int64(maxSecondsLatencyHubStatus) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		}

		wg.Wait()
		zlog.Debug("healthz wait group done")
		json.NewEncoder(w).Encode(h)

	})
}

func healthCheckSearch(ctx context.Context, h *Healthz, wg *sync.WaitGroup, searchEngine *eosws.SearchEngine, headBlockNum uint32, zlogger *zap.Logger) {
	defer wg.Done()
	done := make(chan struct{})

	go func() {
		h.Search = &Search{}
		searchRequest := &pbsearch.RouterRequest{
			Query:               "action:onblock",
			Limit:               1,
			WithReversible:      true,
			Descending:          true,
			UseLegacyBoundaries: true,
			BlockCount:          99999999999999,
		}

		traces, _, err := searchEngine.DoRequest(ctx, searchRequest)
		if err != nil {
			h.Errors = append(h.Errors, err.Error())
			close(done)
			return
		}

		if len(traces) == 0 {
			h.Errors = append(h.Errors, "no traces returned from search")
			close(done)
			return
		}

		blockNum := uint32(traces[len(traces)-1].GetBlockNum())

		h.Search.BlockLatency = int(headBlockNum - blockNum)
		if blockNum > headBlockNum {
			h.Search.BlockLatency = 0
		}
		h.Search.HeadBlock = blockNum
		isHealthy := h.Search.BlockLatency < maxBlockLatencyStreams
		h.Healthy.Search = newBool(isHealthy)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxSubsystemResponseTime):
		h.Errors = append(h.Errors, "Search health check time out")
		h.Healthy.Search = newBool(false)
	}

}

func healthCheckHub(h *Healthz, hub *hub.SubscriptionHub, zlogger *zap.Logger) (headBlockNum uint32) {
	done := make(chan struct{})

	go func() {
		hubHeadBlock := hub.HeadBlock()

		h.Hub = &Hub{}
		h.Hub.HeadBlockNum = uint32(hubHeadBlock.Num())
		h.Healthy.Hub = newBool(true)

		logOptions := []zap.Field{zap.Stringer("head_block", hubHeadBlock)}

		if block, ok := hubHeadBlock.(*bstream.PreprocessedBlock); ok {
			headBlockTime := block.Block.Time()
			h.Hub.HeadBlockTime = headBlockTime
			h.Hub.TimeLatencySec = int64(time.Now().Sub(headBlockTime) / time.Second)
			h.Healthy.Hub = newBool(h.Hub.TimeLatencySec*2 < int64(maxBlockLatencyStreams))

			logOptions = append(logOptions, zap.Time("head_block_time", headBlockTime))
		}

		zlogger.Debug("hub health check", logOptions...)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxSubsystemResponseTime):
		h.Errors = append(h.Errors, "Hub health check time out")
		h.Healthy.Hub = newBool(false)
	}

	return h.Hub.HeadBlockNum
}

func healthCheckBigTable(h *Healthz, wg *sync.WaitGroup, db eosws.DB, r *http.Request, headBlockNum uint32) {
	defer wg.Done()
	done := make(chan struct{})

	go func() {
		bigTableHeadBlocks, err := db.ListBlocks(r.Context(), math.MaxUint32, 1)
		if err != nil {
			h.Errors = append(h.Errors, err.Error())
			close(done)
			return
		}
		h.TRXDB = &TRXDB{}
		if len(bigTableHeadBlocks) == 1 {
			bgHeadBlock := bigTableHeadBlocks[0]
			blockNum := eos.BlockNum(bgHeadBlock.Id)
			h.TRXDB.HeadBlock = blockNum
			h.TRXDB.BlockLatency = int(headBlockNum - blockNum)
			if blockNum > headBlockNum {
				h.TRXDB.BlockLatency = 0
			}
		}
		isHealthy := h.TRXDB.BlockLatency < maxBlockLatencyStreams
		h.Healthy.TRXDB = newBool(isHealthy)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxSubsystemResponseTime):
		h.Errors = append(h.Errors, "Big Table health check time out")
		h.Healthy.TRXDB = newBool(false)
	}

}

func healthCheckStateDB(ctx context.Context, h *Healthz, wg *sync.WaitGroup, stateClient pbstatedb.StateClient, headBlockNum uint32) {
	defer wg.Done()
	done := make(chan struct{})

	go func() {
		h.StateDB = &StateDB{}

		res, err := stateClient.GetTableRow(ctx, &pbstatedb.GetTableRowRequest{
			Contract:   "eosio",
			Table:      "global",
			Scope:      "eosio",
			PrimaryKey: "global",
		})

		if err != nil {
			h.Errors = append(h.Errors, err.Error())
		} else {
			if res.UpToBlock != nil {
				statedbHeadBlock := uint32(res.UpToBlock.Num)

				h.StateDB.HeadBlock = statedbHeadBlock
				h.StateDB.BlockLatency = int(headBlockNum - statedbHeadBlock)

				if statedbHeadBlock > headBlockNum {
					h.StateDB.BlockLatency = 0
				}
			}
		}
		isHealthy := h.StateDB.BlockLatency < maxBlockLatencyStreams
		h.Healthy.SatetDB = newBool(isHealthy)

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxSubsystemResponseTime):
		h.Errors = append(h.Errors, "Flux health check time out")
		h.Healthy.SatetDB = newBool(false)
	}

}

func healthCheckMerger(h *Healthz, wg *sync.WaitGroup, headBlockNum uint32, blockStore dstore.Store, zlogger *zap.Logger) {
	defer wg.Done()
	done := make(chan struct{})

	go func() {
		h.Merger = &Merger{}
		baseBlockNum := headBlockNum - (headBlockNum % 100)
		baseBlockNum -= 100
		baseFilename := fmt.Sprintf("%010d", baseBlockNum)
		zlogger.Debug("lastBlockMergedFile called", zap.Uint32("head_block_num", headBlockNum), zap.Uint32("base_block_num", baseBlockNum), zap.String("base_filename", baseFilename))

		var exists bool
		var err error
		for !exists && err == nil {

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			zlogger.Debug("searching for", zap.String("base_filename", baseFilename))
			exists, err = blockStore.FileExists(ctx, baseFilename)
			baseBlockNum -= 100
			baseFilename = fmt.Sprintf("%010d", baseBlockNum)
		}

		if err != nil {
			h.Errors = append(h.Errors, err.Error())
			close(done)
			return
		}

		zlogger.Debug("found", zap.String("base_filename", baseFilename))

		lastMergedBlockNum := baseBlockNum + 99
		h.Merger.LastMergedBlock = lastMergedBlockNum
		h.Merger.BlockLatency = int(headBlockNum - lastMergedBlockNum)
		if lastMergedBlockNum > headBlockNum {
			h.Merger.BlockLatency = 0
		}
		isHealthy := h.Merger.BlockLatency < maxBlockLatencyArchives
		h.Healthy.Merger = newBool(isHealthy)

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxSubsystemResponseTime):
		h.Errors = append(h.Errors, "Merger health check time out")
		h.Healthy.Merger = newBool(false)
	}

}

func newBool(v bool) *bool {
	b := false
	if v {
		b = true
	}
	return &b
}
