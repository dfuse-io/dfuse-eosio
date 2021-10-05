package booter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	_ "github.com/dfuse-io/dfuse-eosio/booter/migrator"
	_ "github.com/dfuse-io/dfuse-eosio/booter/ultraops"
	eosboot "github.com/dfuse-io/eosio-boot"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("not found")
)

type booter struct {
	*shutter.Shutter
	config *Config
	nodeos *eos.API
}

type state struct {
	Revision string    `json:"revision"`
	BootedAt time.Time `json:"booted_at"`
}

func newBooter(config *Config) *booter {
	return &booter{
		Shutter: shutter.New(),
		config:  config,
		nodeos:  eos.New(config.NodeosAPIAddress),
	}
}

func (b *booter) Launch() {
	zlog.Info("starting booter", zap.Reflect("config", b.config))
	b.OnTerminating(b.cleanUp)
	var err error
	var booterState *state
	if b.stateExists() {
		zlog.Info("retrieving booter state")
		booterState, err = b.getState()
		if err != nil {
			zlog.Error("unable to retrieve booter state", zap.Error(err))
			b.Shutdown(err)
			return
		}
	}

	keybag, err := b.setupKeybag()
	if err != nil {
		zlog.Error("failed to setup keybag", zap.Error(err))
		b.Shutdown(err)
		return
	}

	// implementing boot sequence
	booter, err := eosboot.New(
		b.config.BootSeqFile,
		b.nodeos,
		filepath.Join(b.config.Datadir, "cache"),
		eosboot.WithKeyBag(keybag),
		eosboot.WithLogger(zlog),
	)
	if err != nil {
		zlog.Error("failed to initialize booter", zap.Error(err))
		b.Shutdown(err)
		return
	}

	if booterState != nil && booter.Revision() == booterState.Revision {
		zlog.Info("chain has already been booted",
			zap.String("boot_sequence_revision", booterState.Revision),
			zap.Time("booted_at", booterState.BootedAt),
		)
		return
	}

	b.waitOnNodeosReady()
	zlog.Info("nodeos is ready, starting injection")

	bootSeqChecksum, err := booter.Run()
	if err != nil {
		zlog.Error("failed to boot chain", zap.Error(err))
		b.Shutdown(err)
		return
	}

	zlog.Info("booter successfully ran",
		zap.String("bootseq_checksum", bootSeqChecksum),
	)

	err = b.storeState(bootSeqChecksum)
	if err != nil {
		zlog.Error("failed to store booter state", zap.Error(err))
		b.Shutdown(err)
		return
	}

	return
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

func (b *booter) storeState(bootSeqChecksum string) error {
	zlog.Debug("storing booter state")

	s := &state{
		Revision: bootSeqChecksum,
		BootedAt: time.Now(),
	}

	cnt, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("unable to store booter state: %w", err)
	}

	filename := b.getStateFilePath()
	fl, err := os.OpenFile(b.getStateFilePath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open booter state for writting %q: %w", filename, err)
	}
	defer fl.Close()

	_, err = fl.Write(cnt)
	if err != nil {
		return fmt.Errorf("unable to write booter state %q: %w", filename, err)
	}

	return nil
}

func (b *booter) getState() (*state, error) {
	zlog.Debug("getting booter state")

	rawState, err := ioutil.ReadFile(b.getStateFilePath())
	if err != nil {
		return nil, fmt.Errorf("reading boot seq: %s", err)
	}

	s := &state{}

	if err := json.Unmarshal(rawState, &s); err != nil {
		return nil, fmt.Errorf("parsing booter state: %w", err)
	}
	return s, nil
}

func (b *booter) getStateFilePath() string {
	return filepath.Join(b.config.Datadir, "state.json")
}

func (b *booter) setupKeybag() (keybag *eos.KeyBag, err error) {
	if b.config.VaultPath != "" {
		keybag, err = b.newKeyBagFromVault(b.config.VaultPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load vault file: %w", err)
		}
	} else {
		keybag = eos.NewKeyBag()
	}

	if b.config.PrivateKey != "" {
		keybag = eos.NewKeyBag()
		keybag.Add(b.config.PrivateKey)
	}
	return keybag, nil
}

func (b *booter) stateExists() bool {
	info, err := os.Stat(b.getStateFilePath())
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
