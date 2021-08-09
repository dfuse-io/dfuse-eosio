package booter

import (
	"fmt"
	"os"

	"github.com/dfuse-io/shutter"
	"github.com/streamingfast/dlauncher/launcher"
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

	if !fileExists(a.config.BootSeqFile) {
		zlog.Info("boot sequence file does not exist, continue without booting", zap.String("bootseq_file", a.config.BootSeqFile))
		return nil
	}

	b := newBooter(a.config)

	a.OnTerminating(b.Shutdown)
	b.OnTerminated(a.Shutdown)

	go b.Launch()

	return nil
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
