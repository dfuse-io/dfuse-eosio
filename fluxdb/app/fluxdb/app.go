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

package fluxdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/metrics"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/server"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	StoreDSN                 string // Storage connection string
	BlockStreamAddr          string // gRPC endpoint to get real-time blocks
	ThreadsNum               int    // Number of threads of parallel processing
	EnableServerMode         bool   // Enables flux server mode, launch a server
	EnableInjectMode         bool   // Enables flux inject mode, writes into kvd
	EnablePipeline           bool   // Connects to blocks pipeline, can be used to have a development server only fluxdb
	EnableReprocSharderMode  bool   // Enables flux reproc shard mode, exclusive option, cannot be set if either server, injector or reproc-injector mode is set
	EnableReprocInjectorMode bool   // Enables flux reproc injector mode, exclusive option, cannot be set if either server, injector or reproc-shard mode is set
	HTTPListenAddr           string // Address to server FluxDB queries on
	BlockStoreURL            string // dbin blocks store

	// Available for reproc mode only (either reproc shard or reproc injector)
	ReprocShardStoreURL string
	ReprocShardCount    uint64

	// Available for reproc-shard only
	ReprocSharderStartBlockNum uint64
	ReprocSharderStopBlockNum  uint64

	// Available for reproc-injector only
	ReprocInjectorShardIndex uint64
}

type App struct {
	*shutter.Shutter
	config *Config
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
	}
}

