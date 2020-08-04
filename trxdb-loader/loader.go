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

package trxdb_loader

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dfuse-eosio/trxdb-loader/metrics"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/shutter"
	eosgo "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type Job = func(blockNum uint64, blk *pbcodec.Block, fObj *forkable.ForkableObject) (err error)

const MaxRetries = 5
const BackoffBaseTime = 10 * time.Minute

type TrxDBLoader struct {
	*shutter.Shutter
	processingJob             Job
	db                        trxdb.DBWriter
	batchSize                 uint64
	lastTickBlock             uint64
	lastTickTime              time.Time
	lastBlockSecs             []float64
	lastBlockSecsPointer      int
	blocksStore               dstore.Store
	blockFilter               func(blk *bstream.Block) error
	blockStreamAddr           string
	source                    bstream.Source
	endBlock                  uint64
	parallelFileDownloadCount int
	healthy                   bool
	retryCnt                  int
	truncationWindow          uint64

	forkDB *forkable.ForkDB
}

func NewTrxDBLoader(
	blockStreamAddr string,
	blocksStore dstore.Store,
	batchSize uint64,
	db trxdb.DBWriter,
	parallelFileDownloadCount int,
	blockFilter func(blk *bstream.Block) error,
	truncationWindow uint64,
) *TrxDBLoader {

	loader := &TrxDBLoader{
		blockStreamAddr:           blockStreamAddr,
		blocksStore:               blocksStore,
		Shutter:                   shutter.New(),
		db:                        db,
		batchSize:                 batchSize,
		forkDB:                    forkable.NewForkDB(forkable.ForkDBWithLogger(zlog)),
		parallelFileDownloadCount: parallelFileDownloadCount,
		retryCnt:                  1,
		blockFilter:               blockFilter,
		truncationWindow:          truncationWindow,
		lastBlockSecs:             make([]float64, 100),
		lastBlockSecsPointer:      0,
	}

	// By default, everything is assumed to be the full job, pipeline building overrides that
	loader.processingJob = loader.FullJob

	if d, ok := db.(trxdb.Debugeable); ok {
		d.Dump()
	}

	return loader
}

func (l *TrxDBLoader) BuildPipelineLive(allowLiveOnEmptyTable bool) error {
	l.processingJob = l.FullJob

	tracker := bstream.NewTracker(200)
	tracker.AddGetter(bstream.BlockStreamHeadTarget, bstream.RetryableBlockRefGetter(30, 10*time.Second, bstream.StreamHeadBlockRefGetter(l.blockStreamAddr)))

	startAtBlockX := false
	var blockX uint64

	startLIB, err := l.db.GetLastWrittenIrreversibleBlockRef(context.Background())
	if err != nil {
		if err == kvdb.ErrNotFound && allowLiveOnEmptyTable {
			startAtBlockX = true
			blockX, err = tracker.GetRelativeBlock(context.Background(), -(int64(l.truncationWindow)), bstream.BlockStreamHeadTarget)
			if err != nil {
				return fmt.Errorf("get relative block: %w", err)
			}

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
		if startAtBlockX {
			// We explicity want to start back from beginning, hence no gate at all
			zlog.Info("forcing block start block 1")

			handler = h
			blockNum = blockX
		} else {
			// We start back from last written LIB, use a gate to start processing at the right place
			if startBlockRef.ID() == "" {
				startBlockID = startLIB.ID()
			} else {
				startBlockID = startBlockRef.ID()
			}

			handler = bstream.NewBlockIDGate(startBlockID, bstream.GateExclusive, h, bstream.GateOptionWithLogger(zlog))
			blockNum = uint64(eosgo.BlockNum(startBlockID))
		}

		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			src := blockstream.NewSource(
				context.Background(),
				l.blockStreamAddr,
				300,
				subHandler,
				blockstream.WithRequester("trxdb-loader"),
			)
			return src
		})

		var filterPreprocessFunc bstream.PreprocessFunc
		if l.blockFilter != nil {
			filterPreprocessFunc = func(blk *bstream.Block) (interface{}, error) {
				return nil, l.blockFilter(blk)
			}
		}

		fileSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			fs := bstream.NewFileSource(
				l.blocksStore,
				blockNum,
				l.parallelFileDownloadCount,
				filterPreprocessFunc,
				subHandler,
			)
			return fs
		})

		return bstream.NewJoiningSource(fileSourceFactory,
			liveSourceFactory,
			handler,
			bstream.JoiningSourceLogger(zlog),
			bstream.JoiningSourceTargetBlockID(startBlockRef.ID()),
			bstream.JoiningSourceTargetBlockNum(bstream.GetProtocolFirstStreamableBlock),
			bstream.JoiningSourceLiveTracker(300, bstream.StreamHeadBlockRefGetter(l.blockStreamAddr)),
		)
	})

	forkableHandler := forkable.New(l,
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepNew|forkable.StepIrreversible),
		forkable.EnsureAllBlocksTriggerLongestChain(),
	)

	es := bstream.NewEternalSource(sf, forkableHandler, bstream.EternalSourceWithLogger(zlog))
	l.source = es
	return nil
}

