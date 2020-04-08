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

package eosio

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/dauth"
	"github.com/dfuse-io/derr"
	eos "github.com/dfuse-io/dfuse-eosio/dgraphql/eos"
	eosResolver "github.com/dfuse-io/dfuse-eosio/dgraphql/eos/resolvers"
	"github.com/dfuse-io/dgraphql"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/kvdb/eosdb"
	pbabicodec "github.com/dfuse-io/pbgo/dfuse/abicodec/eosio/v1"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	pbtokenmeta "github.com/dfuse-io/pbgo/dfuse/tokenmeta/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	HTTPListenAddr  string
	GRPCListenAddr  string
	SearchAddr      string
	SearchAddrV2    string
	ABICodecAddr    string
	BlockMetaAddr   string
	TokenmetaAddr   string
	KVDBDSN         string
	AuthPlugin      string
	MeteringPlugin  string
	NetworkID       string
	OverrideTraceID bool
}

type App struct {
	*shutter.Shutter
	config    *Config
	ReadyFunc func()
}

func New(config *Config) *App {
	return &App{
		Shutter:   shutter.New(),
		config:    config,
		ReadyFunc: func() {},
	}
}

func (a *App) Run() error {
	zlog.Info("starting dgraphql eosio", zap.Reflect("config", a.config))

	auth, err := dauth.New(a.config.AuthPlugin)
	derr.Check("unable to initialize dauth", err)

	meter, err := dmetering.New(a.config.MeteringPlugin)
	derr.Check("unable to initialize dmetering", err)
	dmetering.SetDefaultMeter(meter)

	zlog.Info("creating db reader")
	dbReader, err := eosdb.New(a.config.KVDBDSN)
	if err != nil {
		return fmt.Errorf("invalid eosdb connection info provided: %w", err)
	}

	zlog.Info("creating abicodec grpc client")
	abiConn, err := dgrpc.NewInternalClient(a.config.ABICodecAddr)
	if err != nil {
		return fmt.Errorf("failed getting abi grpc client: %w", err)
	}
	abiClient := pbabicodec.NewDecoderClient(abiConn)

	zlog.Info("creating blockmeta grpc client")
	blockMetaClient, err := pbblockmeta.NewClient(a.config.BlockMetaAddr)
	if err != nil {
		return fmt.Errorf("failed creating blockmeta client: %w", err)
	}

	zlog.Info("creating tokenmeta grpc client")
	tokenmetaConn, err := dgrpc.NewInternalClient(a.config.TokenmetaAddr)
	if err != nil {
		return fmt.Errorf("failed getting token meta grpc client: %w", err)
	}
	tokenmetaClient := pbtokenmeta.NewEOSClient(tokenmetaConn)

	zlog.Info("creating search grpc client")
	searchRouterClient := a.mustSetupSearchClient()

	zlog.Info("configuring resolver and parsing schemas")
	resolver := eosResolver.NewRoot(searchRouterClient, dbReader, blockMetaClient, abiClient, tokenmetaClient)
	schemas, err := dgraphql.NewSchemas(resolver, eos.CommonSchema(), eos.AlphaSchema())
	if err != nil {
		return fmt.Errorf("unable to parse schema: %w", err)
	}

	zlog.Info("starting dgraphql server")
	server := dgraphql.NewServer(
		a.config.GRPCListenAddr,
		a.config.HTTPListenAddr,
		"eos",
		a.config.NetworkID,
		a.config.OverrideTraceID,
		auth,
		meter,
		schemas,
	)

	a.OnTerminating(server.Shutdown)
	server.OnTerminated(a.Shutdown)

	go server.Launch()

	// Move this to where it fits
	a.ReadyFunc()

	return nil
}

func (a *App) mustSetupSearchClient() pbsearch.RouterClient {
	var searchRouterClient pbsearch.RouterClient

	searchConn, err := dgrpc.NewInternalClient(a.config.SearchAddr)
	derr.Check("failed getting abi grpc client", err)
	searchClientV1 := pbsearch.NewRouterClient(searchConn)

	if a.config.SearchAddrV2 != "" {
		searchConnv2, err := dgrpc.NewInternalClient(a.config.SearchAddrV2)
		if err != nil {
			zlog.Warn("failed getting abi grpc client", zap.Error(err))
		}
		searchClientV2 := pbsearch.NewRouterClient(searchConnv2)
		searchRouterClient = dgraphql.NewMultiRouterClient(searchClientV1, searchClientV2)
	} else {
		searchRouterClient = searchClientV1
	}
	return searchRouterClient
}

func (a *App) OnReady(f func()) {
	a.ReadyFunc = f
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
