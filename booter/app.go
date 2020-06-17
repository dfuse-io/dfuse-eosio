package booter

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	NodeosAPIAddress string
	BootSeqFile      string
	Datadir          string
	VaultPath        string
	PrivateKey       string
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
	zlog.Info("running booter", zap.Reflect("config", a.config))

	if (a.config.NodeosAPIAddress == "") && (a.config.BootSeqFile != "") {
		return fmt.Errorf("cannot inject bootsequence without a nodeos api address")
	}

	if a.config.BootSeqFile == "" {
		zlog.Info("no boot sequence specified, booter will not launch")
		return nil
	}

	b := newBooter(a.config)

	a.OnTerminating(b.Shutdown)
	b.OnTerminated(a.Shutdown)

	go b.Launch()

	return nil
}
