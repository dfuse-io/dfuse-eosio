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
	_ "github.com/dfuse-io/dauth/null" // auth plugin
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/dfuse-eosio/sqlsync"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type Config struct {
	SQLDSN          string
	BlockStreamAddr string
	BlockmetaAddr   string
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

	blocksStore, err := dstore.NewDBinStore(a.Config.SourceStoreURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	fluxClient := fluxdb.NewClient(a.Config.FluxHTTPAddr, http.DefaultTransport)

	db, err := sqlsync.NewDB(a.Config.SQLDSN)
	if err != nil {
		return fmt.Errorf("sql db setup: %w", err)
	}

	var startBlockRef bstream.BlockRef
	var bootstrapRequired bool
	if db.Empty() {
		blockmetaConn, err := dgrpc.NewInternalClient(a.Config.BlockmetaAddr)
		if err != nil {
			return fmt.Errorf("failed getting blockmeta grpc client: %w", err)
		}
		blockidCli := pbblockmeta.NewBlockIDClient(blockmetaConn)
		libResp, err := blockidCli.LIBID(context.Background(), &pbblockmeta.LIBRequest{})
		if err != nil {
			return err
		}
		startBlockRef = bstream.NewBlockRef(libResp.Id, uint64(eos.BlockNum(libResp.Id)))
		bootstrapRequired = true

	} else {
		sb, err := db.GetStartBlock()
		if err != nil {
			return err
		}
		startBlockRef = sb

	}

	sqlSyncer := sqlsync.NewSQLSync(db, fluxClient, a.Config.BlockStreamAddr, blocksStore)

	go func() {
		zlog.Info("starting sql syncer pipeline")
		go a.Shutdown(sqlSyncer.Launch(bootstrapRequired, startBlockRef))
	}()

	httpServer := &http.Server{Addr: a.Config.HTTPListenAddr, Handler: sqlSyncer.HealthzHandler()}
	go func() {
		zlog.Info("serving HTTP healthz", zap.String("listen_addr", a.Config.HTTPListenAddr))
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
