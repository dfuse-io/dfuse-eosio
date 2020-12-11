package firehose

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	blockstreamv2 "github.com/dfuse-io/bstream/blockstream/v2"
	"github.com/dfuse-io/bstream/hub"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dgraphql/insecure"
	"github.com/dfuse-io/dgraphql/metrics"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/dstore"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/shutter"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Config struct {
	BlocksStoreURL          string
	UpstreamBlockStreamAddr string
	GRPCListenAddr          string
	BlockmetaAddr           string
}

type Modules struct {
	Tracker *bstream.Tracker
}

type App struct {
	*shutter.Shutter
	config    *Config
	modules   *Modules
	ReadyFunc func()
	isReady   func() bool
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter:   shutter.New(),
		config:    config,
		modules:   modules,
		ReadyFunc: func() {},
	}
}

func (a *App) Run() error {
	dmetrics.Register(metrics.MetricSet)
	zlog.Info("running block stream", zap.Reflect("config", a.config))
	blocksStore, err := dstore.NewDBinStore(a.config.BlocksStoreURL)
	if err != nil {
		return fmt.Errorf("failed setting up blocks store: %w", err)
	}

	ctx := context.Background()
	start := uint64(0)
	if a.config.UpstreamBlockStreamAddr != "" {
		zlog.Info("starting with support for live blocks")
		zlog.Debug("getting relative block", zap.Int("relative_to", -200))
		for retries := 0; ; retries++ {
			almostLastBlock, err := a.modules.Tracker.GetRelativeBlock(ctx, -200, bstream.BlockStreamHeadTarget)
			if err != nil {
				if retries%5 == 4 {
					zlog.Warn("cannot get 'almostLastBlock', retrying", zap.Int("retries", retries), zap.Error(err))
					time.Sleep(time.Second)
				}
				continue
			}
			zlog.Info("get almost last block", zap.Uint64("block_num", almostLastBlock))
			start, _, err = a.modules.Tracker.ResolveStartBlock(ctx, almostLastBlock)
			if err != nil {
				if retries%5 == 4 {
					zlog.Warn("cannot resolve start block 'almostLastBlock'", zap.Int("retries", retries), zap.Error(err))
					time.Sleep(time.Second)
				}
				continue
			}
			break
		}
	}

	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return blockstream.NewSource(
			context.Background(),
			a.config.UpstreamBlockStreamAddr,
			100,
			bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
				metrics.HeadTimeDrift.SetBlockTime(blk.Time())
				return h.ProcessBlock(blk, obj)
			}),
			blockstream.WithRequester("blockstream"),
		)
	})

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		zlog.Info("creating file source", zap.Uint64("start_block_num", startBlockNum))
		src := bstream.NewFileSource(blocksStore, startBlockNum, 1, nil, h)
		return src
	})

	zlog.Info("setting up subscription hub")

	buffer := bstream.NewBuffer("hub-buffer", zlog.Named("hub"))
	tailManager := bstream.NewSimpleTailManager(buffer, 350)
	tailManager.Launch()
	subscriptionHub, err := hub.NewSubscriptionHub(
		start,
		buffer,
		tailManager.TailLock,
		fileSourceFactory,
		liveSourceFactory,
		hub.Withlogger(zlog),
		hub.WithRealtimeTolerance(1*time.Minute),
		hub.WithoutMemoization(), // This should be tweakable on the Hub, by the bstreamv2.Server
	)
	if err != nil {
		return fmt.Errorf("setting up subscription hub: %w", err)
	}

	go subscriptionHub.Launch()
	//	subscriptionHub.WaitReady()

	bsv2Tracker := a.modules.Tracker.Clone()

	zlog.Info("setting up blockstream V2 server")

	s := blockstreamv2.NewServer(bsv2Tracker, blocksStore, a.config.GRPCListenAddr, subscriptionHub)
	s.SetPreprocFactory(func(req *pbbstream.BlocksRequestV2) (bstream.PreprocessFunc, error) {
		filter, err := filtering.NewBlockFilter([]string{req.IncludeFilterExpr}, []string{req.ExcludeFilterExpr}, nil)
		if err != nil {
			return nil, fmt.Errorf("parsing: %w", err)
		}
		preproc := &filtering.FilteringPreprocessor{Filter: filter}
		return preproc.PreprocessBlock, nil
	})

	a.isReady = s.IsReady
	// Move this to where it fits
	a.ReadyFunc()

	go func() {
		insecure := strings.Contains(a.config.GRPCListenAddr, "*")
		addr := strings.Replace(a.config.GRPCListenAddr, "*", "", -1)

		if err := startGRPCServer(s, insecure, addr); err != nil {
			a.Shutdown(err)
		}
	}()

	return nil
}

func startGRPCServer(s *blockstreamv2.Server, insecure bool, listenAddr string) error {
	// TODO: this is heavily duplicated with `dgraphql`, eventually should all go to `dgrpc`
	// so we have better exposure of gRPC services inside the mesh, and ways to
	// expose them externally too.
	if insecure {
		return startGRPCServerInsecure(s, listenAddr)
	}
	return startGRPCServerSecure(s, listenAddr)
}

func startGRPCServerSecure(s *blockstreamv2.Server, listenAddr string) error {
	srv := newGRPCServer(s, false)

	grpcListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listening grpc %q: %w", listenAddr, err)
	}

	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		return fmt.Errorf("unable to create logger: %w", err)
	}

	srv.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{insecure.Cert},
		ClientCAs:    insecure.CertPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}
	srv.ErrorLog = errorLogger

	if err := srv.ServeTLS(grpcListener, "", ""); err != nil {
		return fmt.Errorf("grpc server serve tls: %w", err)
	}
	return nil
}

func startGRPCServerInsecure(s *blockstreamv2.Server, listenAddr string) error {
	grpcListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listening grpc %q: %w", listenAddr, err)
	}

	gs := newGRPCServer(s, false)

	zlog.Info("serving gRPC", zap.String("grpc_addr", listenAddr))
	if err := gs.Serve(grpcListener); err != nil {
		return fmt.Errorf("error on gs.Serve: %w", err)
	}
	return nil
}

func newGRPCServer(s *blockstreamv2.Server, overrideTraceID bool) http.Server {
	serverOptions := []dgrpc.ServerOption{dgrpc.WithLogger(zlog)}
	if overrideTraceID {
		serverOptions = append(serverOptions, dgrpc.OverrideTraceID())
	}

	zlog.Info("configuring grpc server")
	gs := dgrpc.NewServer(serverOptions...)
	pbbstream.RegisterBlockStreamV2Server(gs, s)
	//reflection.Register(gs)

	grpcRouter := mux.NewRouter()
	grpcRouter.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// To satisfy GCP's load balancers
		w.Write([]byte("ok"))
	})
	grpcRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gs.ServeHTTP(w, r)
	})

	return http.Server{
		Handler: grpcRouter,
	}
}
