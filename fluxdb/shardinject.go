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
	"encoding/gob"
	"fmt"
	"io"

	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type ShardInjector struct {
	*shutter.Shutter

	shardsStore dstore.Store
	db          *FluxDB
}

func NewShardInjector(shardsStore dstore.Store, db *FluxDB) *ShardInjector {
	return &ShardInjector{
		Shutter:     shutter.New(),
		shardsStore: shardsStore,
		db:          db,
	}
}

func (s *ShardInjector) Run() (err error) {
	ctx, cancelInjector := context.WithCancel(context.Background())
	s.Shutter.OnTerminating(func(_ error) {
		cancelInjector()
	})

	startAfter, err := s.db.getLastBlock(ctx)
	if err != nil {
		return err
	}

	err = s.shardsStore.Walk(ctx, "", "", func(filename string) error {
		zlog.Info("processing shard file", zap.String("filename", filename))

		reader, err := s.shardsStore.OpenObject(ctx, filename)
		if err != nil {
			return fmt.Errorf("opening object from shards store %q: %w", filename, err)
		}
		defer reader.Close()

		requests, err := readWriteRequestsForBatch(reader, uint32(startAfter.Num()))
		if err != nil {
			return fmt.Errorf("unable to read all write requests in batch %q: %w", filename, err)
		}

		// TODO: make sure the `LAST` is replaced by a shard-aware value.
		if err := s.db.WriteBatch(ctx, requests); err != nil {
			return fmt.Errorf("write batch %q: %w", filename, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walking shards store: %w", err)
	}

	return nil
}

func readWriteRequestsForBatch(reader io.Reader, startAfter uint32) ([]*WriteRequest, error) {
	decoder := gob.NewDecoder(reader)

	var requests []*WriteRequest
	for {
		req := &WriteRequest{}
		err := decoder.Decode(req)
		if err == io.EOF {
			return requests, nil
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read WriteRequest: %w", err)
		}
		if req.BlockNum <= startAfter {
			continue
		}
		requests = append(requests, req)

	}
}
