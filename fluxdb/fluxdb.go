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
	"fmt"

	"github.com/dfuse-io/shutter"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"go.uber.org/zap"
)

type FluxDB struct {
	*shutter.Shutter
	store  store.KVStore
	source bstream.Source

	idxCache               *indexCache
	newRowsPerTable        map[string]uint32
	newRowsIndexingTrigger int

	SpeculativeWritesFetcher func(ctx context.Context, headBlockID string, upToBlockNum uint32) (speculativeWrites []*WriteRequest)
	HeadBlock                func(ctx context.Context) bstream.BlockRef

	shardIndex int
	shardCount int
	stopBlock  uint32

	ready bool
}

func New(kvStore store.KVStore) *FluxDB {
	return &FluxDB{
		Shutter:                shutter.New(),
		store:                  kvStore,
		newRowsPerTable:        make(map[string]uint32),
		idxCache:               newIndexCache(),
		newRowsIndexingTrigger: 1000,
	}
}

func (fdb *FluxDB) Launch(devMode bool, httpListenAddr string) {
	fdb.OnTerminating(func(e error) {
		zlog.Info("shutting down fluxdb's source")
		fdb.source.Shutdown(e)
		zlog.Info("source shutdown")
	})

	if devMode {
		// in dev mode we do not want to run the pipeline just a read
		zlog.Info("not using a pipeline, waiting forever (serve mode)")
		fdb.SpeculativeWritesFetcher = func(ctx context.Context, headBlockID string, upToBlockNum uint32) (speculativeWrites []*WriteRequest) {
			return nil
		}

		fdb.HeadBlock = func(ctx context.Context) bstream.BlockRef {
			lastWrittenBlock, err := fdb.FetchLastWrittenBlock(ctx)
			if err != nil {
				fdb.Shutdown(fmt.Errorf("failed fetching the last written block ID: %w", err))
				return nil
			}
			return lastWrittenBlock
		}

		<-fdb.Terminating()
		zlog.Info("fluxdb server completed")

	} else {
		// running the pipeline, this call is blocking
		fdb.source.Run()
		<-fdb.source.Terminating()

		err := fdb.source.Err()

		zlog.Info("fluxdb source shutdown", zap.Error(err))
		fdb.Shutdown(err)
	}

	return
}

func (fdb *FluxDB) SetSharding(shardIndex, shardCount int) {
	fdb.shardIndex = shardIndex
	fdb.shardCount = shardCount
}

func (fdb *FluxDB) SetStopBlock(stopBlock uint32) {
	fdb.stopBlock = stopBlock
}

func (fdb *FluxDB) IsSharding() bool {
	return fdb.shardCount != 0
}

func (fdb *FluxDB) Close() error {
	return fdb.store.Close()
}

func (fdb *FluxDB) IsReady() bool {
	return fdb.ready
}

// SetReady marks the process as ready, meaning it has crossed the
// "close to real-time" threshold.
func (fdb *FluxDB) SetReady() {
	fdb.ready = true
}
