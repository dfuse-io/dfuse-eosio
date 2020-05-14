package sqlsync

import (
	"context"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
)

func (s *SQLSync) Launch() error {
	zlog.Info("launching pipeline")
	go s.source.Run()

	<-s.source.Terminated()
	if err := s.source.Err(); err != nil {
		zlog.Error("source shutdown with error", zap.Error(err))
		return err
	}
	zlog.Info("source is done")

	return nil
}

func (t *SQLSync) SetupPipeline(startBlock bstream.BlockRef, blockstreamAddr string, blocksStore dstore.Store) {

	sf := bstream.SourceFromRefFactory(func(startBlockRef bstream.BlockRef, h bstream.Handler) bstream.Source {

		if startBlockRef.ID() == "" {
			startBlockRef = startBlock
		}

		archivedBlockSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			src := bstream.NewFileSource(blocksStore, startBlockRef.Num(), 1, nil, subHandler)
			return src
		})

		zlog.Info("new live joining source", zap.Uint64("start_block_num", startBlockRef.Num()))
		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				context.Background(),
				blockstreamAddr,
				200,
				subHandler,
			)
		})

		options := []bstream.JoiningSourceOption{}
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
		forkable.WithFilters(forkable.StepIrreversible),
	}
	if startBlock.ID() != "" {
		forkOptions = append(forkOptions, forkable.WithExclusiveLIB(startBlock))
	}
	forkableHandler := forkable.New(t, forkOptions...)

	t.source = bstream.NewEternalSource(sf, forkableHandler)

	t.OnTerminating(func(e error) {
		t.source.Shutdown(e)
	})
}
