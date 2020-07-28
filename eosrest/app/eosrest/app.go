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

package eosrest

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dfuse-io/dauth/authenticator"
	dauthMiddleware "github.com/dfuse-io/dauth/authenticator/middleware"
	_ "github.com/dfuse-io/dauth/authenticator/null" // auth plugin
	_ "github.com/dfuse-io/dauth/ratelimiter/null"   // ratelimiter plugin
	"github.com/dfuse-io/dfuse-eosio/eosrest"
	"github.com/dfuse-io/dfuse-eosio/eosrest/rest"
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/logging"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/shutter"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Config struct {
	HTTPListenAddr string

	NodeosRPCEndpoint string

	KVDBDSN       string
	BlockmetaAddr string
	SearchAddr    string
	FluxHTTPAddr  string

	MeteringPlugin string
	AuthPlugin     string
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
	zlog.Info("running eosrest app", zap.Reflect("config", a.Config))

	meter, err := dmetering.New(a.Config.MeteringPlugin)
	if err != nil {
		return fmt.Errorf("metering setup: %w", err)
	}
	dmetering.SetDefaultMeter(meter)

	kdb, err := trxdb.New(a.Config.KVDBDSN, trxdb.WithLogger(zlog))
	if err != nil {
		return fmt.Errorf("trxdb setup: %w", err)
	}

	db := eosrest.NewTRXDB(kdb)

	//blockmetaConn, err := dgrpc.NewInternalClient(a.Config.BlockmetaAddr)
	//if err != nil {
	//	return fmt.Errorf("failed getting blockmeta grpc client: %w", err)
	//}

	fluxURLStr := a.Config.FluxHTTPAddr
	if !strings.HasPrefix(fluxURLStr, "http") {
		fluxURLStr = "http://" + fluxURLStr
	}

	fluxClient := fluxdb.NewClient(fluxURLStr, nil)

	//irrFinder := eosrest.NewDBReaderBaseIrrFinder(db)
	//abiGetter := eosrest.NewDefaultABIGetter(fluxClient)
	//accountGetter := eosrest.NewApiAccountGetter(api)

	blockmetaClient, err := pbblockmeta.NewClient(a.Config.BlockmetaAddr)
	if err != nil {
		return fmt.Errorf("blockmeta connection error: %w", err)
	}

	auth, err := authenticator.New(a.Config.AuthPlugin)
	if err != nil {
		return fmt.Errorf("unable to initialize dauth: %w", err)
	}

	authMiddleware := dauthMiddleware.NewAuthMiddleware(auth, eosrest.DfuseErrorHandler).Handler
	corsMiddleware := eosrest.NewCORSMiddleware()

	fluxURL, err := url.Parse(fluxURLStr)
	if err != nil {
		return fmt.Errorf("cannot parse flux address: %w", err)
	}

	fluxProxy := rest.NewReverseProxy(fluxURL, false)

	searchConn, err := dgrpc.NewInternalClient(a.Config.SearchAddr)
	if err != nil {
		return fmt.Errorf("failed getting abi grpc client: %w", err)
	}
	searchClientV1 := pbsearch.NewRouterClient(searchConn)
	searchRouterClient := searchClientV1

	searchQueryHandler := eosrest.NewSearchEngine(db, searchRouterClient)

	// Order of router definitions is important, prefix:(/a/b) must be defined before /a
	router := mux.NewRouter()

	// Root path to return 200
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // needed for transitioning load balancers to /healthz without downtime
		_, _ = w.Write([]byte("ok"))
	})

	// Setup healthz
	healthzRouter := router.PathPrefix("/").Subrouter()
	healthzRouter.Path("/healthz").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok\n"))
	}))

	// Core endpoints
	coreRouter := router.PathPrefix("/").Subrouter()
	coreRouter.Use(eosrest.LoggingMiddleware)
	coreRouter.Use(eosrest.PreTrackingMiddleware)

	chainRouter := coreRouter.PathPrefix("/").Subrouter()
	//wsRouter := coreRouter.PathPrefix("/").Subrouter()
	restRouter := coreRouter.PathPrefix("/").Subrouter()
	fluxRestRouter := coreRouter.PathPrefix("/").Subrouter()
	historyRestRouter := coreRouter.PathPrefix("/").Subrouter()

	/// Chain endpoints
	chainRouter.Use(authMiddleware)
	chainRouter.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zlogger := logging.Logger(r.Context(), zlog)
			tok, err := jwtmiddleware.FromAuthHeader(r)

			fields := []zap.Field{zap.String("url_path", r.URL.Path), zap.Bool("authenticated", err != nil)}
			if err != nil {
				fields = append(fields, zap.String("token", tok))
			}

			zlogger.Debug("performing native EOS chain API call", fields...)

			// Passthrough
			h.ServeHTTP(w, r)
		})
	})

	apiURLStr := a.Config.NodeosRPCEndpoint
	if !strings.HasPrefix(apiURLStr, "http") {
		apiURLStr = "http://" + apiURLStr
	}
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return fmt.Errorf("cannot parse api-addr: %w", err)
	}

	dumbAPIProxy := rest.NewReverseProxy(apiURL, true)
	billedDumbAPIProxy := dmetering.NewMeteringMiddleware(
		dumbAPIProxy,
		meter,
		"eosrest", "Chain RPC",
		true, true,
	)

	chainRouter.PathPrefix("/v1/chain").Handler(billedDumbAPIProxy)

	/// Primary REST API endpoints
	restRouter.Use(authMiddleware)
	restRouter.Use(eosrest.RESTTrackingMiddleware)
	//////////////////////////////////////////////////////////////////////
	// Billable event on REST APIs
	// WARNING: Middleware is **configured** to ONLY track Query Ingress / Egress bytes.
	//          This means that the middleware DOES NOT track Query requests / responses.
	//          Req / Resp (Docs) is counted in the different endpoints
	//////////////////////////////////////////////////////////////////////
	restRouter.Use(dmetering.NewMeteringMiddlewareFuncWithOptions(
		meter,
		"eosrest", "REST API",
		false, true))
	//////////////////////////////////////////////////////////////////////
	restRouter.Path("/v0/search/transactions").Handler(searchQueryHandler)
	restRouter.Path("/v0/block_id/by_time").Handler(rest.BlockTimeHandler(blockmetaClient))
	restRouter.Path("/v0/transactions/{id}").Handler(rest.GetTransactionHandler(db))

	// FluxDB (Chain State) REST API endpoints
	fluxRestRouter.Use(authMiddleware)
	fluxRestRouter.Use(eosrest.RESTTrackingMiddleware)
	//////////////////////////////////////////////////////////////////////
	// Billable event on REST APIs
	// WARNING: Middleware is **configured** to ONLY track Query Ingress / Egress bytes.
	//          This means that the middleware DOES NOT track Query requests / responses.
	//          Req / Resp (Docs) is counted in the different endpoints
	//////////////////////////////////////////////////////////////////////
	fluxRestRouter.Use(dmetering.NewMeteringMiddlewareFuncWithOptions(
		meter,
		"eosrest", "REST API - Chain State",
		false, true))
	//////////////////////////////////////////////////////////////////////
	fluxRestRouter.Path("/v0/state/abi").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/abi/bin_to_json").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/permission_links").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/key_accounts").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/table").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/table/row").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/table_scopes").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/tables/accounts").Handler(fluxProxy)
	fluxRestRouter.Path("/v0/state/tables/scopes").Handler(fluxProxy)

	historyRestRouter.Use(eosrest.RESTTrackingMiddleware)
	historyRestRouter.Path("/v1/history/get_key_accounts").Methods("GET", "POST").Handler(rest.GetKeyAccounts(fluxClient))

	server := &http.Server{Addr: a.Config.HTTPListenAddr, Handler: handlers.CompressHandlerLevel(corsMiddleware(router), gzip.BestSpeed)}

	go func() {
		zlog.Info("serving HTTP", zap.String("listen_addr", a.Config.HTTPListenAddr))
		go a.Shutdown(server.ListenAndServe())
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
