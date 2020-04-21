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
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/server"
	"github.com/dfuse-io/dstore"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	StoreDSN           string // Storage connection string
	EnableLivePipeline bool   // connecs to a live source, can be turn off when doing re-processing
	BlockStreamAddr    string // gRPC endpoint to get real-time blocks
	ThreadsNum         int    // Number of threads of parallel processing
	EnableServerMode   bool   // Enables flux server mode, launch a server
	EnableInjectMode   bool   // Enables flux inject mode, writes into kvd
	HTTPListenAddr     string // Address to server FluxDB queries on
	EnableDevMode      bool   // Set to true to have a fluxdb not syncing with an actual live block source (**never** use this in prod)
	BlockStoreURL      string // dbin blocks store
}

type App struct {
	*shutter.Shutter
	config         *Config
	readinessProbe pbhealth.HealthClient
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
	}
}

func (a *App) Run() error {
	zlog.Info("running fluxdb", zap.Reflect("config", a.config))

	kvStore, err := fluxdb.NewKVStore(a.config.StoreDSN)
	if err != nil {
		return fmt.Errorf("unable to create store: %w", err)
	}

	db := fluxdb.New(kvStore)

	zlog.Info("initiating fluxdb pipeline")
	fluxDBHandler := fluxdb.NewHandler(db)

	db.SpeculativeWritesFetcher = fluxDBHandler.FetchSpeculativeWrites
	db.HeadBlock = fluxDBHandler.HeadBlock

	blocksStore, err := dstore.NewDBinStore(a.config.BlockStoreURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	db.BuildPipeline(fluxDBHandler.InitializeStartBlockID, fluxDBHandler, a.config.EnableLivePipeline, blocksStore, a.config.BlockStreamAddr, a.config.ThreadsNum)

	a.OnTerminating(func(e error) {
		db.Shutdown(nil)
	})

	db.OnTerminated(a.Shutdown)

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

	go db.Launch(a.config.EnableDevMode, a.config.HTTPListenAddr)

	return nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	url := fmt.Sprintf("http://%s/healthz", a.config.HTTPListenAddr)
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

func startHealthCheckServer(fdb *fluxdb.FluxDB, httpListenAddr string) {
	http.DefaultServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !derr.IsShuttingDown() && fdb.IsReady() {
			w.Write([]byte("r1`eady\n"))
		} else {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		}
	})

	zlog.Info("listening & serving HTTP content", zap.String("http_listen_addr", httpListenAddr))
	err := http.ListenAndServe(httpListenAddr, http.DefaultServeMux)
	zlog.Error("unable to start inject health check HTTP server", zap.Error(err))
}
