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

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
package eosws

import (
	"context"
	"fmt"
	drateLimiter "github.com/dfuse-io/dauth/ratelimiter"
	"github.com/dfuse-io/derr"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/dauth/authenticator"
	dauthMiddleware "github.com/dfuse-io/dauth/authenticator/middleware"
	_ "github.com/dfuse-io/dauth/authenticator/null" // auth plugin
	_ "github.com/dfuse-io/dauth/ratelimiter/null"   // ratelimiter plugin
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/completion"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/rest"
	stateHelper "github.com/dfuse-io/dfuse-eosio/eosws/statedb"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dipp"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/logging"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbheadinfo "github.com/dfuse-io/pbgo/dfuse/headinfo/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
type Config struct {
	HTTPListenAddr      string
	NodeosRPCEndpoint   string
	BlockmetaAddr       string
	KVDBDSN             string
	BlockStreamAddr     string
	SourceStoreURL      string
	SearchAddr          string
	SearchAddrSecondary string
	StateDBHTTPAddr     string
	StateDBGRPCAddr     string

	AuthenticateNodeosAPI bool

	MeteringPlugin           string
	AuthPlugin               string
	RatelimiterPlugin        string
	UseOpencensusStackdriver bool

	FetchPrice     bool
	FetchVoteTally bool
	WithCompletion bool

	FilesourceRateLimitPerBlock time.Duration
	BlocksBufferSize            int
	RealtimeTolerance           time.Duration

	DataIntegrityProofSecret string
	HealthzSecret            string

	DisabledWsMessage map[string]interface{}
}

type Modules struct {
	BlockFilter func(blk *bstream.Block) error
}

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
type App struct {
	*shutter.Shutter
	Config  *Config
	Modules *Modules
}

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
func New(config *Config, modules *Modules) *App {
	eosws.DisabledWsMessage = config.DisabledWsMessage
	return &App{
		Shutter: shutter.New(),
		Config:  config,
		Modules: modules,
	}

}

// Deprecated: The features in the eosws package will be moved to other packages like Dgraphql
func (a *App) Run() error {
	zlog.Info("running eosws app", zap.Reflect("config", a.Config))

	dmetrics.Register(metrics.Metricset)
	meter, err := dmetering.New(a.Config.MeteringPlugin)
	if err != nil {
		return fmt.Errorf("metering setup: %w", err)
	}
	dmetering.SetDefaultMeter(meter)

	ctx, cancel := context.WithCancel(context.Background())
	a.OnTerminating(func(_ error) { cancel() })

	apiURLStr := a.Config.NodeosRPCEndpoint
	if !strings.HasPrefix(apiURLStr, "http") {
		apiURLStr = "http://" + apiURLStr
	}
	api := eos.New(apiURLStr)

	rateLimiter, err := drateLimiter.New(a.Config.RatelimiterPlugin)
	derr.Check("unable to initialize rate limiter", err)

	kdb, err := trxdb.New(a.Config.KVDBDSN, trxdb.WithLogger(zlog))
	if err != nil {
		return fmt.Errorf("trxdb setup: %w", err)
	}

	if d, ok := kdb.(trxdb.Debugeable); ok {
		zlog.Info("trxdb dsn", zap.String("DSN", a.Config.KVDBDSN))
		d.Dump()
	} else {
		zlog.Info("trxdb driver database is not debugeable")
	}

	db := eosws.NewTRXDB(kdb)

	blocksStore, err := dstore.NewDBinStore(a.Config.SourceStoreURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	var preprocessor bstream.PreprocessFunc
	if a.Modules.BlockFilter != nil {
		preprocessor = bstream.PreprocessFunc(func(blk *bstream.Block) (interface{}, error) {
			return nil, a.Modules.BlockFilter(blk)
		})
	}

	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return blockstream.NewSource(ctx, a.Config.BlockStreamAddr, 300, h, blockstream.WithRequester("eosws"))
	})

	buffer := bstream.NewBuffer("sub-hub", zlog)
	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		src := bstream.NewFileSource(blocksStore, startBlockNum, 1, preprocessor, h)
		return src
	})

	blockmetaConn, err := dgrpc.NewInternalClient(a.Config.BlockmetaAddr)
	if err != nil {
		return fmt.Errorf("failed getting blockmeta grpc client: %w", err)
	}
	headinfoCli := pbheadinfo.NewHeadInfoClient(blockmetaConn)

	var head bstream.BlockRef
	var lib bstream.BlockRef
	for {
		hi, err := headinfoCli.GetHeadInfo(ctx, &pbheadinfo.HeadInfoRequest{Source: pbheadinfo.HeadInfoRequest_STREAM})
		if err != nil || hi == nil {
			select {
			case <-time.After(1 * time.Second):
			case <-a.Shutter.Terminating():
				return nil
			}
			continue
		}
		head = bstream.NewBlockRef(hi.HeadID, hi.HeadNum)
		lib = bstream.NewBlockRef(hi.LibID, hi.LibNum)
		break
	}

	var hubStartBlockNum uint64
	halfBufferSize := uint64(a.Config.BlocksBufferSize / 2)
	if lib.Num() <= halfBufferSize {
		hubStartBlockNum = 2
	} else {
		hubStartBlockNum = lib.Num() - halfBufferSize
	}

	tailManager := bstream.NewSimpleTailManager(buffer, a.Config.BlocksBufferSize)
	subscriptionHub, err := hub.NewSubscriptionHub(
		hubStartBlockNum,
		buffer,
		tailManager.TailLock,
		fileSourceFactory,
		liveSourceFactory,
		hub.Withlogger(zlog),
	)
	if err != nil {
		return fmt.Errorf("could not create subscription hub: %w", err)
	}
	go subscriptionHub.Launch()
	go tailManager.Launch()

	stateConn, err := dgrpc.NewInternalClient(a.Config.StateDBGRPCAddr)
	if err != nil {
		return fmt.Errorf("failed getting statedb grpc conn: %w", err)
	}
	stateClient := pbstatedb.NewStateClient(stateConn)

	voteTallyHub := eosws.NewVoteTallyHub(stateHelper.NewDefaultFluxHelper(stateClient))
	if a.Config.FetchVoteTally {
		go voteTallyHub.Launch(context.Background())
	}

	headInfoHub := eosws.NewHeadInfoHub(head.ID(), lib.ID(), subscriptionHub)

	priceHub := eosws.NewPriceHub()
	if a.Config.FetchPrice {
		go priceHub.Launch(context.Background())
	}

	irrFinder := eosws.NewDBReaderBaseIrrFinder(db)

	abiGetter := eosws.NewDefaultABIGetter(stateClient)
	accountGetter := eosws.NewApiAccountGetter(api)

	blockmetaClient, err := pbblockmeta.NewClient(a.Config.BlockmetaAddr)
	if err != nil {
		return fmt.Errorf("blockmeta connection error: %w", err)
	}

	wsHandler := eosws.NewWebsocketHandler(abiGetter, accountGetter, db, subscriptionHub, stateClient, voteTallyHub, headInfoHub, priceHub, irrFinder, a.Config.FilesourceRateLimitPerBlock)

	auth, err := authenticator.New(a.Config.AuthPlugin)
	if err != nil {
		return fmt.Errorf("unable to initialize dauth: %w", err)
	}

	authMiddleware := dauthMiddleware.NewAuthMiddleware(auth, eosws.DfuseErrorHandler).Handler
	corsMiddleware := eosws.NewCORSMiddleware()
	compressionMiddleware := mux.MiddlewareFunc(eosws.CompressionMiddleware)
	rateLimiterMiddleware := eosws.NewAuthFeatureMiddleware(func(ctx context.Context, credentials authenticator.Credentials) error {

		// todo replace ith r.URL.Path to get more granular blocking possibilities
		method := "eosws"
		if !rateLimiter.Gate(credentials.GetUserID(), method) {
			return eosws.RateLimitTooManyRequests(ctx)
		}

		return nil
	}).Handler
	hasEosqTierMiddleware := eosws.NewAuthFeatureMiddleware(func(ctx context.Context, credentials authenticator.Credentials) error {
		type authTier interface {
			AuthenticatedTier() string
		}
		if c, ok := credentials.(authTier); ok {
			if tier := c.AuthenticatedTier(); tier != "eosq-v1" {
				return eosws.AuthInvalidTierError(ctx, tier, "eosq-v1")
			}
		}

		return nil
	}).Handler

	stateHTTPAddr := a.Config.StateDBHTTPAddr
	if !strings.HasPrefix(stateHTTPAddr, "http") {
		stateHTTPAddr = "http://" + stateHTTPAddr
	}

	stateHTTPURL, err := url.Parse(stateHTTPAddr)
	if err != nil {
		return fmt.Errorf("cannot parse statedb HTTP address: %w", err)
	}

	statedbProxy := rest.NewReverseProxy(stateHTTPURL, false)

	var searchRouterClient pbsearch.RouterClient

	searchConn, err := dgrpc.NewInternalClient(a.Config.SearchAddr)
	if err != nil {
		return fmt.Errorf("failed getting abi grpc client: %w", err)
	}
	searchClientV1 := pbsearch.NewRouterClient(searchConn)

	if a.Config.SearchAddrSecondary != "" {
		zlog.Info("setting up secondary search router")
		searchConnv2, err := dgrpc.NewInternalClient(a.Config.SearchAddrSecondary)
		if err != nil {
			zlog.Warn("failed getting abi grpc client", zap.Error(err))
		}
		searchClientV2 := pbsearch.NewRouterClient(searchConnv2)
		zlog.Info("search client will be a MultiRouterClient")
		multiRouterClient := eosws.NewMultiRouterClient(searchClientV1, searchClientV2)
		go func() {
			zlog.Info("starting atomic level switcher, port :1066")
			if err := http.ListenAndServe(":1066", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				multiRouterClient.Toggle.Toggle()
				w.Write([]byte(fmt.Sprintf("switch toggles: %t", multiRouterClient.Toggle.Load())))
			})); err != nil {
				zlog.Info("failed listening on :1066 to switch multi search router:", zap.Error(err))
			}
		}()
		searchRouterClient = multiRouterClient
	} else {
		searchRouterClient = searchClientV1
	}

	searchQueryHandler := eosws.NewSearchEngine(db, searchRouterClient)

	// Order of router definitions is important, prefix:(/a/b) must be defined before /a
	router := mux.NewRouter()

	// Root path to return 200, needed for transitioning load balancers to /healthz without downtime
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { //
		_, _ = w.Write([]byte("ok"))
	})

	// Setup healthz
	healthzHandler := rest.HealthzHandler(subscriptionHub, api, blocksStore, db, stateClient, searchQueryHandler, a.Config.HealthzSecret)
	healthzRouter := router.PathPrefix("/").Subrouter()
	healthzRouter.Path("/healthz").Handler(healthzHandler)

	// Setup simple check to determine if search is stuck, workaround for elusive bug
	searchNotStuckHandler := rest.SearchNotStuckHandler(searchQueryHandler)
	healthzRouter.Path("/search_not_stuck").Handler(searchNotStuckHandler)

	// Core endpoints
	coreRouter := router.PathPrefix("/").Subrouter()
	coreRouter.Use(eosws.OpenCensusMiddleware)
	coreRouter.Use(eosws.LoggingMiddleware)
	coreRouter.Use(eosws.PreTrackingMiddleware)

	chainRouter := coreRouter.PathPrefix("/").Subrouter()
	wsRouter := coreRouter.PathPrefix("/").Subrouter()
	restRouter := coreRouter.PathPrefix("/").Subrouter()
	statedbRestRouter := coreRouter.PathPrefix("/").Subrouter()
	historyRestRouter := coreRouter.PathPrefix("/").Subrouter()
	eosqRestRouter := coreRouter.PathPrefix("/").Subrouter()

	/// Chain endpoints
	if a.Config.AuthenticateNodeosAPI {
		chainRouter.Use(authMiddleware)
	}

	chainRouter.Use(compressionMiddleware)
	chainRouter.Use(dipp.NewProofMiddlewareFunc(a.Config.DataIntegrityProofSecret))
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

	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return fmt.Errorf("cannot parse api-addr: %w", err)
	}

	dumbAPIProxy := rest.NewReverseProxy(apiURL, true)
	billedDumbAPIProxy := dmetering.NewMeteringMiddleware(
		dumbAPIProxy,
		meter,
		"eosws", "Chain RPC",
		true, true,
	)

	authTxPusher := dauthMiddleware.NewAuthMiddleware(auth, eosws.EOSChainErrorHandler).Handler(
		dmetering.NewMeteringMiddleware(
			rest.NewTxPusher(api, subscriptionHub),
			meter,
			"eosws", "Push Transaction",
			true, true,
		),
	)
	txPushRouter := rest.NewTxPushRouter(billedDumbAPIProxy, authTxPusher)
	chainRouter.PathPrefix("/v1/chain").Handler(txPushRouter)

	/// WebSocket endpoints
	wsRouter.Use(authMiddleware)
	wsRouter.Use(rateLimiterMiddleware)
	wsRouter.Path("/v1/stream").Handler(wsHandler)

	/// Primary REST API endpoints
	restRouter.Use(compressionMiddleware)
	restRouter.Use(authMiddleware)
	restRouter.Use(rateLimiterMiddleware)
	restRouter.Use(eosws.RESTTrackingMiddleware)
	restRouter.Use(dipp.NewProofMiddlewareFunc(a.Config.DataIntegrityProofSecret))
	//////////////////////////////////////////////////////////////////////
	// Billable event on REST APIs
	// WARNING: Middleware is **configured** to ONLY track Query Ingress / Egress bytes.
	//          This means that the middleware DOES NOT track Query requests / responses.
	//          Req / Resp (Docs) is counted in the different endpoints
	//////////////////////////////////////////////////////////////////////
	restRouter.Use(dmetering.NewMeteringMiddlewareFuncWithOptions(
		meter,
		"eosws", "REST API",
		false, true))
	//////////////////////////////////////////////////////////////////////

	restRouter.Path("/v0/search/transactions").Handler(searchQueryHandler)
	restRouter.Path("/v0/block_id/by_time").Handler(rest.BlockTimeHandler(blockmetaClient))
	restRouter.Path("/v0/transactions/{id}").Handler(rest.GetTransactionHandler(db))

	// FluxDB (Chain State) REST API endpoints
	statedbRestRouter.Use(compressionMiddleware)
	statedbRestRouter.Use(authMiddleware)
	statedbRestRouter.Use(rateLimiterMiddleware)
	statedbRestRouter.Use(eosws.RESTTrackingMiddleware)
	statedbRestRouter.Use(dipp.NewProofMiddlewareFunc(a.Config.DataIntegrityProofSecret))
	//////////////////////////////////////////////////////////////////////
	// Billable event on REST APIs
	// WARNING: Middleware is **configured** to ONLY track Query Ingress / Egress bytes.
	//          This means that the middleware DOES NOT track Query requests / responses.
	//          Req / Resp (Docs) is counted in the different endpoints
	//////////////////////////////////////////////////////////////////////
	statedbRestRouter.Use(dmetering.NewMeteringMiddlewareFuncWithOptions(
		meter,
		"eosws", "REST API - Chain State",
		false, true))
	//////////////////////////////////////////////////////////////////////
	statedbRestRouter.Path("/v0/state/abi").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/abi/bin_to_json").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/permission_links").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/key_accounts").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/table").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/table/row").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/table_scopes").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/tables/accounts").Handler(statedbProxy)
	statedbRestRouter.Path("/v0/state/tables/scopes").Handler(statedbProxy)

	historyRestRouter.Use(compressionMiddleware)
	historyRestRouter.Use(eosws.RESTTrackingMiddleware)
	historyRestRouter.Path("/v1/history/get_key_accounts").Methods("GET", "POST").Handler(rest.GetKeyAccounts(stateClient))

	/// Rest routes (Eosq accessible only)
	eosqRestRouter.Use(compressionMiddleware)
	eosqRestRouter.Use(authMiddleware)
	eosqRestRouter.Use(rateLimiterMiddleware)
	eosqRestRouter.Use(hasEosqTierMiddleware)
	eosqRestRouter.Use(eosws.RESTTrackingMiddleware)

	//////////////////////////////////////////////////////////////////////
	// Billable event on EOSQ APIs
	// WARNING: Middleware is **configured** to ONLY track Query Ingress / Egress bytes.
	//          This means that the middleware DOES NOT track Query requests / responses.
	//          Req / Resp (Docs) is counted in the different endpoints
	//////////////////////////////////////////////////////////////////////
	eosqRestRouter.Use(dmetering.NewMeteringMiddlewareFuncWithOptions(
		meter,
		"eosws", "REST API - eosq",
		false, true))
	//////////////////////////////////////////////////////////////////////

	eosqRestRouter.Path("/v0/transactions").Handler(rest.ListTransactionsHandler(db))

	eosqRestRouter.Path("/v0/blocks").Handler(rest.GetBlocksHandler(db))
	eosqRestRouter.Path("/v0/blocks/{blockID}").Handler(rest.GetBlockHandler(db))
	eosqRestRouter.Path("/v0/blocks/{blockID}/transactions").Handler(rest.GetBlockTransactionsHandler(db))
	eosqRestRouter.Path("/v0/simple_search").Handler(rest.SimpleSearchHandler(db, blockmetaClient))

	zlog.Info("waiting for subscription hub to reach expected head block")
	retryDelay := time.Duration(0)
	for {
		select {
		case <-time.After(retryDelay):
			retryDelay = 100 * time.Millisecond
		case <-a.Terminating():
			return nil
		}
		headBlock := subscriptionHub.HeadBlock()
		if headBlock == nil {
			continue
		}
		if headBlock.Num() < head.Num() {
			continue
		}
		break
	}
	go headInfoHub.Launch(context.Background())

	if a.Config.WithCompletion {
		completionInstance, err := completion.New(ctx, db)
		if err != nil {
			return fmt.Errorf("unable to initialize completion: %w", err)
		}
		completionPipeline := completion.NewPipeline(completionInstance, head.ID(), lib.ID(), subscriptionHub)
		eosqRestRouter.Path("/v0/search/completion").Handler(rest.GetCompletionHandler(completionInstance))
		go completionPipeline.Launch()
	}

	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		return fmt.Errorf("unable to create error logger: %w", err)
	}

	server := &http.Server{
		Addr:     a.Config.HTTPListenAddr,
		Handler:  corsMiddleware(router),
		ErrorLog: errorLogger,
	}

	go func() {
		zlog.Info("serving HTTP", zap.String("listen_addr", a.Config.HTTPListenAddr))
		go a.Shutdown(server.ListenAndServe())
	}()

	return nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("http://%s/healthz?secret=%s", a.Config.HTTPListenAddr, a.Config.HealthzSecret)
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
