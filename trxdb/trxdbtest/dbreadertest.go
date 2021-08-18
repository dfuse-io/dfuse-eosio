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

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/kvdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dbReaderTests = []DriverTestFunc{
	TestGetBlock,
	TestGetBlockByNum,
	TestListBlocks,
	TestListSiblingBlocks,
	TestGetClosestIrreversibleIDAtBlockNum,
	TestGetIrreversibleIDAtBlockID,
	TestGetLastWrittenBlockID,
}

func TestGetBlock(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		block         *pbcodec.Block
		blockID       string
		expectErr     error
		expectBlockID string
	}{
		{
			name:          "sunny path",
			block:         ct.Block(t, "00000002aa"),
			blockID:       "00000002aa",
			expectBlockID: "00000002aa",
		},
		{
			name:      "block does not exist",
			block:     ct.Block(t, "00000002aa"),
			blockID:   "00000003aa",
			expectErr: kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			require.NoError(t, db.PutBlock(ctx, test.block))
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetBlock(ctx, test.blockID)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.block.Id, resp.Block.Id)
			}
		})
	}
}

func TestGetBlockByNum(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name           string
		blocks         []*pbcodec.Block
		blockNum       uint32
		expectErr      error
		expectBlockIDs []string
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000004aa"),
			},
			blockNum:       3,
			expectBlockIDs: []string{"00000003aa"},
		},
		{
			name: "block does not exist",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
			},
			blockNum:  3,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "return multiple blocks with same number",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000002dd"),
			},
			blockNum:       2,
			expectBlockIDs: []string{"00000002aa", "00000002dd"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetBlockByNum(ctx, test.blockNum)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				ids := []string{}
				for _, blk := range resp {
					ids = append(ids, blk.Id)
				}
				assert.ElementsMatch(t, test.expectBlockIDs, ids)
			}
		})
	}
}

func TestGetClosestIrreversibleIDAtBlockNum(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		irrBlock      []*pbcodec.Block
		blockNum      uint32
		expectBlockID string
		expectErr     error
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			irrBlock: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
			},
			blockNum:      8,
			expectBlockID: "00000005aa",
		},
		{
			name: "no irr blocks",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			irrBlock:  nil,
			blockNum:  8,
			expectErr: kvdb.ErrNotFound,
		},
		{
			name: "looking for irr block",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			irrBlock: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000004aa"),
				ct.Block(t, "00000005aa"),
			},
			blockNum:      5,
			expectBlockID: "00000005aa",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
			}

			for _, blk := range test.irrBlock {
				require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetClosestIrreversibleIDAtBlockNum(ctx, test.blockNum)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectBlockID, resp.ID())
			}
		})
	}
}
func TestGetLastWrittenBlockID(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		expectBlockID string
		expectError   error
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			expectBlockID: "00000008aa",
		},
		{
			name:          "not found",
			blocks:        []*pbcodec.Block{},
			expectBlockID: "",
			expectError:   kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()
			db, clean := driverFactory()
			defer clean()

			for _, blk := range test.blocks {
				require.NoError(t, db.PutBlock(ctx, blk))
			}
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetLastWrittenBlockID(ctx)

			if test.expectError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectError, err)
			}

			assert.Equal(t, test.expectBlockID, resp)
		})
	}
}

func TestGetIrreversibleIDAtBlockID(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		blocks        []*pbcodec.Block
		irrBlock      []*pbcodec.Block
		blockID       string
		expectBlockID string
		expectErr     error
	}{
		{
			name: "sunny path",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			irrBlock: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
			},
			blockID:       "00000008aa",
			expectBlockID: "00000007aa",
		},
		{
			name: "no irr blocks",
			blocks: []*pbcodec.Block{
				ct.Block(t, "00000002aa"),
				ct.Block(t, "00000003aa"),
				ct.Block(t, "00000005aa"),
				ct.Block(t, "00000006aa"),
				ct.Block(t, "00000007aa"),
				ct.Block(t, "00000008aa"),
			},
			irrBlock:  nil,
			blockID:   "00000008aa",
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
			}

			for _, blk := range test.irrBlock {
				require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
			}

			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetIrreversibleIDAtBlockID(ctx, test.blockID)

			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectBlockID, resp.ID())
			}
		})
	}
}

func TestListBlocks(t *testing.T, driverFactory DriverFactory) {

	ctx := context.Background()
	driver, cleanup := driverFactory()
	defer cleanup()

	putBlock := func(id string) {
		b := ct.Block(t, id)
		err := driver.PutBlock(ctx, b)
		require.NoError(t, err)
		err = driver.UpdateNowIrreversibleBlock(ctx, b)
		require.NoError(t, err)
	}

	putBlock("00000003deadbeef")
	putBlock("00000004deadbeef")
	putBlock("00000005deadbeef")
	putBlock("00000006deadbeef")
	err := driver.Flush(ctx)
	require.NoError(t, err)

	resps, err := driver.ListBlocks(context.Background(), 4, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(resps))
	assert.Equal(t, "00000004deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000003deadbeef", resps[1].Block.Id)

	resps, err = driver.ListBlocks(context.Background(), 10, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(resps))
	assert.Equal(t, "00000006deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000005deadbeef", resps[1].Block.Id)

	resps, err = driver.ListBlocks(context.Background(), 2, 2)
	require.NoError(t, err)
	require.Equal(t, 0, len(resps))
}

func TestListSiblingBlocks(t *testing.T, driverFactory DriverFactory) {

	ctx := context.Background()
	driver, cleanup := driverFactory()
	defer cleanup()

	putBlock := func(id string) {
		b := ct.Block(t, id)
		err := driver.UpdateNowIrreversibleBlock(ctx, b)
		require.NoError(t, err)
		err = driver.PutBlock(ctx, b)
		require.NoError(t, err)
	}

	putBlock("00000003aa")
	putBlock("00000004aa")
	putBlock("00000005aa")
	putBlock("00000006aa")
	putBlock("00000007aa")

	driver.Flush(ctx)
	//todo covert to test table ....
	resps, err := driver.ListSiblingBlocks(context.Background(), 5, 2)
	require.NoError(t, err)
	require.Equal(t, 5, len(resps))
	assert.Equal(t, "00000007aa", resps[0].Block.Id)
	assert.Equal(t, "00000006aa", resps[1].Block.Id)
	assert.Equal(t, "00000005aa", resps[2].Block.Id)
	assert.Equal(t, "00000004aa", resps[3].Block.Id)
	assert.Equal(t, "00000003aa", resps[4].Block.Id)

	resps, err = driver.ListSiblingBlocks(context.Background(), 7, 2)
	require.NoError(t, err)
	require.Equal(t, 3, len(resps))
	assert.Equal(t, "00000007aa", resps[0].Block.Id)
	assert.Equal(t, "00000006aa", resps[1].Block.Id)
	assert.Equal(t, "00000005aa", resps[2].Block.Id)

	resps, err = driver.ListSiblingBlocks(context.Background(), 3, 2)
	require.NoError(t, err)
	require.Equal(t, 3, len(resps))
	assert.Equal(t, "00000005aa", resps[0].Block.Id)
	assert.Equal(t, "00000004aa", resps[1].Block.Id)
	assert.Equal(t, "00000003aa", resps[2].Block.Id)

}