func (l *TrxDBLoader) BuildPipelineBatch(startBlockNum uint64, numBlocksBeforeStart uint64) {
	l.BuildPipelineJob(startBlockNum, numBlocksBeforeStart, l.FullJob)
}

func (l *TrxDBLoader) BuildPipelinePatch(startBlockNum uint64, numBlocksBeforeStart uint64) {
	l.BuildPipelineJob(startBlockNum, numBlocksBeforeStart, l.PatchJob)
}

func (l *TrxDBLoader) BuildPipelineJob(startBlockNum uint64, numBlocksBeforeStart uint64, job Job) {
	l.processingJob = job

	gate := bstream.NewBlockNumGate(startBlockNum, bstream.GateInclusive, l, bstream.GateOptionWithLogger(zlog))
	gate.MaxHoldOff = 1000

	forkableHandler := forkable.New(gate,
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepNew|forkable.StepIrreversible),
	)

	getBlocksFrom := startBlockNum
	if getBlocksFrom > numBlocksBeforeStart {
		getBlocksFrom = startBlockNum - numBlocksBeforeStart // Make sure you cover that irreversible block
	}

	var filterPreprocessFunc bstream.PreprocessFunc
	if l.blockFilter != nil {
		filterPreprocessFunc = func(blk *bstream.Block) (interface{}, error) {
			return nil, l.blockFilter(blk)
		}
	}

	fs := bstream.NewFileSource(
		l.blocksStore,
		getBlocksFrom,
		l.parallelFileDownloadCount,
		filterPreprocessFunc,
		forkableHandler,
		bstream.FileSourceWithLogger(zlog),
	)
	l.source = fs
}

