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

package abicodec

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/dfuse-eosio/abicodec"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/streamingfast/dgrpc"
	"github.com/streamingfast/dstore"
	pbhealth "github.com/streamingfast/pbgo/grpc/health/v1"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type Config struct {
	GRPCListenAddr     string
	SearchAddr         string
	KvdbDSN            string
	CacheBaseURL       string
	CacheStateName     string
	ExportABIsEnabled  bool
	ExportABIsBaseURL  string
	ExportABIsFilename string
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
	zlog.Info("running abicodec cache", zap.Reflect("config", a.config))

	zlog.Info("initiating cache", zap.String("", a.config.CacheBaseURL), zap.String("cache_state_name", a.config.CacheStateName))
	store, err := dstore.NewSimpleStore(a.config.CacheBaseURL)
	if err != nil {
		return fmt.Errorf("unable to init store: %w", err)
	}

	cache, err := abicodec.NewABICache(store, a.config.CacheStateName)
	if err != nil {
		return fmt.Errorf("unable to init ABI cache: %w", err)
	}

	backuper := abicodec.NewBackuper(cache, a.config.ExportABIsEnabled, a.config.ExportABIsBaseURL, a.config.ExportABIsFilename)
	go backuper.BackupPeriodically(30 * time.Second)

	backuper.OnTerminated(a.Shutdown)
	a.OnTerminating(backuper.Shutdown)

	dbReader, err := trxdb.New(a.config.KvdbDSN, trxdb.WithLogger(zlog))
	if err != nil {
		return fmt.Errorf("unable to init KVDB connection: %w", err)
	}

	server := abicodec.NewServer(cache, a.config.GRPCListenAddr)

	server.OnTerminated(a.Shutdown)
	a.OnTerminating(server.Shutdown)

	go server.Serve()

	onLive := func() {
		backuper.IsLive = true
		server.SetReady()
	}
	syncer, err := abicodec.NewSyncer(cache, dbReader, a.config.SearchAddr, onLive)
	if err != nil {
		return fmt.Errorf("unable to create ABI syncer: %w", err)
	}

	syncer.OnTerminated(a.Shutdown)
	a.OnTerminating(syncer.Shutdown)

	go syncer.Sync()

	gs, err := dgrpc.NewInternalClient(a.config.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("cannot create readiness probe")
	}
	a.readinessProbe = pbhealth.NewHealthClient(gs)

	return nil
}

func (a *App) IsReady() bool {
	if a.readinessProbe == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := a.readinessProbe.Check(ctx, &pbhealth.HealthCheckRequest{})
	if err != nil {
		zlog.Info("abicodec readiness probe error", zap.Error(err))
		return false
	}

	if resp.Status == pbhealth.HealthCheckResponse_SERVING {
		return true
	}

	return false
}
