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

package merged_filter

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type MergedFilter struct {
	*shutter.Shutter

	srcStore    dstore.Store
	destStore   dstore.Store
	blockFilter *filtering.BlockFilter

	// live mode
	tracker          *bstream.Tracker
	truncationWindow uint64

	// batch mode
	batchMode  bool
	startBlock uint64
	stopBlock  uint64
}

func NewBatchMergedFilter(blockFilter *filtering.BlockFilter, srcBlockStore, destBlockStore dstore.Store, startBlock, stopBlock uint64) *MergedFilter {
	return &MergedFilter{
		Shutter:     shutter.New(),
		srcStore:    srcBlockStore,
		destStore:   destBlockStore,
		blockFilter: blockFilter,
		batchMode:   true,
		startBlock:  startBlock,
		stopBlock:   stopBlock,
	}
}

func NewMergedFilter(blockFilter *filtering.BlockFilter, srcBlockStore, destBlockStore dstore.Store, tracker *bstream.Tracker, truncationWindow uint64) *MergedFilter {
	roundDownTruncationWindow := truncationWindow - (truncationWindow % 100)

	return &MergedFilter{
		Shutter:          shutter.New(),
		srcStore:         srcBlockStore,
		destStore:        destBlockStore,
		tracker:          tracker,
		blockFilter:      blockFilter,
		truncationWindow: roundDownTruncationWindow,
	}
}

func (f *MergedFilter) Launch() {
	ctx, cancel := context.WithCancel(context.Background())
	f.OnTerminating(func(err error) {
		cancel()
	})

	if f.batchMode {
		f.Shutdown(f.run(ctx, f.startBlock))
		return
	}

	startBlock, err := f.findLiveStartBlock(ctx)
	if err != nil {
		f.Shutdown(err)
		return
	}

	f.Shutdown(f.run(ctx, startBlock))
}

func (f *MergedFilter) findLiveStartBlock(ctx context.Context) (uint64, error) {
	headRef, err := f.tracker.Get(ctx, bstream.BlockStreamHeadTarget)
	if err != nil {
		return 0, err
	}

	headNum := headRef.Num()
	headBase := headNum - (headNum % 100) // Round down to base 100
	for {
		baseFile := fmt.Sprintf("%010d", headBase)

		if f.truncationWindow != 0 {
			lowerWindowBoundary := headNum - f.truncationWindow
			if headBase <= lowerWindowBoundary {
				zlog.Info("reached end of truncation window, no reason to start ealier", zap.String("base_file", baseFile))
				break
			}
		}

		zlog.Info("checking for destination base file existence", zap.String("base_file", baseFile))
		exists, err := f.destStore.FileExists(ctx, baseFile)
		if err != nil {
			return 0, err
		}

		if exists {
			zlog.Info("destination base file exists, starting just after this", zap.String("base_file", baseFile))
			headBase += 100
			break
		}

		headBase -= 100
	}

	return headBase, nil
}

func (f *MergedFilter) run(ctx context.Context, startBase uint64) error {
	currentBase := startBase

	logRateLimiter := rate.NewLimiter(2, 2)
	for {
		currentBaseFile := fmt.Sprintf("%010d", currentBase)

		destExists, err := f.destStore.FileExists(ctx, currentBaseFile)
		if err != nil {
			return err
		}
		if !destExists {
			srcExists, err := f.srcStore.FileExists(ctx, currentBaseFile)
			if err != nil {
				return err
			}
			if !srcExists {
				if logRateLimiter.Allow() {
					zlog.Info("waiting for base file", zap.String("base_file", currentBaseFile))
				}
				time.Sleep(5 * time.Second)
				continue
			}
			if logRateLimiter.Allow() {
				zlog.Info("processing base file", zap.String("base_file", currentBaseFile))
			}
			err = derr.Retry(5, func(ctx context.Context) error {
				return f.process(ctx, currentBaseFile)
			})
			if err != nil {
				return err
			}
		} else {
			if logRateLimiter.Allow() {
				zlog.Info("file already exists at destination", zap.String("base_file", currentBaseFile))
			}
		}

		currentBase += 100
		if f.batchMode && currentBase >= f.stopBlock {
			zlog.Info("stopping: reached stop block and batchMode is set.")
			break
		}
	}
	return nil

}

func (f *MergedFilter) process(ctx context.Context, currentBaseFile string) error {
	var count int
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	readCloser, err := f.srcStore.OpenObject(ctx, currentBaseFile)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	blkReader, err := codec.NewBlockReader(readCloser)
	if err != nil {
		return err
	}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return err
	}
	defer writePipe.Close()

	writeObjectDone := make(chan error, 1)
	go func() {
		writeObjectDone <- f.destStore.WriteObject(ctx, currentBaseFile, readPipe)
	}()

	blkWriter, err := codec.NewBlockWriter(writePipe)
	if err != nil {
		return err
	}

	for {
		blk, err := blkReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		count++

		if err = f.blockFilter.TransformInPlace(blk); err != nil {
			return err
		}

		if err = blkWriter.Write(blk); err != nil {
			return err
		}
	}

	err = writePipe.Close()
	if err != nil {
		return err
	}

	err = <-writeObjectDone
	if err != nil {
		return err
	}

	zlog.Info("uploaded filtered file", zap.String("base_file", currentBaseFile), zap.Int("block_count", count))
	return nil
}
