// Copyright 2020 dfuse Platform Inc.
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

package fluxdb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/metrics"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
)

var ErrCleanSourceStop = errors.New("clean source stop")

func BuildReprocessingPipeline(
	handler bstream.Handler,
	blocksStore dstore.Store,
	startBlockNum uint64,
	numBlocksBeforeStart uint64,
	parallelDownloadCount int,
) bstream.Source {
	gate := bstream.NewBlockNumGate(startBlockNum, bstream.GateInclusive, handler, bstream.GateOptionWithLogger(zlog))
	gate.MaxHoldOff = 1000

	forkableSource := forkable.New(gate, forkable.WithLogger(zlog), forkable.WithFilters(forkable.StepIrreversible))

	getBlocksFrom := startBlockNum
	if getBlocksFrom > numBlocksBeforeStart {
		getBlocksFrom = startBlockNum - numBlocksBeforeStart // Make sure you cover that irreversible block
	}

	source := bstream.NewFileSource(
		blocksStore,
		getBlocksFrom,
		parallelDownloadCount,
		PreprocessBlock,
		forkableSource,
		bstream.FileSourceWithLogger(zlog),
	)
	return source
}

func (fdb *FluxDB) BuildPipeline(getBlockID bstream.EternalSourceStartBackAtBlock, handler bstream.Handler, blocksStore dstore.Store, publisherAddr string, parallelDownloadCount int) {
	sf := bstream.SourceFromRefFactory(func(startBlock bstream.BlockRef, h bstream.Handler) bstream.Source {

		// Exclusive, we never want to process the same block
		// twice. When doing reprocessing, we'll need to provide the block
		// just before.
		gate := bstream.NewBlockIDGate(startBlock.ID(), bstream.GateExclusive, h, bstream.GateOptionWithLogger(zlog))

		forkableOptions := []forkable.Option{forkable.WithLogger(zlog), forkable.WithFilters(forkable.StepNew | forkable.StepIrreversible)}
		if startBlock != EmptyBlockRef {
			// Only when we do **not** start from the beginning (i.e. startBlock is the empty block ref), that the
			// forkable should be initialized with an initial LIB value. Otherwise, when we start fresh, the forkable
			// will automatically set its LIB to the first streamable block of the chain.
			forkableOptions = append(forkableOptions, forkable.WithExclusiveLIB(startBlock))
		}

		forkHandler := forkable.New(gate, forkableOptions...)

		liveSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				context.Background(),
				publisherAddr,
				250,
				bstream.NewPreprocessor(PreprocessBlock, subHandler),
				blockstream.WithRequester("fluxdb"),
			)
		})

		fileSourceFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			fs := bstream.NewFileSource(
				blocksStore,
				startBlock.Num(),
				parallelDownloadCount,
				PreprocessBlock,
				subHandler,
			)

			return fs
		})

		return bstream.NewJoiningSource(fileSourceFactory, liveSourceFactory, forkHandler,
			bstream.JoiningSourceLogger(zlog),
			bstream.JoiningSourceTargetBlockID(startBlock.ID()),
			bstream.JoiningSourceTargetBlockNum(2),
		)
	})

	es := bstream.NewDelegatingEternalSource(sf, getBlockID, handler, bstream.EternalSourceWithLogger(zlog))

	fdb.source = es
}

// FluxDBHandler is a pipeline that writes in FluxDB
type FluxDBHandler struct {
	db  *FluxDB
	ctx context.Context

	writeEnabled                bool
	writeOnEachIrreversibleStep bool
	serverForkDB                *forkable.ForkDB

	speculativeReadsLock sync.RWMutex
	speculativeWrites    []*WriteRequest
	headBlock            bstream.BlockRef

	batchWrites       []*WriteRequest
	batchOpen         time.Time
	batchClose        time.Time
	batchWritableRows int

	lastBlockIDCheck time.Time
}

func NewHandler(db *FluxDB) *FluxDBHandler {
	return &FluxDBHandler{
		db:  db,
		ctx: context.Background(),
	}
}

func (p *FluxDBHandler) EnableWrites() {
	p.writeEnabled = true
}

func (p *FluxDBHandler) EnableWriteOnEachIrreversibleStep() {
	p.writeOnEachIrreversibleStep = true
}

func (p *FluxDBHandler) InitializeStartBlockID() (startBlock bstream.BlockRef, err error) {
	startBlock, err = p.db.FetchLastWrittenBlock(p.ctx)
	if err != nil {
		return nil, err
	}

	zlog.Info("initializing pipeline forkdb", zap.Stringer("block", startBlock))
	p.serverForkDB = forkable.NewForkDB(forkable.ForkDBWithLogger(zlog))
	p.serverForkDB.InitLIB(startBlock)

	return startBlock, nil
}

func (p *FluxDBHandler) HeadBlock(ctx context.Context) bstream.BlockRef {
	p.speculativeReadsLock.RLock()
	defer p.speculativeReadsLock.RUnlock()

	return p.headBlock
}

