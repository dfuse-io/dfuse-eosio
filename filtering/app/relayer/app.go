package relayer

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	RelayerAddr       string
	GRPCListenAddr    string
	IncludeFilterExpr string
	ExcludeFilterExpr string
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
	zlog.Info("running filtering relayer", zap.Reflect("config", a.config))

	if a.config.RelayerAddr == "" {
		return fmt.Errorf("relayer addr is mandatory field")
	}

	if a.config.GRPCListenAddr == "" {
		return fmt.Errorf("grcp listen addr is mandatory field")
	}

	blockFilter, err := filtering.NewBlockFilter(a.config.IncludeFilterExpr, a.config.ExcludeFilterExpr)
	if err != nil {
		return fmt.Errorf("block filter: %w", err)
	}

	filterer := filtering.NewRelayer(a.config.RelayerAddr, a.config.GRPCListenAddr, blockFilter)

	a.OnTerminating(filterer.Shutdown)
	filterer.OnTerminated(a.Shutdown)

	go filterer.Launch()

	return nil
}
