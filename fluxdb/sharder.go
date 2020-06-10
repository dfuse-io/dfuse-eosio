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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/abourget/llerrgroup"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dstore"
	"github.com/minio/highwayhash"
	"go.uber.org/zap"
)

type Sharder struct {
	shardsStore dstore.Store
	startBlock  uint32
	stopBlock   uint32
	shardCount  int

	// A slice of shards, each shard is itself a slice of WriteRequest, one per block processed in this batch.
	// So, assuming 2 shards with 5 blocks, that would yield `[0][#5, #6, #7, #8, #9], [1][#5, #6, #7, #8, #9]`.
	buffers     []*bytes.Buffer
	gobEncoders []*gob.Encoder
}

func NewSharder(shardsStore dstore.Store, shardCount int, startBlock, stopBlock uint32) *Sharder {
	s := &Sharder{
		buffers:     make([]*bytes.Buffer, shardCount),
		gobEncoders: make([]*gob.Encoder, shardCount),
		shardCount:  shardCount,
		shardsStore: shardsStore,
		startBlock:  startBlock,
		stopBlock:   stopBlock,
	}

	for i := 0; i < shardCount; i++ {
		buf := bytes.NewBuffer(nil)
		s.buffers[i] = buf
		s.gobEncoders[i] = gob.NewEncoder(buf)
	}

	return s
}

func (s *Sharder) ProcessBlock(rawBlk *bstream.Block, rawObj interface{}) error {
	if rawBlk.Num()%600 == 0 {
		zlog.Info("processing block (printed each 600 blocks)", zap.Stringer("block", rawBlk))
	}

	fObj := rawObj.(*forkable.ForkableObject)
	if fObj.Step != forkable.StepIrreversible {
		panic("unsupported, received step is not irreversible")
	}

	unshardedRequest := fObj.Obj.(*WriteRequest)
	if unshardedRequest.BlockNum > s.stopBlock {
		err := s.writeShards()
		if err != nil {
			return fmt.Errorf("unable to write shards to store: %s", err)
		}

		return ErrCleanSourceStop
	}

	// Compute the N shard write requests, 1 write request per shard, the slice index is the shard index
	shardedRequests := make([]*WriteRequest, s.shardCount)
	for _, entry := range unshardedRequest.SingletEntries {
		shardIndex := s.goesToShard(entry.Key())

		var shardedRequest *WriteRequest
		if shardedRequest = shardedRequests[shardIndex]; shardedRequest == nil {
			shardedRequest = &WriteRequest{}
			shardedRequests[shardIndex] = shardedRequest
		}

		shardedRequest.SingletEntries = append(shardedRequest.SingletEntries, entry)
	}

	for _, row := range unshardedRequest.TabletRows {
		shardIndex := s.goesToShard(row.Key())

		var shardedRequest *WriteRequest
		if shardedRequest = shardedRequests[shardIndex]; shardedRequest == nil {
			shardedRequest = &WriteRequest{}
			shardedRequests[shardIndex] = shardedRequest
		}

		shardedRequest.TabletRows = append(shardedRequest.TabletRows, row)
	}

	// Loop over N shards computed above, and assign them correctly to the global shards slice
	for shardIndex, encoder := range s.gobEncoders {
		shardedRequest := shardedRequests[shardIndex]
		if shardedRequest == nil {
			shardedRequest = &WriteRequest{}
		}

		shardedRequest.BlockNum = unshardedRequest.BlockNum
		shardedRequest.BlockID = unshardedRequest.BlockID

		if err := encoder.Encode(shardedRequest); err != nil {
			return fmt.Errorf("encoding sharded request: %s", err)
		}
	}

	return nil
}

var emptyHashKey [32]byte

func (s *Sharder) goesToShard(key string) int {
	bigInt := highwayhash.Sum64([]byte(key), emptyHashKey[:])
	elementShard := bigInt % uint64(s.shardCount)
	return int(elementShard)
}

func (s *Sharder) writeShards() error {
	eg := llerrgroup.New(12)
	for shardIndex, buffer := range s.buffers {
		if eg.Stop() {
			break
		}

		shardIndex := shardIndex
		buffer := buffer
		eg.Go(func() error {
			baseName := fmt.Sprintf("%03d/%010d-%010d", shardIndex, s.startBlock, s.stopBlock)

			zlog.Info("encoding shard", zap.Int("shard_index", shardIndex), zap.Uint32("start", s.startBlock), zap.Uint32("stop", s.stopBlock))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			err := s.shardsStore.WriteObject(ctx, baseName, bytes.NewReader(buffer.Bytes()))
			if err != nil {
				return fmt.Errorf("unable to correctly write shard %d: %s", shardIndex, err)
			}

			buffer.Truncate(0)

			return nil
		})
	}
	return eg.Wait()
}
