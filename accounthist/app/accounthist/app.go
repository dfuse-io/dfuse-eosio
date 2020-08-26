package accounthist

import (
	"fmt"
	"net/http"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/server"
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

	service     *accounthist.Service
	httpServer  *http.Server
	blockFilter func(blk *bstream.Block) error

	//shutdownFuncs []stopFunc
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
	conf := a.config

	kvdb, err := store.New(conf.KvdbDSN)
	if err != nil {
		zlog.Fatal("could not create kvstore", zap.Error(err))
	}

	if true {
		kvdb = accounthist.NewRWCache(kvdb)
	}

	blocksStore, err := dstore.NewDBinStore(conf.BlocksStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}

	service := accounthist.NewService(kvdb, blocksStore, a.modules.BlockFilter, a.config.ShardNum, a.config.MaxEntriesPerAccount, a.config.FlushBlocksInterval, a.config.StartBlockNum, a.config.StopBlockNum, a.modules.Tracker)

	if err = service.SetupSource(); err != nil {
		return fmt.Errorf("error setting up source: %w", err)
	}

	server := server.New(conf.GRPCListenAddr, service)
	go server.Serve()

	// FIXME: what's in a go routine, what's in `Launch()`, which returns an error, dunno dunno!

	a.OnTerminating(service.Shutdown)
	service.OnTerminated(a.Shutdown)

	go service.Launch()

	return nil
}
