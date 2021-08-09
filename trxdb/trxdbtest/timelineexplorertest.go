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

package trxdbtest

import (
	"context"
	"testing"
	"time"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/kvdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noon = time.Date(2020, time.February, 02, 12, 0, 0, 0, time.UTC)
var twopm = time.Date(2020, time.February, 02, 14, 0, 0, 0, time.UTC)
var fourpm = time.Date(2020, time.February, 02, 16, 0, 0, 0, time.UTC)

var timelineExplorerTests = []DriverTestFunc{
	TestBlockIDAt,
	TestBlockIDAfter,
	TestBlockIDBefore,
}

func TestBlockIDAt(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		time          time.Time
		expectBlockID string
		expectErr     error
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			time:          noon,
			expectBlockID: "00000008aa",
		},
		{
			name: "no block that matches",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			time:      fourpm,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name:      "no blocks",
			blocks:    []*pbcodec.Block{},
			time:      fourpm,
			expectErr: kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
				require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			id, err := db.BlockIDAt(ctx, test.time)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectBlockID, id)
			}
		})
	}
}

func TestBlockIDAfter(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		time          time.Time
		inclusive     bool
		expectBlockID string
		expectTime    time.Time
		expectErr     error
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(fourpm)),
			},
			time:          twopm,
			expectTime:    fourpm,
			expectBlockID: "00000003aa",
		},
		{
			name: "no block that matches",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			time:      fourpm,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "should not match block when not inclusive",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			inclusive: false,
			time:      twopm,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "should  match block when inclusive",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			inclusive:     true,
			time:          twopm,
			expectTime:    twopm,
			expectBlockID: "00000003aa",
		},
		{
			name:      "no blocks",
			blocks:    []*pbcodec.Block{},
			time:      fourpm,
			expectErr: kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
				require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			id, foundTime, err := db.BlockIDAfter(ctx, test.time, test.inclusive)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectBlockID, id)
				assert.Equal(t, test.expectTime, foundTime.UTC())
			}
		})
	}
}

func TestBlockIDBefore(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		time          time.Time
		inclusive     bool
		expectBlockID string
		expectTime    time.Time
		expectErr     error
	}{
		{
			name: "no block that matches",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(twopm)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(fourpm)),
			},
			time:      noon,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(fourpm)),
			},
			time:          twopm,
			expectTime:    noon,
			expectBlockID: "00000008aa",
		},
		{
			name: "should not match block when not inclusive",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			inclusive: false,
			time:      noon,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "should  match block when inclusive",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000008aa", ct.BlockTimestamp(noon)),
				ct.Block(t, "00000003aa", ct.BlockTimestamp(twopm)),
			},
			inclusive:     true,
			time:          noon,
			expectTime:    noon,
			expectBlockID: "00000008aa",
		},
		{
			name:      "no blocks",
			blocks:    []*pbcodec.Block{},
			time:      fourpm,
			expectErr: kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
				require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			id, foundTime, err := db.BlockIDBefore(ctx, test.time, test.inclusive)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectBlockID, id)
				assert.Equal(t, test.expectTime, foundTime.UTC())
			}
		})
	}
}
