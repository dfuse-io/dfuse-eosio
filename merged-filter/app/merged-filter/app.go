package merged_filter

import (
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	mergedFilter "github.com/dfuse-io/dfuse-eosio/merged-filter"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type App struct {
	*shutter.Shutter
	config *Config
}

type Config struct {
	DestBlocksStoreURL   string
	SourceBlocksStoreURL string

	BlockstreamAddr string

	BatchMode       bool
	BatchStartBlock uint64
	BatchStopBlock  uint64

	TruncationEnabled bool
	TruncationWindow  uint64

	IncludeFilterExpr string
	ExcludeFilterExpr string
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
	}
}

func (a *App) Run() error {
	srcBlocksStore, err := dstore.NewDBinStore(a.config.SourceBlocksStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}

	zlog.Info("reading from store", zap.String("store_url", a.config.SourceBlocksStoreURL))

	destBlocksStore, err := dstore.NewDBinStore(a.config.DestBlocksStoreURL)
	if err != nil {
		return fmt.Errorf("setting up archive store: %w", err)
	}

	zlog.Info("writing to store", zap.String("store_url", a.config.DestBlocksStoreURL))

	blockFilter, err := filtering.NewBlockFilter(a.config.IncludeFilterExpr, a.config.ExcludeFilterExpr)
	if err != nil {
		return err
	}

	var filter *mergedFilter.MergedFilter
	if a.config.BatchMode {
		filter = mergedFilter.NewBatchMergedFilter(blockFilter, srcBlocksStore, destBlocksStore, a.config.BatchStartBlock, a.config.BatchStopBlock)
	} else {
		tracker := bstream.NewTracker(250)
		tracker.AddGetter(bstream.BlockStreamHeadTarget, bstream.RetryableBlockRefGetter(20, 10*time.Second, bstream.StreamHeadBlockRefGetter(a.config.BlockstreamAddr)))

		truncationWindow := a.config.TruncationWindow
		if !a.config.TruncationEnabled {
			truncationWindow = 0
		}
		filter = mergedFilter.NewMergedFilter(blockFilter, srcBlocksStore, destBlocksStore, tracker, truncationWindow)
	}

	a.OnTerminating(func(err error) {
		filter.Shutdown(err)
	})

	filter.OnTerminated(func(err error) {
		a.Shutdown(err)
	})

	go filter.Launch()
	return nil

}
