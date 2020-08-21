package wallet_api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	wallet "github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/server"
	"github.com/dfuse-io/shutter"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type startFunc func()
type stopFunc func(error)

type Config struct {
	KvdbDSN        string
	GRPCListenAddr string

	SourceStoreURL string //FileSourceBaseURL

	BlockstreamAddr string // LiveSourceAddress
}

type App struct {
	*shutter.Shutter

	Config *Config

	walletStore *wallet.Store
	httpServer  *http.Server

	shutdownFuncs []stopFunc
}

func New(config *Config) *App {
	kvdb, err := store.New(config.KvdbDSN)
	if err != nil {
		zlog.Fatal("could not create kvstore", zap.Error(err))
	}
	walletStore := wallet.NewStore(kvdb)

	app := &App{
		Config:      config,
		Shutter:     shutter.New(),
		walletStore: walletStore,

		shutdownFuncs: make([]stopFunc, 0, 2),
	}

	return app
}

func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	blockStreamStart, blockStreamStop, err := a.blockService(ctx)
	if err != nil {
		return fmt.Errorf("could not launch subscription hub: %w", err)
	}
	a.shutdownFuncs = append(a.shutdownFuncs, blockStreamStop)
	blockStreamStart()

	httpServeStart, httpServeStop, err := a.httpService()
	if err != nil {
		return fmt.Errorf("could not launch subscription hub: %w", err)
	}
	a.shutdownFuncs = append(a.shutdownFuncs, httpServeStop)
	httpServeStart()

	go a.handleSignals()

	a.shutdownFuncs = append(a.shutdownFuncs, func(_ error) { cancel() })

	<-ctx.Done()
	return nil
}

func (a *App) Stop(err error) {
	if err != nil {
		zlog.Info("app encountered error. shutting down.", zap.Error(err))
	}

	for _, f := range a.shutdownFuncs {
		f(err)
	}
}

func (a *App) httpService() (startFunc, stopFunc, error) {
	// Change to a gRPC service
	httpServer := server.New(a.Config.GRPCListenAddr, a.walletStore)

	start := func() {
		go func() {
			err := httpServer.Serve()
			if err != http.ErrServerClosed {
				a.Stop(err)
			}
			a.Stop(nil)
		}()
	}

	stop := func(err error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Stop(ctx)
	}
	return start, stop, nil
}

func (a *App) blockService(ctx context.Context) (startFunc, stopFunc, error) {
	lastProcessedBlock, err := a.walletStore.GetLastProcessedBlock(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get last processed block: %w", err)
	}

	// TODO: configure this!

	//liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
	//	return blockstream.NewSource(ctx, a.Config.BlockstreamAddr, 300, h, blockstream.WithName("wallet-api"))
	//})
	//
	//blockStore, err := dstore.NewStore(a.Config.FileSourceBaseURL, a.Config.FileSourceExtension, a.Config.FileSourceCompressionType, a.Config.FileSourceOverwrite)
	//if err != nil {
	//	return fmt.Errorf("could not create block store: %w", err)
	//}

	//fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
	//	src := bstream.NewFileSource(blockStore, startBlockNum, 1, nil, h)
	//	return src
	//})

	// mock version
	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return bstream.NewMockSource(nil, h)
	})

	// mock version
	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return bstream.NewMockSource(nil, h)
	})

	buffer := bstream.NewBuffer("wallet-api", zlog)
	tailManager := bstream.NewSimpleTailManager(buffer, 10)
	subscriptionHub, err := hub.NewSubscriptionHub(lastProcessedBlock, buffer, tailManager.TailLock, fileSourceFactory, liveSourceFactory)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create subscription hub: %w", err)
	}

	blockHandler := a.walletStore.GetBlockHandler(ctx)
	gate := forkable.NewIrreversibleBlockNumGate(lastProcessedBlock, bstream.GateInclusive, blockHandler) // only process irreversible blocks?
	source := subscriptionHub.NewSourceFromBlockNum(lastProcessedBlock, gate)

	source.OnTerminated(func(err error) {
		a.Stop(err)
	})

	start := func() {
		go subscriptionHub.Launch()
		go tailManager.Launch()
		go source.Run()
	}

	stop := func(err error) {
		source.Shutdown(err)
	}

	return start, stop, nil
}

func (a *App) handleSignals() {
	signalStream := make(chan os.Signal)
	signal.Notify(signalStream, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	sig := <-signalStream
	zlog.Info("received signal. shutting down.", zap.String("signal", sig.String()))

	a.Stop(nil)
}
