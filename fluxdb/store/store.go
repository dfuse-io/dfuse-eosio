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

package store

import (
	"context"
	"errors"

	"github.com/dfuse-io/bstream"
	"go.uber.org/zap"
)

// BreakScan error can be used on scanning function to notify termination of scanning
var BreakScan = errors.New("break scan")

var ErrNotFound = errors.New("not found")

type Batch interface {
	Flush(ctx context.Context) error
	FlushIfFull(ctx context.Context) error

	// FIXME: Maybe the batch "adder/setter" should not event care about the key and compute
	//        it straight? Since this is per storage engine, it would be a good place since
	//        all saved element would pass through those methods...
	SetRow(key string, value []byte)
	SetLast(key string, value []byte)
	SetIndex(key string, value []byte)

	Reset()
}

type OnBlockRef func(key string, blockRef bstream.BlockRef) error
type OnTabletRow func(key string, value []byte) error

// KVStore represents the abstraction needed by FluxDB to correctly use different
// underlying KV storage engine.
//
// TODO: For now, most functions receive the actual pre-computed key to fetch or to write.
//       While we affine the interface, we will see if make it lower-level (i.e. `Get(key)`
//       directly) or if we keep higher level and defer more job into the implementation
//       (i.e. removing the key parameters).
type KVStore interface {
	Close() error

	// NewBatch returns the batch implementation suitable for the underlying store.
	//
	// FIXME: For now, we kept the `logger` parameter, not clear the intent was here. Let's
	//        decide if this was required later on when we are close to finish the refactoring.
	NewBatch(logger *zap.Logger) Batch

	FetchIndex(ctx context.Context, tableKey, prefixKey, keyStart string) (rowKey string, rawIndex []byte, err error)

	HasTabletRow(ctx context.Context, tabletKey string) (exists bool, err error)

	FetchTabletRow(ctx context.Context, key string) (value []byte, err error)

	FetchTabletRows(ctx context.Context, keys []string, onTabletRow OnTabletRow) error

	FetchSingletEntry(ctx context.Context, keyStart, keyEnd string) (key string, value []byte, err error)

	ScanTabletRows(ctx context.Context, keyStart, keyEnd string, onTabletRow OnTabletRow) error

	// FetchLastWrittenBlock returns the latest written block reference that was correctly
	// committed to the storage system.
	//
	// If no block was ever written yet, this must return `nil, ErrNotFound`.
	FetchLastWrittenBlock(ctx context.Context, key string) (out bstream.BlockRef, err error)

	ScanLastShardsWrittenBlock(ctx context.Context, keyPrefix string, onBlockRef OnBlockRef) error
}
