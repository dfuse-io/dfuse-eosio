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
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type MergedFilter struct {
	*shutter.Shutter

	srcStore         dstore.Store
	destStore        dstore.Store
	blockFilter      *filtering.BlockFilter
	tracker          *bstream.Tracker
	truncationWindow uint64
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

	headRef, err := f.tracker.Get(ctx, bstream.BlockStreamHeadTarget)
	if err != nil {
		f.Shutdown(err)
		return
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
			f.Shutdown(err)
			return
		}

		if exists {
			zlog.Info("destination base file exists, starting just after this", zap.String("base_file", baseFile))
			headBase += 100
			break
		}

		headBase -= 100
	}

	f.Shutdown(f.process(ctx, headBase))
}

func (f *MergedFilter) process(ctx context.Context, startBase uint64) error {
	currentBase := startBase
	var lastPrinted string
	for {
		currentBaseFile := fmt.Sprintf("%010d", currentBase)

		if lastPrinted != currentBaseFile {
			zlog.Info("processing base file", zap.String("base_file", currentBaseFile))
		}
		lastPrinted = currentBaseFile

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
				zlog.Info("waiting for base file", zap.String("base_file", currentBaseFile))
				time.Sleep(5 * time.Second)
				continue
			}

			var count int
			err = func() error {
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

				return nil
			}()
			if err != nil {
				return err
			}

			zlog.Info("uploaded filtered file", zap.String("base_file", currentBaseFile), zap.Int("block_count", count))
		} else {
			zlog.Info("file already exists at destination", zap.String("base_file", currentBaseFile))
		}

		currentBase += 100
		// if currentBase >= endBlock {
		// 	break
		// }
	}
	return nil
}
