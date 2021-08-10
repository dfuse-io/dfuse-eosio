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

package trxdb_loader

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	trxdbloader "github.com/dfuse-io/dfuse-eosio/trxdb-loader"
	"github.com/dfuse-io/dfuse-eosio/trxdb-loader/metrics"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/dstore"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type Config struct {
	ChainID                   string // Chain ID
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
	HTTPListenAddr            string //  http listen address for /healthz endpoint
	EnableTruncationMarker    bool   // Enables the storage of truncation markers
	TruncationWindow          uint64 // Truncate date within this duration
	PurgerInterval            uint64 // Purger at every X block
}

type App struct {
	*shutter.Shutter
	config  *Config
	modules *Modules
}

type Modules struct {
	BlockFilter func(blk *bstream.Block) error
	BlockMeta   pbblockmeta.BlockIDClient
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
		modules: modules,
	}
}

func (a *App) Run() error {
	zlog.Info("launching trxdb loader", zap.Reflect("config", a.config))

	dmetrics.Register(metrics.Metricset)

	switch a.config.ProcessingType {
	case "live", "batch", "patch":
	default:
		return fmt.Errorf("unknown processing-type value %q", a.config.ProcessingType)
	}

	blocksStore, err := dstore.NewDBinStore(a.config.BlockStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}

	chainID, err := hex.DecodeString(a.config.ChainID)
	if err != nil {
		return fmt.Errorf("decoding chain_id from command line argument: %w", err)
	}

	trxdbOption := []trxdb.Option{trxdb.WithLogger(zlog)}
	if a.config.EnableTruncationMarker {
		trxdbOption = append(trxdbOption, trxdb.WithPurgeableStoreOption(a.config.TruncationWindow, a.config.PurgerInterval))
	}

	db, err := trxdb.New(a.config.KvdbDsn, trxdbOption...)
	if err != nil {
		return fmt.Errorf("unable to create trxdb: %w", err)
	}

	db.SetWriterChainID(chainID)

	loader := trxdbloader.NewTrxDBLoader(
		a.config.BlockStreamAddr,
		blocksStore,
		a.config.BatchSize,
		db,
		a.config.ParallelFileDownloadCount,
		a.modules.BlockFilter,
		a.config.TruncationWindow,
		a.modules.BlockMeta,
	)

	healthzHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loader.Healthy() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.Write([]byte("ready\n"))
	})

	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		return fmt.Errorf("unable to create error logger: %w", err)
	}

	httpSrv := &http.Server{
		Addr:     a.config.HTTPListenAddr,
		Handler:  healthzHandler,
		ErrorLog: errorLogger,
	}
	zlog.Info("starting webserver", zap.String("http_addr", a.config.HTTPListenAddr))
	go httpSrv.ListenAndServe()

	switch a.config.ProcessingType {
	case "live":
		err := loader.BuildPipelineLive(a.config.AllowLiveOnEmptyTable)
		if err != nil {
			return err
		}
	case "batch":
		loader.StopBeforeBlock(uint64(a.config.StopBlockNum))
		loader.BuildPipelineBatch(uint64(a.config.StartBlockNum), uint64(a.config.NumBlocksBeforeStart))
	case "patch":
		loader.StopBeforeBlock(uint64(a.config.StopBlockNum))
		loader.BuildPipelinePatch(uint64(a.config.StartBlockNum), uint64(a.config.NumBlocksBeforeStart))
	}

	a.OnTerminating(func(err error) {
		loader.Shutdown(err)
		db.Close()
	})

	loader.OnTerminated(func(err error) {
		db.Close()
		a.Shutdown(err)
	})

	go loader.Launch()
	return nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	url := fmt.Sprintf("http://%s/healthz", a.config.HTTPListenAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		zlog.Warn("is ready request building error", zap.Error(err))
		return false
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		zlog.Debug("is ready request execution error", zap.Error(err))
		return false
	}

	if res.StatusCode == 200 {
		return true
	}
	return false
}
