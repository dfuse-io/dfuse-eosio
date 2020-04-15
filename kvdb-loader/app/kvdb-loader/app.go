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

package kvdb_loader

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	kvdbloader "github.com/dfuse-io/dfuse-eosio/kvdb-loader"
	"github.com/dfuse-io/dfuse-eosio/kvdb-loader/metrics"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	ChainId                   string // Chain ID
	ProcessingType            string // The actual processing type to perform, either `live`, `batch` or `patch`
	BlockStoreURL             string // GS path to read batch files from
	BlockStreamAddr           string // [LIVE] Address of grpc endpoint
	KvdbDsn                   string // Storage connection string
	BatchSize                 uint64 // DB batch size
	StartBlockNum             uint64 // [BATCH] Block number where we start processing
	StopBlockNum              uint64 // [BATCH] Block number where we stop processing
	NumBlocksBeforeStart      uint64 // [BATCH] Number of blocks to fetch before start block
	ParallelFileDownloadCount int    // Number of threads of parallel file download
	AllowLiveOnEmptyTable     bool   // [LIVE] force pipeline creation if live request and table is empty
	Protocol                  string // Protocol to load, EOS or ETH
	HTTPListenAddr            string //  http listen address for /healthz endpoint
}

type App struct {
	*shutter.Shutter
	Config *Config
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		Config:  config,
	}
}

func (a *App) Run() error {
	zlog.Info("launching kvdb loader", zap.Reflect("config", a.Config))

	switch a.Config.ProcessingType {
	case "live", "batch", "patch":
	default:
		return fmt.Errorf("unknown processing-type value %q", a.Config.ProcessingType)
	}

	go metrics.ServeMetrics()

	blocksStore, err := dstore.NewDBinStore(a.Config.BlockStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}
	var loader kvdbloader.Loader
	switch a.Config.Protocol {
	case "EOS":
		chainID, err := hex.DecodeString(a.Config.ChainId)
		if err != nil {
			return fmt.Errorf("decoding chain_id from command line argument: %w", err)
		}

		db, err := eosdb.New(a.Config.KvdbDsn)
		if err != nil {
			return fmt.Errorf("unable to create eosdb: %w", err)
		}
		// FIXME: make sure we call CLOSE() at the end!
		//defer db.Close()

		db.SetWriterChainID(chainID)

		l := kvdbloader.NewBigtableLoader(a.Config.BlockStreamAddr, blocksStore, a.Config.BatchSize, db, a.Config.ParallelFileDownloadCount)

		loader = l

	case "ETH":
		return fmt.Errorf("support for ETH temporarily removed")

	default:
		return fmt.Errorf("unsupported --protocol, use EOS or ETH: %q", a.Config.Protocol)
	}

	healthzHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loader.Healthy() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.Write([]byte("ready\n"))
	})

	httpSrv := &http.Server{
		Addr:         a.Config.HTTPListenAddr,
		Handler:      healthzHandler,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	}
	zlog.Info("starting webserver", zap.String("http_addr", a.Config.HTTPListenAddr))
	go httpSrv.ListenAndServe()

	switch a.Config.ProcessingType {
	case "live":
		err := loader.BuildPipelineLive(a.Config.AllowLiveOnEmptyTable)
		if err != nil {
			return err
		}
	case "batch":
		loader.StopBeforeBlock(uint64(a.Config.StopBlockNum))
		loader.BuildPipelineBatch(uint64(a.Config.StartBlockNum), uint64(a.Config.NumBlocksBeforeStart))
	case "patch":
		loader.StopBeforeBlock(uint64(a.Config.StopBlockNum))
		loader.BuildPipelinePatch(uint64(a.Config.StartBlockNum), uint64(a.Config.NumBlocksBeforeStart))
	}

	a.OnTerminating(loader.Shutdown)
	loader.OnTerminated(a.Shutdown)

	go loader.Launch()
	return nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	url := fmt.Sprintf("http://%s/healthz", a.Config.HTTPListenAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		zlog.Warn("IsReady request building error", zap.Error(err))
		return false
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		zlog.Debug("IsReady request execution error", zap.Error(err))
		return false
	}

	if res.StatusCode == 200 {
		return true
	}
	return false
}
