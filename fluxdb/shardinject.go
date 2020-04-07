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
	"go.uber.org/zap"
)

type ShardInjector struct {
	shardsStore dstore.Store
	db          *FluxDB
}

func NewShardInjector(shardsStore dstore.Store, db *FluxDB) *ShardInjector {
	return &ShardInjector{
		shardsStore: shardsStore,
		db:          db,
	}
}

func (s *ShardInjector) Run() (err error) {
	err = s.shardsStore.Walk("", "", func(filename string) error {
		zlog.Info("processing shard file", zap.String("filename", filename))

		reader, err := s.shardsStore.OpenObject(filename)
		if err != nil {
			return fmt.Errorf("opening object from shards store %q: %s", filename, err)
		}
		defer reader.Close()

		requests, err := readWriteRequestsForBatch(reader)
		if err != nil {
			return fmt.Errorf("unable to read all write requests in batch %q: %s", filename, err)
		}

		// TODO: make sure the `LAST` is replaced by a shard-aware value.
		if err := s.db.WriteBatch(context.Background(), requests); err != nil {
			return fmt.Errorf("write batch %q: %s", filename, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking shards store: %s", err)
	}

	return nil
}

func readWriteRequestsForBatch(reader io.Reader) ([]*WriteRequest, error) {
	decoder := gob.NewDecoder(reader)

	var requests []*WriteRequest
	for {
		req := &WriteRequest{}
		err := decoder.Decode(req)
		if err == io.EOF {
			return requests, nil
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read WriteRequest: %s", err)
		}
		requests = append(requests, req)
	}
}