func (p *FluxDBHandler) FetchSpeculativeWrites(ctx context.Context, headBlockID string, upToBlockNum uint32) (speculativeWrites []*WriteRequest) {
	p.speculativeReadsLock.RLock()
	defer p.speculativeReadsLock.RUnlock()

	for _, write := range p.speculativeWrites {
		if write.BlockNum > upToBlockNum {
			continue
		}
		speculativeWrites = append(speculativeWrites, write)
	}

	return
}

func (p *FluxDBHandler) updateSpeculativeWrites(newHeadBlock bstream.BlockRef) {
	blocks := p.serverForkDB.ReversibleSegment(newHeadBlock)
	if len(blocks) == 0 {
		return
	}

	var newWrites []*WriteRequest
	for _, blk := range blocks {
		req := blk.Object.(*WriteRequest)
		newWrites = append(newWrites, req)
	}

	p.speculativeReadsLock.RLock()
	defer p.speculativeReadsLock.RUnlock()

	p.speculativeWrites = newWrites
	p.headBlock = newHeadBlock
}

func (p *FluxDBHandler) ProcessBlock(rawBlk *bstream.Block, rawObj interface{}) error {
	blk := rawBlk.ToNative().(*pbcodec.Block)
	blkRef := bstream.BlockRefFromID(rawBlk.ID())
	if rawBlk.Num()%120 == 0 {
		zlog.Info("processing block (1/120)", zap.Stringer("block", rawBlk))
	}

	// TODO: implement based on a Forkable object.. will be quite simpler
	fObj := rawObj.(*forkable.ForkableObject)

	switch fObj.Step {
	case forkable.StepNew:
		metrics.HeadTimeDrift.SetBlockTime(blk.MustTime())
		metrics.HeadBlockNumber.SetUint64(blk.Num())
		if !p.db.IsReady() {
			if isNearRealtime(blk, time.Now()) && p.HeadBlock(context.Background()) != nil {
				zlog.Info("realtime blocks flowing, marking process as ready")
				p.db.SetReady()
			}
		}

		p.serverForkDB.AddLink(
			blkRef,
			bstream.BlockRefFromID(rawBlk.PreviousID()),
			fObj.Obj.(*WriteRequest),
		)

		p.updateSpeculativeWrites(rawBlk)

	case forkable.StepIrreversible:
		if fObj.StepCount-1 != fObj.StepIndex { // last irreversible block in multi-block step
			return nil
		}

		now := time.Now()
		if p.writeEnabled {
			if len(p.batchWrites) == 0 {
				p.batchOpen = now
				p.batchClose = now.Add(1 * time.Second) // Always flush at least the previous LIB
			}

			zlog.Debug("accumulating write request from irreversible blocks", zap.Stringer("block", rawBlk), zap.Int("block_count", len(fObj.StepBlocks)))
			for _, newIrrBlk := range fObj.StepBlocks {
				req := newIrrBlk.Obj.(*WriteRequest)

				p.batchWrites = append(p.batchWrites, req)
				p.batchWritableRows += len(req.TabletRows)
			}

			zlog.Debug("write request stats irreversible blocks", zap.Stringer("block", rawBlk), zap.Int("writable_rows", p.batchWritableRows), zap.Time("batch_close_at", p.batchClose))
			if p.batchWritableRows > 5000 || now.After(p.batchClose) || p.writeOnEachIrreversibleStep {
				defer func() {
					p.batchWrites = nil
					p.batchWritableRows = 0
					// p.abisWritten = 0
				}()

				err := p.db.WriteBatch(p.ctx, p.batchWrites)
				if err != nil {
					return err
				}

				timePerBlock := time.Now().Sub(p.batchOpen) / time.Duration(len(p.batchWrites))
				zlog.Info("wrote irreversible segment of blocks starting here",
					zap.String("block_id", rawBlk.ID()),
					zap.Uint64("block_num", rawBlk.Num()),
					zap.Duration("batch_elapsed", time.Now().Sub(p.batchOpen)),
					zap.Duration("batch_elapsed_per_block", timePerBlock),
					zap.Int("batch_write_count", len(p.batchWrites)),
					zap.Int("batch_writable_row_count", p.batchWritableRows),
				)
			}

			p.serverForkDB.MoveLIB(blkRef)
		} else {
			// Fetch from database, and sync with the writer before truncating the LIB here.
			// Don't ask more than once each 2 seconds..
			if p.lastBlockIDCheck.Before(time.Now().Add(-2 * time.Second)) {
				lastWrittenBlock, err := p.db.FetchLastWrittenBlock(p.ctx)
				if err != nil {
					return err
				}

				if lastWrittenBlock.ID() != p.serverForkDB.LIBID() {
					zlog.Info("writer's LIB updated, advancing server forkDB in return",
						zap.String("block_id", lastWrittenBlock.ID()),
						zap.Uint64("block_num", lastWrittenBlock.Num()),
					)

					p.serverForkDB.MoveLIB(lastWrittenBlock)
				}

				p.lastBlockIDCheck = time.Now()
			}
		}

	default:
		panic(fmt.Errorf("unsupported forkable step %q", fObj.Step))
	}

	return nil
}

func isNearRealtime(blk *pbcodec.Block, now time.Time) bool {
	tm, _ := ptypes.Timestamp(blk.Header.Timestamp)
	return now.Add(-15 * time.Second).Before(tm)
}
