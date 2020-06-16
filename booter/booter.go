package boot

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	eosboot "github.com/eoscanada/eos-go/boot"
	"go.uber.org/zap"
)

type booter struct {
	*shutter.Shutter
	config *Config
	nodeos *eos.API
}

func newBooter(config *Config) *booter {
	return &booter{
		Shutter: shutter.New(),
		config:  config,
		nodeos:  eos.New(config.NodeosAPIAddress),
	}
}

func (b *booter) Launch() (err error) {
	zlog.Info("starting booter",
		zap.Reflect("boot_seq", b.config.BootSeqFile),
		zap.Reflect("nodeos_api_address", b.config.NodeosAPIAddress),
	)
	b.OnTerminating(b.cleanUp)

	b.waitOnNodeosReady()

	zlog.Info("nodeos is ready, starting injection")

	var keybag *eos.KeyBag
	if b.config.VaultPath != "" {
		keybag, err = b.newKeyBagFromVault(b.config.VaultPath)
		if err != nil {
			b.Shutdown(fmt.Errorf("unable to load vault file: %w", err))
			return err
		}
	} else {
		keybag = eos.NewKeyBag()
	}

	if b.config.PrivateKey != "" {
		keybag = eos.NewKeyBag()
		keybag.Add(b.config.PrivateKey)
	}

	// implementing boot sequence
	booter := eosboot.New(
		b.config.BootSeqFile,
		b.nodeos,
		eosboot.WithKeyBag(keybag),
		eosboot.WithCachePath(b.config.CachePath),
	)

	err = booter.Run()
	if err != nil {
		zlog.Error("failed to boot chain", zap.Error(err))
	}

	return nil
}

func (b *booter) cleanUp(err error) {
	zlog.Debug("terminating booter, cleaning up")
}

func (b *booter) waitOnNodeosReady() {
	for {
		out, err := b.nodeos.GetInfo(context.Background())
		if err != nil {
			zlog.Debug("nodeos get info not responding, waiting and trying again")
			time.Sleep(1 * time.Second)
			continue
		}
		zlog.Info("nodeos is ready",
			zap.Reflect("info", out),
		)
		return
	}
}
