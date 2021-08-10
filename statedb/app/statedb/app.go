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

package statedb

import (
	"fmt"
	"net/http"

	"github.com/dfuse-io/bstream"
	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dfuse-eosio/statedb/grpc"
	"github.com/dfuse-io/dfuse-eosio/statedb/metrics"
	"github.com/dfuse-io/dfuse-eosio/statedb/server"
	"github.com/dfuse-io/dmetrics"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/streamingfast/fluxdb"
	appFluxdb "github.com/streamingfast/fluxdb/app/fluxdb"
	"go.uber.org/zap"
)

type Config struct {
	*appFluxdb.Config

	HTTPListenAddr string
	GRPCListenAddr string
}

type Modules struct {
	BlockFilter        func(blk *bstream.Block) error
	BlockMeta          pbblockmeta.BlockIDClient
	StartBlockResolver bstream.StartBlockResolver
}

type App struct {
	*appFluxdb.App

	config *Config
}

func New(config *Config, modules *Modules) *App {
	app := &App{
		config: config,
	}

	app.App = appFluxdb.New(
		config.Config,
		&appFluxdb.Modules{
			// Required dependencies
			OnInjectMode:       app.startForInjectMode,
			OnServerMode:       app.startForServeMode,
			BlockMapper:        &statedb.BlockMapper{},
			StartBlockResolver: modules.StartBlockResolver,

			// Optional dependencies
			BlockFilter: modules.BlockFilter,
			BlockMeta:   modules.BlockMeta,
		},
	)

	return app
}

func (a *App) Run() error {
	zlog.Info("running statedb", zap.Reflect("config", a.config))
	if err := a.config.Validate(); err != nil {
		return fmt.Errorf("invalid app config: %w", err)
	}

	dmetrics.Register(metrics.MetricSet)

	return a.App.Run()
}

func (a *App) startForServeMode(db *fluxdb.FluxDB) {
	zlog.Info("setting up server")
	httpServer := server.New(a.config.HTTPListenAddr, db)
	go httpServer.Serve()

	grpcServer := grpc.New(a.config.GRPCListenAddr, db)
	go grpcServer.Serve()
}

func (a *App) startForInjectMode(db *fluxdb.FluxDB) {
	http.DefaultServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !derr.IsShuttingDown() && db.IsReady() {
			w.Write([]byte("ready\n"))
		} else {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		}
	})

	zlog.Info("listening & serving HTTP health endpoint", zap.String("http_listen_addr", a.config.HTTPListenAddr))
	go func() {
		err := http.ListenAndServe(a.config.HTTPListenAddr, http.DefaultServeMux)
		a.Shutdown(fmt.Errorf("unable to start inject health check HTTP server: %w", err))
	}()
}
