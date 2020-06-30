package filtering

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	RelayerAddr    string
	GRPCListenAddr string
	FilterIn       string
	FilterOut      string
}

type App struct {
	*shutter.Shutter
	config   *Config
	launcher *launcher.Launcher
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
	}
}

func (a *App) Run() error {
	zlog.Info("running filtering", zap.Reflect("config", a.config))

	if a.config.RelayerAddr == "" {
		return fmt.Errorf("relayer addr is mandatory field")
	}

	if a.config.GRPCListenAddr == "" {
		return fmt.Errorf("grcp listen addr is mandatory field")
	}

	blockFilter, err := filtering.NewBlockFilter(a.config.FilterIn, a.config.FilterOut)
	if err != nil {
		return fmt.Errorf("block filter: %w", err)
	}

	filterer := filtering.NewFilterer(a.config.RelayerAddr, a.config.GRPCListenAddr, blockFilter)

	a.OnTerminating(filterer.Shutdown)
	filterer.OnTerminated(a.Shutdown)

	go filterer.Launch()

	return nil
}