func (l *TrxDBLoader) Launch() {
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

func (l *TrxDBLoader) InitLIB(libID string) {
	// Only works on EOS!
	l.forkDB.InitLIB(bstream.NewBlockRefFromID(libID))
}

// StopBeforeBlock indicates the stop block (exclusive), means that
// block num will not be inserted.
func (l *TrxDBLoader) StopBeforeBlock(blockNum uint64) {
	l.endBlock = blockNum
}

func (l *TrxDBLoader) setUnhealthy() {
	if l.healthy {
		l.healthy = false
	}
}

func (l *TrxDBLoader) setHealthy() {
	if !l.healthy {
		l.healthy = true
	}
}

func (l *TrxDBLoader) Healthy() bool {
	return l.healthy
}

// fullJob does all the database insertions needed to load the blockchain
// into our database.
func (l *TrxDBLoader) FullJob(blockNum uint64, block *pbcodec.Block, fObj *forkable.ForkableObject) (err error) {
	blkTime := block.MustTime()

	if traceEnabled {
		zlog.Debug("full job received a block to process",
			zap.Stringer("step", fObj.Step),
			zap.Stringer("block", block.AsRef()),
		)
	}

	switch fObj.Step {
	case forkable.StepNew:
		l.ShowProgress(blockNum)
		l.setHealthy()

		defer metrics.HeadBlockTimeDrift.SetBlockTime(blkTime)
		defer metrics.HeadBlockNumber.SetUint64(blockNum)

		// this could have a db write
		if err := l.db.PutBlock(context.Background(), block); err != nil {
			return fmt.Errorf("store block: %s", err)
		}

		return l.FlushIfNeeded(blockNum, blkTime)
	case forkable.StepIrreversible:

		if l.endBlock != 0 && blockNum >= l.endBlock && fObj.StepCount == fObj.StepIndex+1 {
			err := l.DoFlush(blockNum, "reached end block")
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

		// this could have a db write
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

func (l *TrxDBLoader) ProcessBlock(blk *bstream.Block, obj interface{}) (err error) {
	if l.IsTerminating() {
		return nil
	}

	return l.processingJob(blk.Num(), blk.ToNative().(*pbcodec.Block), obj.(*forkable.ForkableObject))
}

func (l *TrxDBLoader) DoFlush(blockNum uint64, reason string) error {
	zlog.Debug("flushing block", zap.Uint64("block_num", blockNum), zap.String("reason", reason))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	err := l.db.Flush(ctx)
	if err != nil {
		for ok := true; ok; ok = l.retryCnt <= MaxRetries && err != nil {
			zlog.Error("db flush failed", zap.Error(err))
			retryBackoff := time.Duration(l.retryCnt) * BackoffBaseTime
			zlog.Info("retrying flush", zap.Float64("backoff_time", retryBackoff.Seconds()))

			time.Sleep(retryBackoff)
			ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			err = l.db.Flush(ctx)
			l.retryCnt++
		}

		if err != nil {
			return fmt.Errorf("db flush failed after reaching max retries (%d): %w", MaxRetries, err)
		}
	} else {
		l.retryCnt = 1
	}
	return nil
}

func (l *TrxDBLoader) FlushIfNeeded(blockNum uint64, blockTime time.Time) error {
	batchSizeReached := blockNum%l.batchSize == 0
	closeToHeadBlockTime := time.Since(blockTime) < 25*time.Second

	if batchSizeReached || closeToHeadBlockTime {
		reason := "needed"
		if batchSizeReached {
			reason += ", batch size reached"
		}

		if closeToHeadBlockTime {
			reason += ", close to head block"
		}

		err := l.DoFlush(blockNum, reason)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(blockNum)
	}
	return nil
}

func (l *TrxDBLoader) ShowProgress(blockNum uint64) {
	now := time.Now()
	if l.lastTickTime.Before(now.Add(-5 * time.Second)) {
		if !l.lastTickTime.IsZero() {

			blockSec := float64(blockNum-l.lastTickBlock) / float64(now.Sub(l.lastTickTime)/time.Second)

			var totalBlockSec float64 = 0
			var avgBlockSec float64 = 0

			l.lastBlockSecs[l.lastBlockSecsPointer%len(l.lastBlockSecs)] = blockSec
			l.lastBlockSecsPointer++

			for _, curBlockSec := range l.lastBlockSecs {
				totalBlockSec += curBlockSec
			}

			if l.lastBlockSecsPointer < len(l.lastBlockSecs) {
				avgBlockSec = totalBlockSec / float64(l.lastBlockSecsPointer)
			} else {
				avgBlockSec = totalBlockSec / float64(len(l.lastBlockSecs))
			}

			zlog.Info("5sec AVG INSERT RATE",
				zap.Uint64("block_num", blockNum),
				zap.Uint64("last_tick_block", l.lastTickBlock),
				zap.Float64("block_sec", math.Round(blockSec*100)/100),
				zap.Float64("100_tick_avg", math.Round(avgBlockSec*100)/100),
			)
		}
		l.lastTickTime = now
		l.lastTickBlock = blockNum
	}
}

func (l *TrxDBLoader) ShouldPushLIBUpdates(dposLIBNum uint64) bool {
	if dposLIBNum > l.forkDB.LIBNum() {
		return true
	}
	return false
}

func (l *TrxDBLoader) UpdateIrreversibleData(nowIrreversibleBlocks []*bstream.PreprocessedBlock) error {
	for _, blkObj := range nowIrreversibleBlocks {
		blk := blkObj.Block.ToNative().(*pbcodec.Block)

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
func (l *TrxDBLoader) PatchJob(blockNum uint64, blk *pbcodec.Block, fObj *forkable.ForkableObject) (err error) {
	switch fObj.Step {
	case forkable.StepNew:
		l.ShowProgress(blockNum)
		return l.FlushIfNeeded(blockNum, blk.MustTime())

	case forkable.StepIrreversible:
		if l.endBlock != 0 && blockNum >= l.endBlock && fObj.StepCount == fObj.StepIndex+1 {
			err := l.DoFlush(blockNum, "batch end block reached")
			if err != nil {
				return err
			}

			l.Shutdown(nil)
			return nil
		}
	}

	return nil
}
