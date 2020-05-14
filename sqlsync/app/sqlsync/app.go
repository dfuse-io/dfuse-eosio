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

package sqlsync

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	_ "github.com/dfuse-io/dauth/null" // auth plugin
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/dfuse-eosio/sqlsync"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	SQLDSN          string
	BlockStreamAddr string
	SourceStoreURL  string
	FluxHTTPAddr    string
	HTTPListenAddr  string // for healthz only
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
	zlog.Info("running sqlsync app", zap.Reflect("config", a.Config))

	ctx, cancel := context.WithCancel(context.Background())
	a.OnTerminating(func(_ error) { cancel() })

	blocksStore, err := dstore.NewDBinStore(a.Config.SourceStoreURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	liveSourceFactory := bstream.SourceFactory(func(h bstream.Handler) bstream.Source {
		return blockstream.NewSource(ctx, a.Config.BlockStreamAddr, 300, h)
	})

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		src := bstream.NewFileSource(blocksStore, startBlockNum, 1, nil, h)
		return src
	})

	fluxClient := fluxdb.NewClient(a.Config.FluxHTTPAddr, http.DefaultTransport)

	db, err := sqlsync.NewDB(a.Config.SQLDSN)
	if err != nil {
		return fmt.Errorf("sql db setup: %w", err)
	}

	sqlSyncer := sqlsync.NewSQLSync(db, fluxClient, liveSourceFactory, fileSourceFactory)

	go func() {
		zlog.Info("starting sql syncer pipeline")
		go a.Shutdown(sqlSyncer.Launch())
	}()

	httpServer := &http.Server{Addr: a.Config.HTTPListenAddr, Handler: sqlSyncer.HealthzHandler()}
	go func() {
		zlog.Info("serving HTTP", zap.String("listen_addr", a.Config.HTTPListenAddr))
		go a.Shutdown(httpServer.ListenAndServe())
	}()

	return nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("http://%s/healthz", a.Config.HTTPListenAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		zlog.Warn("unable to build get health request", zap.Error(err))
		return false
	}

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		zlog.Debug("unable to execute get health request", zap.Error(err))
		return false
	}

	return res.StatusCode == 200
}
