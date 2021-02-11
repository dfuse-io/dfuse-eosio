package tokenmeta

import (
	"context"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
)

func (t *TokenMeta) SetupPipeline(startBlock bstream.BlockRef, blockFilter func(blk *bstream.Block) error, blockstreamAddr string, blocksStore dstore.Store) {
	var preprocessor bstream.PreprocessFunc
	if blockFilter != nil {
		preprocessor = bstream.PreprocessFunc(func(blk *bstream.Block) (interface{}, error) {
			return nil, blockFilter(blk)
		})
	}

	sf := bstream.SourceFromRefFactory(func(startBlockRef bstream.BlockRef, h bstream.Handler) bstream.Source {
		if startBlockRef.ID() == "" {
			startBlockRef = startBlock
		}

		archivedBlockSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			src := bstream.NewFileSource(blocksStore, startBlockRef.Num(), 2, preprocessor, subHandler)
			return src
		})

		zlog.Info("new live joining source", zap.Stringer("start_block", startBlockRef))
		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				context.Background(),
				blockstreamAddr,
				200,
				subHandler,
			)
		})

		options := []bstream.JoiningSourceOption{bstream.JoiningSourceLogger(zlog)}
		if startBlockRef.ID() != "" {
			options = append(options, bstream.JoiningSourceTargetBlockID(startBlockRef.ID()))
		}

		js := bstream.NewJoiningSource(
			archivedBlockSourceFactory,
			liveSourceFactory,
			h,
			options...)
		return js
	})

	forkOptions := []forkable.Option{
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepIrreversible),
	}
	if startBlock.ID() != "" {
		zlog.Info("setting exclusive LIB on forkable", zap.Stringer("start_block", startBlock))
		forkOptions = append(forkOptions, forkable.WithExclusiveLIB(startBlock))
	} else {
		zlog.Warn("wtf startblock empty blah blah", zap.Stringer("start_block", startBlock))
	}
	if t.blockmeta != nil {
		zlog.Info("setting irreversibility checker on forkable")
		forkOptions = append(forkOptions, forkable.WithIrreversibilityChecker(t.blockmeta, 2*time.Second))
	}

	forkableHandler := forkable.New(t, forkOptions...)

	// EternalSource -> (headBlockHandler -> forkableHandler) wrapped in bstream.WithHeadMetrics handler
	headBlockHandler := bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
		t.cache.SetHeadBlockTime(blk.Timestamp)
		return forkableHandler.ProcessBlock(blk, obj)
	})

	t.source = bstream.NewEternalSource(sf, bstream.WithHeadMetrics(headBlockHandler, HeadBlockNum, HeadTimeDrift), bstream.EternalSourceWithLogger(zlog))

	t.OnTerminating(func(e error) {
		t.source.Shutdown(e)
	})
}
