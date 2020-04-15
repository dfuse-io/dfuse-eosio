// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kvdb_loader

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/kvdb-loader/metrics"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/shutter"
	eosgo "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type Job = func(blockNum uint64, blk *pbeos.Block, fObj *forkable.ForkableObject) (err error)

type BigtableLoader struct {
	*shutter.Shutter
	processingJob             Job
	db                        eosdb.DBWriter
	batchSize                 uint64
	lastTickBlock             uint64
	lastTickTime              time.Time
	blocksStore               dstore.Store
	blockStreamAddr           string
	source                    bstream.Source
	endBlock                  uint64
	parallelFileDownloadCount int
	healthy                   bool

	forkDB *forkable.ForkDB
}

func NewBigtableLoader(
	blockStreamAddr string,
	blocksStore dstore.Store,
	batchSize uint64,
	db eosdb.DBWriter,
	parallelFileDownloadCount int,
) *BigtableLoader {
	loader := &BigtableLoader{
		blockStreamAddr:           blockStreamAddr,
		blocksStore:               blocksStore,
		Shutter:                   shutter.New(),
		db:                        db,
		batchSize:                 batchSize,
		forkDB:                    forkable.NewForkDB(),
		parallelFileDownloadCount: parallelFileDownloadCount,
	}

	// By default, everything is assumed to be the full job, pipeline building overrides that
	loader.processingJob = loader.FullJob

	return loader
}

func (l *BigtableLoader) BuildPipelineLive(allowLiveOnEmptyTable bool) error {
	l.processingJob = l.FullJob

	startAtBlockOne := false
	startLIB, err := l.db.GetLastWrittenIrreversibleBlockRef(context.Background())
	if err != nil {
		if err == kvdb.ErrNotFound && allowLiveOnEmptyTable {
			zlog.Info("forcing block start block 1")
			startAtBlockOne = true
		} else {
			return fmt.Errorf("failed getting latest written LIB: %w", err)
		}
	}

	if startLIB != nil {
		zlog.Info("initializing LIB", zap.Stringer("lib", startLIB))
		l.InitLIB(startLIB.ID())
	}

	sf := bstream.SourceFromRefFactory(func(startBlockRef bstream.BlockRef, h bstream.Handler) bstream.Source {
		var handler bstream.Handler
		var blockNum uint64
		var startBlockID string
		if startAtBlockOne {
			// We explicity want to start back from beginning, hence no gate at all
			handler = h
			blockNum = uint64(1)
		} else {
			// We start back from last written LIB, use a gate to start processing at the right place
			if startBlockRef.ID() == "" {
				startBlockID = startLIB.ID()
			} else {
				startBlockID = startBlockRef.ID()
			}

			handler = bstream.NewBlockIDGate(startBlockID, bstream.GateExclusive, h)
			blockNum = uint64(eosgo.BlockNum(startBlockID))
		}

		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			src := blockstream.NewSource(
				context.Background(),
				l.blockStreamAddr,
				300,
				subHandler,
			)
			src.SetName("kvdb-loader")
			return src
		})
		fileSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			fs := bstream.NewFileSource(
				l.blocksStore,
				blockNum,
				l.parallelFileDownloadCount,
				nil,
				subHandler,
			)
			return fs
		})

		js := bstream.NewJoiningSource(fileSourceFactory,
			liveSourceFactory,
			handler,
			bstream.JoiningSourceTargetBlockID(startBlockRef.ID()),
			bstream.JoiningSourceTargetBlockNum(2),
			bstream.JoiningSourceName("kvdb-loader"),
		)
		js.SetName("kvdb_loader")
		return js
	})

	forkableHandler := forkable.New(l,
		forkable.WithFilters(forkable.StepNew|forkable.StepIrreversible),
		forkable.EnsureAllBlocksTriggerLongestChain(),
		forkable.WithName("kvdb-loader"),
	)

	es := bstream.NewEternalSource(sf, forkableHandler)
	l.source = es
	return nil
}

func (l *BigtableLoader) BuildPipelineBatch(startBlockNum uint64, numBlocksBeforeStart uint64) {
	l.BuildPipelineJob(startBlockNum, numBlocksBeforeStart, l.FullJob)
}

func (l *BigtableLoader) BuildPipelinePatch(startBlockNum uint64, numBlocksBeforeStart uint64) {
	l.BuildPipelineJob(startBlockNum, numBlocksBeforeStart, l.PatchJob)
}

func (l *BigtableLoader) BuildPipelineJob(startBlockNum uint64, numBlocksBeforeStart uint64, job Job) {
	l.processingJob = job

	gate := bstream.NewBlockNumGate(startBlockNum, bstream.GateInclusive, l)
	gate.MaxHoldOff = 1000

	forkableHandler := forkable.New(gate,
		forkable.WithFilters(forkable.StepNew|forkable.StepIrreversible),
	)

	getBlocksFrom := startBlockNum
	if getBlocksFrom > numBlocksBeforeStart {
		getBlocksFrom = startBlockNum - numBlocksBeforeStart // Make sure you cover that irreversible block
	}

	fs := bstream.NewFileSource(
		l.blocksStore,
		getBlocksFrom,
		l.parallelFileDownloadCount,
		nil,
		forkableHandler,
	)
	l.source = fs
}

func (l *BigtableLoader) Launch() {
	l.source.OnTerminating(func(err error) {
		l.Shutdown(err)
	})
	l.source.OnTerminated(func(err error) {
		l.setUnhealthy()
	})
	l.OnTerminating(func(err error) {
		l.source.Shutdown(err)
	})
	l.source.Run()
}

