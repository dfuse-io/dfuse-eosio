package accounthist

import (
	"errors"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/grpc"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type startFunc func()
type stopFunc func(error)

type Config struct {
	KvdbDSN              string
	GRPCListenAddr       string
	BlocksStoreURL       string //FileSourceBaseURL
	BlockstreamAddr      string // LiveSourceAddress
	ShardNum             byte
	MaxEntriesPerAccount uint64
	FlushBlocksInterval  uint64
	EnableInjector       bool
	EnableServer         bool

	StartBlockNum uint64
	StopBlockNum  uint64
}

type Modules struct {
	BlockFilter func(blk *bstream.Block) error
	Tracker     *bstream.Tracker
}

type App struct {
	*shutter.Shutter
	config  *Config
	modules *Modules

	service *accounthist.Service
}

func New(config *Config, modules *Modules) *App {
	app := &App{
		Shutter: shutter.New(),
		config:  config,
		modules: modules,
	}

	return app
}

func (a *App) Run() error {
	zlog.Info("starting accounthist app", zap.Reflect("config", a.config))
	if err := a.config.validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	kvdb, err := store.New(a.config.KvdbDSN)
	if err != nil {
		zlog.Fatal("could not create kvstore", zap.Error(err))
	}

	if true {
		kvdb = accounthist.NewRWCache(kvdb)
	}

	blocksStore, err := dstore.NewDBinStore(a.config.BlocksStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}

	service := accounthist.NewService(
		kvdb,
		blocksStore,
		a.modules.BlockFilter,
		a.config.ShardNum,
		a.config.MaxEntriesPerAccount,
		a.config.FlushBlocksInterval,
		a.config.StartBlockNum,
		a.config.StopBlockNum,
		a.modules.Tracker,
	)

	if a.config.EnableServer {
		server := grpc.New(a.config.GRPCListenAddr, service)

		a.OnTerminating(server.Terminate)
		server.OnTerminated(a.Shutdown)

		go server.Serve()
	}

	if a.config.EnableInjector {
		if err = service.SetupSource(); err != nil {
			return fmt.Errorf("error setting up source: %w", err)
		}

		a.OnTerminating(service.Shutdown)
		service.OnTerminated(a.Shutdown)

		go service.Launch()
	}

	return nil
}

func (c *Config) validate() error {
	if !c.EnableInjector && !c.EnableServer {
		return errors.New("both enable injection and enable server were disabled, this is invalid, at least one of them must be enabled, or both")
	}

	return nil
}