func (a *App) Run() error {
	zlog.Info("running fluxdb", zap.Reflect("config", a.config))
	if err := a.config.validate(); err != nil {
		return fmt.Errorf("invalid app config: %w", err)
	}

	dmetrics.Register(metrics.MetricSet)

	kvStore, err := fluxdb.NewKVStore(a.config.StoreDSN)
	if err != nil {
		return fmt.Errorf("unable to create store: %w", err)
	}

	blocksStore, err := dstore.NewDBinStore(a.config.BlockStoreURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	if a.config.EnableInjectMode || a.config.EnableServerMode {
		return a.startStandard(blocksStore, kvStore)
	}

	if a.config.EnableReprocSharderMode {
		return a.startReprocSharder(blocksStore)
	}

	if a.config.EnableReprocInjectorMode {
		return a.startReprocInjector(kvStore)
	}

	return errors.New("invalid configuration, don't know what to start for fluxdb")
}

func (a *App) startStandard(blocksStore dstore.Store, kvStore store.KVStore) error {
	db := fluxdb.New(kvStore)

	zlog.Info("initiating fluxdb handler")
	fluxDBHandler := fluxdb.NewHandler(db)

	db.SpeculativeWritesFetcher = fluxDBHandler.FetchSpeculativeWrites
	db.HeadBlock = fluxDBHandler.HeadBlock

	a.OnTerminating(func(e error) {
		db.Shutdown(nil)
	})

	db.OnTerminated(a.Shutdown)

	if a.config.EnableInjectMode || a.config.EnablePipeline {
		db.BuildPipeline(fluxDBHandler.InitializeStartBlockID, fluxDBHandler, blocksStore, a.config.BlockStreamAddr, a.config.ThreadsNum)
	}

	if a.config.EnableInjectMode {
		zlog.Info("setting up injector mode write")
		fluxDBHandler.EnableWrites()
	}

	if a.config.EnableServerMode {
		zlog.Info("setting up server")
		srv := server.New(a.config.HTTPListenAddr, db)
		go srv.Serve()
	} else {
		zlog.Info("setting injecter mode health check")
		go startHealthCheckServer(db, a.config.HTTPListenAddr)
	}

	go db.Launch(a.config.EnablePipeline, a.config.HTTPListenAddr)

	return nil
}

func (a *App) startReprocSharder(blocksStore dstore.Store) error {
	shardsStore, err := dstore.NewStore(a.config.ReprocShardStoreURL, "shard.zst", "zstd", true)
	if err != nil {
		return fmt.Errorf("unable to create shards store at %s: %w", a.config.ReprocShardStoreURL, err)
	}

	shardingPipe := fluxdb.NewSharder(shardsStore, int(a.config.ReprocShardCount), uint32(a.config.ReprocSharderStartBlockNum), uint32(a.config.ReprocSharderStopBlockNum))

	// FIXME: We should use the new `DPoSLIBNumAtBlockHeightFromBlockStore` to go back as far as neede!
	source := fluxdb.BuildReprocessingPipeline(shardingPipe, blocksStore, a.config.ReprocSharderStartBlockNum, 400, 2)

	a.OnTerminating(func(e error) {
		source.Shutdown(nil)
	})

	source.OnTerminated(func(err error) {
		// FIXME: This `HasSuffix` is sh**ty, need to replace with a better pattern, `source.Shutdown(nil)` is one of them
		if err != nil && strings.HasSuffix(err.Error(), fluxdb.ErrCleanSourceStop.Error()) {
			err = nil
		}

		a.Shutdown(err)
	})

	source.Run()

	// Wait for either source to complete or the app being killed
	select {
	case <-a.Terminating():
	case <-source.Terminated():
	}

	return nil
}

func (a *App) startReprocInjector(kvStore store.KVStore) error {
	db := fluxdb.New(kvStore)

	db.SetSharding(int(a.config.ReprocInjectorShardIndex), int(a.config.ReprocShardCount))
	if err := db.CheckCleanDBForSharding(); err != nil {
		return fmt.Errorf("db is not clean before injecting shards: %w", err)
	}

	shardStoreFullURL := a.config.ReprocShardStoreURL + "/" + fmt.Sprintf("%03d", a.config.ReprocInjectorShardIndex)
	zlog.Info("using shards url", zap.String("store_url", shardStoreFullURL))

	shardStore, err := dstore.NewStore(shardStoreFullURL, "shard.zst", "zstd", true)
	if err != nil {
		return fmt.Errorf("unable to create shards store at %s: %w", shardStoreFullURL, err)
	}

	shardInjector := fluxdb.NewShardInjector(shardStore, db)

	a.OnTerminating(func(e error) {
		shardInjector.Shutdown(nil)
	})

	shardInjector.OnTerminated(a.Shutdown)

	if err := shardInjector.Run(); err != nil {
		return fmt.Errorf("injector failed: %w", err)
	}

	ctx := context.Background()
	lastBlock, err := db.VerifyAllShardsWritten(ctx)
	if err != nil {
		zlog.Info("all shards are not done yet, not updating lastBlockID", zap.Error(err))
		a.Shutdown(nil)
		return nil
	}

	err = db.UpdateGlobalLastBlockID(ctx, lastBlock)
	if err != nil {
		zlog.Error("cannot update lastBlockID", zap.Error(err))
		return fmt.Errorf("cannot update lastBlockID: %w", err)
	}

	a.Shutdown(nil)
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

func startHealthCheckServer(fdb *fluxdb.FluxDB, httpListenAddr string) {
	http.DefaultServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !derr.IsShuttingDown() && fdb.IsReady() {
			w.Write([]byte("ready\n"))
		} else {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		}
	})

	zlog.Info("listening & serving HTTP content", zap.String("http_listen_addr", httpListenAddr))
	err := http.ListenAndServe(httpListenAddr, http.DefaultServeMux)
	zlog.Error("unable to start inject health check HTTP server", zap.Error(err))
}

func (config *Config) validate() error {
	server := config.EnableServerMode
	injector := config.EnableInjectMode
	reprocSharder := config.EnableReprocSharderMode
	reprocInjector := config.EnableReprocInjectorMode

	if !server && !injector && !reprocSharder && !reprocInjector {
		return errors.New("no mode selected, one of enable server, enable injector, enable reproc sharder or enable reproc injector must be set")
	}

	if reprocSharder && (server || injector || reprocInjector) {
		return errors.New("reproc sharder mode is an exclusive option, cannot be set while any of enable server, enable injector or enable reproc injector is set")
	}

	if reprocInjector && (server || injector || reprocSharder) {
		return errors.New("reproc injector mode is an exclusive option, cannot be set while any of enable server, enable injector or enable reproc injector is set")
	}

	if (reprocSharder || reprocInjector) && config.ReprocShardCount <= 0 {
		return errors.New("reproc mode requires you to set a shard count value higher than 0")
	}

	if reprocInjector && config.ReprocInjectorShardIndex >= config.ReprocShardCount {
		return fmt.Errorf("reproc injector mode shard index invalid, got index %d but it's outside possible value for a shard count of %d", config.ReprocInjectorShardIndex, config.ReprocShardCount)
	}

	return nil
}