func (l *BigtableLoader) InitLIB(libID string) {
	// Only works on EOS!
	l.forkDB.InitLIB(bstream.BlockRefFromID(libID))
}

// StopBeforeBlock indicates the stop block (exclusive), means that
// block num will not be inserted.
func (l *BigtableLoader) StopBeforeBlock(blockNum uint64) {
	l.endBlock = blockNum
}

func (l *BigtableLoader) setUnhealthy() {
	if l.healthy {
		l.healthy = false
	}
}

func (l *BigtableLoader) setHealthy() {
	if !l.healthy {
		l.healthy = true
	}
}

func (l *BigtableLoader) Healthy() bool {
	return l.healthy
}

// fullJob does all the database insertions needed to load the blockchain
// into our database.
func (l *BigtableLoader) FullJob(blockNum uint64, block *pbeos.Block, fObj *forkable.ForkableObject) (err error) {
	blkTime := block.MustTime()

	switch fObj.Step {
	case forkable.StepNew:
		l.ShowProgress(blockNum)
		l.setHealthy()

		defer metrics.HeadBlockTimeDrift.SetBlockTime(blkTime)
		defer metrics.HeadBlockNumber.SetUint64(blockNum)

		if err := l.db.PutBlock(context.Background(), block); err != nil {
			return fmt.Errorf("store block: %s", err)
		}
		return l.FlushIfNeeded(blockNum, blkTime)
	case forkable.StepIrreversible:
		if l.endBlock != 0 && blockNum >= l.endBlock && fObj.StepCount == fObj.StepIndex+1 {
			err := l.DoFlush(blockNum)
			if err != nil {
				l.Shutdown(err)
				return err
			}
			l.Shutdown(nil)
			return nil
		}

		// Handle only the first multi-block step Irreversible
		if fObj.StepIndex != 0 {
			return nil
		}

		if err := l.UpdateIrreversibleData(fObj.StepBlocks); err != nil {
			return err
		}

		err = l.FlushIfNeeded(blockNum, blkTime)
		if err != nil {
			zlog.Error("flushIfNeeded", zap.Error(err))
			return err
		}

		return nil

	default:
		return fmt.Errorf("unsupported forkable step %q", fObj.Step)
	}
}

func (l *BigtableLoader) ProcessBlock(blk *bstream.Block, obj interface{}) (err error) {
	if l.IsTerminating() {
		return nil
	}

	return l.processingJob(blk.Num(), blk.ToNative().(*pbeos.Block), obj.(*forkable.ForkableObject))
}

func (l *BigtableLoader) DoFlush(blockNum uint64) error {
	zlog.Debug("flushing block", zap.Uint64("block_num", blockNum))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	err := l.db.Flush(ctx)
	if err != nil {
		return fmt.Errorf("Leaving ProcessBlock on failed flushAllMutations: %s", err)
	}
	return nil
}

func (l *BigtableLoader) FlushIfNeeded(blockNum uint64, blockTime time.Time) error {
	if blockNum%l.batchSize == 0 || time.Since(blockTime) < 25*time.Second {
		err := l.DoFlush(blockNum)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(blockNum)
	}
	return nil
}

func (l *BigtableLoader) ShowProgress(blockNum uint64) {
	now := time.Now()
	if l.lastTickTime.Before(now.Add(-5 * time.Second)) {
		if !l.lastTickTime.IsZero() {
			zlog.Info("5sec AVG INSERT RATE",
				zap.Uint64("block_num", blockNum),
				zap.Uint64("last_tick_block", l.lastTickBlock),
				zap.Float64("block_sec", float64(blockNum-l.lastTickBlock)/float64(now.Sub(l.lastTickTime)/time.Second)),
			)
		}
		l.lastTickTime = now
		l.lastTickBlock = blockNum
	}
}

func (l *BigtableLoader) ShouldPushLIBUpdates(dposLIBNum uint64) bool {
	if dposLIBNum > l.forkDB.LIBNum() {
		return true
	}
	return false
}

func (l *BigtableLoader) UpdateIrreversibleData(nowIrreversibleBlocks []*bstream.PreprocessedBlock) error {
	for _, blkObj := range nowIrreversibleBlocks {
		blk := blkObj.Block.ToNative().(*pbeos.Block)

		if blk.Num() == 1 {
			// Empty block 0, so we don't care
			continue
		}

		if err := l.db.UpdateNowIrreversibleBlock(context.Background(), blk); err != nil {
			return err
		}
	}

	return nil
}

// patchDatabase is a "scratch" pad to define patch code that can be applied
// on an ad-hoc basis. The idea is to leave this function empty when no patch needs
// to be applied.
//
// When a patch is required, the suggested workflow is to develop the patch code in
// a side branch. When the code is ready, the "production" commit is tagged with the
// `patch-<tag>-<date>` where the tag is giving an overview of the patch and the date
// is the effective date (`<year>-<month>-<day>`): `patch-add-trx-meta-written-2019-06-30`.
// The branch is then deleted and the tag is pushed to the remote repository.
func (l *BigtableLoader) PatchJob(blockNum uint64, blk *pbeos.Block, fObj *forkable.ForkableObject) (err error) {
	switch fObj.Step {
	case forkable.StepNew:
		l.ShowProgress(blockNum)
		return l.FlushIfNeeded(blockNum, blk.MustTime())

	case forkable.StepIrreversible:
		if l.endBlock != 0 && blockNum >= l.endBlock && fObj.StepCount == fObj.StepIndex+1 {
			err := l.DoFlush(blockNum)
			if err != nil {
				return err
			}

			l.Shutdown(nil)
			return nil
		}
	}

	return nil
}
