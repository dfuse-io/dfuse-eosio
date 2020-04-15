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

package bigt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/bigtable/bttest"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb"
	eos "github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func TestEnsureBigtableImplementsDBReaderInterface(t *testing.T) {
	var o eosdb.DBReader = &EOSDatabase{}
	_ = o
}

func TestBlocks(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	db.Blocks.PutBlock(Keys.Block("00000001deadbeef"), &pbcodec.Block{})

	require.NoError(t, db.Blocks.FlushMutations(context.Background()))
}

func TestBigtable_ReconstructDeosBlockWithHeader(t *testing.T) {
	blockID := "00000001deadbeef"
	key := Keys.Block(blockID)
	blk := &pbcodec.Block{Id: blockID, Header: &pbcodec.BlockHeader{Producer: "producer.1"}}
	expectedProducer := blk.Header.Producer // "producer.1"

	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	db.Blocks.PutBlock(key, blk)
	db.Blocks.PutMetaWritten(key)
	require.NoError(t, db.Blocks.FlushMutations(ctx))

	resp, err := db.GetBlock(context.Background(), blockID)
	require.NoError(t, err)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Block)
	assert.NotNil(t, resp.Block.Header)
	assert.Nil(t, resp.Block.Transactions)
	assert.Nil(t, resp.Block.TransactionTraces)
	assert.Equal(t, expectedProducer, resp.Block.Header.Producer)
}

func TestBigtable_StripTransactionsFromDeosBlock(t *testing.T) {
	blockID := "00000001deadbeef"
	key := Keys.Block(blockID)
	putBlock := &pbcodec.Block{
		Id: blockID,
		ImplicitTransactionOps: []*pbcodec.TrxOp{
			&pbcodec.TrxOp{
				Name: "somebeef",
			},
		},
		TransactionTraces: []*pbcodec.TransactionTrace{
			&pbcodec.TransactionTrace{
				Id: blockID,
			},
		},
		Transactions: []*pbcodec.TransactionReceipt{
			&pbcodec.TransactionReceipt{
				Id: "beefbeef13371337",
			},
		},
	}

	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	db.Blocks.PutBlock(key, putBlock)
	db.Blocks.PutMetaWritten(key)
	require.NoError(t, db.Blocks.FlushMutations(ctx))

	resp, err := db.GetBlock(context.Background(), blockID)
	require.NoError(t, err)

	assert.NotNil(t, resp.Block.ImplicitTransactionOps)
	assert.Nil(t, resp.Block.Transactions)
	assert.Nil(t, resp.Block.TransactionTraces)
}

func TestBigtable_GetBlock(t *testing.T) {

	testCases := []struct {
		name             string
		putBlock         *pbcodec.Block
		putHeader        *pbcodec.BlockHeader
		putIrreversible  bool
		fetchBlockID     string
		putBlockID       string
		expectedErr      error
		expectedBlockID  string
		expectedProducer string
	}{
		{
			name:             "Sunny path",
			putBlockID:       Keys.Block("00000001deadbeef"),
			putBlock:         &pbcodec.Block{Id: "00000001deadbeef"},
			putHeader:        &pbcodec.BlockHeader{Producer: "producer.1"},
			putIrreversible:  true,
			fetchBlockID:     "00000001deadbeef",
			expectedBlockID:  "00000001deadbeef",
			expectedProducer: "producer.1",
			expectedErr:      nil,
		},
		{
			name:             "Not Found",
			putBlockID:       Keys.Block("00000001deadbeef"),
			putBlock:         &pbcodec.Block{Id: "00000001deadbeef"},
			putHeader:        &pbcodec.BlockHeader{Producer: "producer.1"},
			putIrreversible:  true,
			fetchBlockID:     "00000002deadbeef",
			expectedBlockID:  "00000001deadbeef",
			expectedProducer: "producer.1",
			expectedErr:      kvdb.ErrNotFound,
		},
	}

	for _, c := range testCases {

		t.Run(c.name, func(t *testing.T) {
			db, cleanup := newServer(t)
			defer cleanup()
			ctx := context.Background()

			db.Blocks.PutBlock(c.putBlockID, c.putBlock)
			db.Blocks.PutMetaIrreversible(c.putBlockID, c.putIrreversible)
			db.Blocks.PutMetaWritten(c.putBlockID)
			require.NoError(t, db.Blocks.FlushMutations(ctx))

			resp, err := db.GetBlock(context.Background(), c.fetchBlockID)
			require.Equal(t, c.expectedErr, err)

			if c.expectedErr == nil {
				assert.Equal(t, c.expectedBlockID, resp.Block.Id)
				assert.Equal(t, true, resp.Irreversible)
			}
		})
	}
}

func TestBigtable_GetIrreversibleIDBlockNumAndID(t *testing.T) {
	type blockPut struct {
		block        *pbcodec.Block
		irreversible bool
	}

	testCases := []struct {
		name              string
		putBlocks         []blockPut
		fetchBlockNum     uint32
		fetchBlockID      string
		expectedBlockID   bstream.BlockRef
		expectedErr       error
		closestByBlockNum bstream.BlockRef
		closestErr        error
	}{
		{
			name: "Sunny path",
			putBlocks: []blockPut{
				{
					block:        &pbcodec.Block{Id: "00000004aaaaaaaa", DposIrreversibleBlocknum: 3},
					irreversible: true,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deadbeef", DposIrreversibleBlocknum: 4},
					irreversible: false,
				},
			},
			fetchBlockID:      "00000005deadbeef",
			expectedBlockID:   bstream.NewBlockRefFromID("00000004aaaaaaaa"),
			fetchBlockNum:     5,
			closestByBlockNum: bstream.NewBlockRefFromID("00000004aaaaaaaa"),
		},
		{
			name: "Inclusive closest, dpos-based for Block ID",
			putBlocks: []blockPut{
				{
					block:        &pbcodec.Block{Id: "00000004aaaaaaaa", DposIrreversibleBlocknum: 3},
					irreversible: true,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deadbeef", DposIrreversibleBlocknum: 4},
					irreversible: true,
				},
			},
			fetchBlockID:      "00000005deadbeef",
			fetchBlockNum:     5,
			expectedBlockID:   bstream.NewBlockRefFromID("00000004aaaaaaaa"),
			closestByBlockNum: bstream.NewBlockRefFromID("00000005deadbeef"),
		},
		{
			name: "Fork it up ! forked block 5, forked block 3",
			putBlocks: []blockPut{
				{
					block:        &pbcodec.Block{Id: "0000000388888888", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "0000000399999999", DposIrreversibleBlocknum: 2},
					irreversible: true,
				},
				{
					block:        &pbcodec.Block{Id: "00000004aaaaaaaa", DposIrreversibleBlocknum: 3},
					irreversible: true,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deadbeef", DposIrreversibleBlocknum: 4},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deed1ee7", DposIrreversibleBlocknum: 3},
					irreversible: false,
				},
			},
			fetchBlockNum:     5,
			closestByBlockNum: bstream.NewBlockRefFromID("00000004aaaaaaaa"),
			fetchBlockID:      "00000005deed1ee7",
			expectedBlockID:   bstream.NewBlockRefFromID("0000000399999999"),
			expectedErr:       nil,
		},
		{
			name: "No escape... missing irr",
			putBlocks: []blockPut{
				{
					block:        &pbcodec.Block{Id: "0000000388888888", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "0000000399999999", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "00000004aaaaaaab", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deadbeef", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
				{
					block:        &pbcodec.Block{Id: "00000005deed1ee7", DposIrreversibleBlocknum: 2},
					irreversible: false,
				},
			},
			fetchBlockID:      "00000005deed1ee7",
			fetchBlockNum:     5,
			expectedBlockID:   nil,
			expectedErr:       kvdb.ErrNotFound,
			closestByBlockNum: nil,
			closestErr:        kvdb.ErrNotFound,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			db, cleanup := newServer(t)
			defer cleanup()
			ctx := context.Background()

			for _, cc := range c.putBlocks {
				key := Keys.Block(cc.block.Id)
				db.Blocks.PutBlock(key, cc.block)
				if cc.irreversible {
					db.Blocks.PutMetaIrreversible(key, cc.irreversible)
				}
				db.Blocks.PutMetaWritten(key)
			}
			require.NoError(t, db.Blocks.FlushMutations(ctx))

			//fmt.Println("Fetching block num", c.fetchBlockNum)
			blockRef, err := db.GetClosestIrreversibleIDAtBlockNum(context.Background(), c.fetchBlockNum)
			if c.closestErr == nil {
				assert.Equal(t, c.closestByBlockNum, blockRef)
			} else {
				require.Equal(t, c.closestErr, err)
			}

			blockID, err := db.GetIrreversibleIDAtBlockID(context.Background(), c.fetchBlockID)
			if c.expectedErr == nil {
				assert.Equal(t, c.expectedBlockID, blockID)
			} else {
				require.Equal(t, c.expectedErr, err)
			}

		})
	}
}

func TestBigtable_GetBlockByNum(t *testing.T) {
	testCases := []struct {
		name             string
		putBlock         *pbcodec.Block
		putHeader        *pbcodec.BlockHeader
		putIrreversible  bool
		fetchBlockNum    uint32
		putBlockID       string
		expectedErr      error
		expectedBlockID  string
		expectedProducer string
	}{
		{
			name:             "Sunny path",
			putBlockID:       Keys.Block("00000001deadbeef"),
			putBlock:         &pbcodec.Block{Id: "00000001deadbeef"},
			putHeader:        &pbcodec.BlockHeader{Producer: "producer.1"},
			putIrreversible:  true,
			fetchBlockNum:    1,
			expectedBlockID:  "00000001deadbeef",
			expectedProducer: "producer.1",
			expectedErr:      nil,
		},
		{
			name:          "Not Found",
			putBlockID:    Keys.Block("00000001deadbeef"),
			putBlock:      &pbcodec.Block{Id: "00000001deadbeef"},
			putHeader:     &pbcodec.BlockHeader{Producer: "producer.1"},
			fetchBlockNum: 2,
			expectedErr:   kvdb.ErrNotFound,
		},
	}

	for _, test := range testCases {

		t.Run(test.name, func(t *testing.T) {
			db, cleanup := newServer(t)
			defer cleanup()
			ctx := context.Background()

			db.Blocks.PutBlock(test.putBlockID, test.putBlock)
			db.Blocks.PutMetaIrreversible(test.putBlockID, test.putIrreversible)
			db.Blocks.PutMetaWritten(test.putBlockID)
			require.NoError(t, db.Blocks.FlushMutations(ctx))

			resps, err := db.GetBlockByNum(context.Background(), test.fetchBlockNum)

			assert.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				assert.Equal(t, test.expectedBlockID, resps[0].Block.Id)
				assert.Equal(t, true, resps[0].Irreversible)
			}

		})
	}
}

func TestBigtable_GetLastWrittenBlockID(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	db.Blocks.PutBlock(Keys.Block("00000001a"), &pbcodec.Block{})
	db.Blocks.PutBlock(Keys.Block("00000002a"), &pbcodec.Block{})
	db.Blocks.PutBlock(Keys.Block("00000003a"), &pbcodec.Block{})
	db.BlocksLast.PutMetaWritten(Keys.Block("00000002a"))
	require.NoError(t, db.FlushAllMutations(ctx))

	lastBlockID, err := db.GetLastWrittenBlockID(ctx)
	require.NoError(t, err)

	assert.Equal(t, "00000002a", lastBlockID)
}

func TestBigtable_GetClosestIrreversibleIDAtBlockNum_Dedupe(t *testing.T) {
	t.Skip() // this isn't really useful, except to test the `latestCellFilter` manually

	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	db.Blocks.PutBlock(Keys.Block("00000001a"), &pbcodec.Block{Id: "00000001a"})
	db.Blocks.PutMetaIrreversible(Keys.Block("00000001a"), true)
	require.NoError(t, db.FlushAllMutations(ctx))

	time.Sleep(10 * time.Millisecond)

	db.Blocks.PutBlock(Keys.Block("00000001a"), &pbcodec.Block{Id: "00000001a"})
	db.Blocks.PutMetaIrreversible(Keys.Block("00000001a"), true)
	require.NoError(t, db.FlushAllMutations(ctx))

	LIB, err := db.GetClosestIrreversibleIDAtBlockNum(ctx, 4)
	require.NoError(t, err)
	assert.Equal(t, "00000001a", LIB)
}

func TestBigtable_GetClosestIrreversibleIDAtBlockNum_AsLast(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	db.Blocks.PutBlock(Keys.Block("00000001a"), &pbcodec.Block{Id: "00000001a"})
	db.Blocks.PutBlock(Keys.Block("00000002a"), &pbcodec.Block{Id: "00000002a"})
	db.Blocks.PutBlock(Keys.Block("00000003a"), &pbcodec.Block{Id: "00000003a"})
	db.Blocks.PutBlock(Keys.Block("00000004a"), &pbcodec.Block{Id: "00000004a"})
	db.Blocks.PutMetaIrreversible(Keys.Block("00000001a"), true)
	db.Blocks.PutMetaIrreversible(Keys.Block("00000002a"), true)
	require.NoError(t, db.FlushAllMutations(ctx))

	LIB, err := db.GetClosestIrreversibleIDAtBlockNum(ctx, 4)
	require.NoError(t, err)
	assert.Equal(t, bstream.NewBlockRefFromID("00000002a"), LIB)

	LIB, err = db.GetClosestIrreversibleIDAtBlockNum(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, bstream.NewBlockRefFromID("00000002a"), LIB)

	// special cases for blockNum 1, 2...
	LIB, err = db.GetClosestIrreversibleIDAtBlockNum(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, bstream.NewBlockRefFromID("0000000100000000000000000000000000000000000000000000000000000000"), LIB)

	LIB, err = db.GetClosestIrreversibleIDAtBlockNum(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, bstream.NewBlockRefFromID("0000000000000000000000000000000000000000000000000000000000000000"), LIB)
}

func TestBigtable_ListBlocks(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	putBlock := func(id string, irr bool) {
		key := Keys.Block(id)
		db.Blocks.PutBlock(key, &pbcodec.Block{Id: id})
		if irr {
			db.Blocks.PutMetaIrreversible(key, true)
		}
		db.Blocks.PutMetaWritten(key)
	}
	putBlock("00000001deadbeef", true)
	putBlock("00000002deadbeef", true)
	putBlock("00000003deadbeef", true)
	putBlock("00000004deadbeef", true)
	require.NoError(t, db.Blocks.FlushMutations(context.Background()))

	resps, err := db.ListBlocks(context.Background(), 4, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(resps))
	assert.Equal(t, "00000004deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000003deadbeef", resps[1].Block.Id)

	resps, err = db.ListBlocks(context.Background(), 10, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(resps))
	assert.Equal(t, "00000004deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000003deadbeef", resps[1].Block.Id)
}

func TestBigtable_ListSiblingBlocks(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	putBlock := func(id string, irr bool) {
		key := Keys.Block(id)
		db.Blocks.PutBlock(key, &pbcodec.Block{Id: id})
		if irr {
			db.Blocks.PutMetaIrreversible(key, true)
		}
		db.Blocks.PutMetaWritten(key)
	}
	putBlock("00000001deadbeef", true)
	putBlock("00000002deadbeef", true)
	putBlock("00000003deadbeef", true)
	putBlock("00000004deadbeef", true)
	putBlock("00000005deadbeef", true)
	require.NoError(t, db.Blocks.FlushMutations(context.Background()))

	//todo covert to test table ....
	resps, err := db.ListSiblingBlocks(context.Background(), 3, 2)
	require.NoError(t, err)
	require.Equal(t, 5, len(resps))
	assert.Equal(t, "00000005deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000004deadbeef", resps[1].Block.Id)
	assert.Equal(t, "00000003deadbeef", resps[2].Block.Id)
	assert.Equal(t, "00000002deadbeef", resps[3].Block.Id)
	assert.Equal(t, "00000001deadbeef", resps[4].Block.Id)

	resps, err = db.ListSiblingBlocks(context.Background(), 5, 2)
	require.NoError(t, err)
	require.Equal(t, 3, len(resps))
	assert.Equal(t, "00000005deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000004deadbeef", resps[1].Block.Id)
	assert.Equal(t, "00000003deadbeef", resps[2].Block.Id)

	resps, err = db.ListSiblingBlocks(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, 3, len(resps))
	assert.Equal(t, "00000003deadbeef", resps[0].Block.Id)
	assert.Equal(t, "00000002deadbeef", resps[1].Block.Id)
	assert.Equal(t, "00000001deadbeef", resps[2].Block.Id)
}

func TestListTransactionsForBlockID(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	testCases := []struct {
		name               string
		blockID            string
		startKey           string
		limit              int
		chainDiscriminator eosdb.ChainDiscriminator
		expectedKeys       []string
		expectedErr        error
	}{
		{
			name:        "NotFound",
			blockID:     "not_present_block_id",
			limit:       999,
			expectedErr: kvdb.ErrNotFound,
		},
		{
			name:    "LimitZero",
			blockID: "00000002deadbeef",
			limit:   0,
		},
		{
			name:         "SingleMatch_None",
			blockID:      "00000001deadbeef",
			limit:        999,
			startKey:     "0002",
			expectedKeys: []string{},
		},
		{
			name:         "SingleMatch_Single",
			blockID:      "00000001deadbeef",
			limit:        999,
			expectedKeys: []string{"01:00000001deadbeef"},
		},
		{
			name:         "MultiMatch_None",
			blockID:      "00000002deadbeef",
			limit:        999,
			startKey:     "0003",
			expectedKeys: []string{},
		},
		{
			name:         "MultiMatch_SingleMatch_ViaStartKey",
			blockID:      "00000002deadbeef",
			limit:        999,
			startKey:     "0000",
			expectedKeys: []string{"02aa:00000002deadbeef", "02bb:00000002deadbeef"},
		},
		{
			name:         "MultiMatch_SingleMatch_ViaLimit",
			blockID:      "00000002deadbeef",
			limit:        1,
			expectedKeys: []string{"02aa:00000002deadbeef"},
		},
		{
			name:         "MultiMatch_All",
			blockID:      "00000002deadbeef",
			limit:        999,
			expectedKeys: []string{"02aa:00000002deadbeef", "02bb:00000002deadbeef"},
		},
		{
			name:               "Match_Single_ThroughFork",
			blockID:            "00000005beefdead",
			limit:              999,
			chainDiscriminator: func(blockID string) bool { return "00000005beefdead" == blockID },
			expectedKeys:       []string{"05:00000005beefdead"},
		},
		{
			name:         "Match_Single_ThroughOtherFork",
			blockID:      "00000005deadbeef",
			limit:        999,
			expectedKeys: []string{"05:00000005deadbeef"},
		},
	}

	populateBigtableWithRows(t, db,
		blockInserter("00000001deadbeef", true, nil, []string{"01"}),
		blockInserter("00000002deadbeef", true, nil, []string{"02aa", "02bb"}),
		blockInserter("00000002beefdead", false, nil, []string{"01"}),
		blockInserter("00000003deadbeef", true, nil, []string{"03"}),
		blockInserter("00000004deadbeef", true, nil, []string{"04"}),
		blockInserter("00000005deadbeef", false, nil, []string{"05"}),
		blockInserter("00000005beefdead", false, nil, []string{"05"}),

		executeTransaction("01", "00000001deadbeef", true),
		executeTransaction("01", "00000002beefdead", false),
		executeTransaction("02aa", "00000002deadbeef", true),
		executeTransaction("02bb", "00000002deadbeef", true),
		executeTransaction("03", "00000003deadbeef", true),
		executeTransaction("04", "00000004deadbeef", false),
		executeTransaction("05", "00000005deadbeef", false),
		executeTransaction("05", "00000005beefdead", false),
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			chainDiscriminator := test.chainDiscriminator
			if chainDiscriminator == nil {
				chainDiscriminator = alwaysInChain
			}

			lst, err := db.ListTransactionsForBlockID(ctx, test.blockID, test.startKey, test.limit) // , chainDiscriminator) FIXME: move this to where we HAVE a discriminator

			if test.expectedErr != nil {
				require.Equal(t, test.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.Len(t, lst.Transactions, len(test.expectedKeys))
				for index, expectedKey := range test.expectedKeys {
					lifecycle := pbcodec.MergeTransactionEvents(lst.Transactions[index], chainDiscriminator)
					assert.Equal(t, expectedKey, transactionLifecycleKey(lifecycle))
				}
			}
		})
	}
}

func TestBigtable_GetAccount(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	name, err := eos.StringToName("eoscanada")
	assert.NoError(t, err)

	key := Keys.Account(name)
	db.Accounts.PutCreator(key, "aaaaaaaaaaaa", nil)
	db.Accounts.PutVerification(key, "github", json.RawMessage(`{}`))
	assert.NoError(t, db.FlushAllMutations(context.Background()))

	accountResponse, err := db.GetAccount(context.Background(), "eoscanada")
	require.NoError(t, err)

	assert.Equal(t, "eoscanada", accountResponse.Account)
	assert.Equal(t, "aaaaaaaaaaaa", accountResponse.Creator)
}

func TestBigtable_ListAccountNames(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	name1, _ := eos.StringToName("eoscanada2")
	name2, _ := eos.StringToName("eoscanada3")
	name3, _ := eos.StringToName("eoscanada1")

	// Adds a row with a key before actual `a:...` keys
	db.Accounts.PutCreator("0:random", "aaaaaaaaaaaa", nil)

	db.Accounts.PutCreator(Keys.Account(name1), "aaaaaaaaaaaa", nil)
	db.Accounts.PutCreator(Keys.Account(name2), "aaaaaaaaaaaa", nil)
	db.Accounts.PutCreator(Keys.Account(name3), "aaaaaaaaaaaa", nil)

	// Adds a row with a key before actual `b:...` keys
	db.Accounts.PutCreator("b:other", "aaaaaaaaaaaa", nil)

	assert.NoError(t, db.FlushAllMutations(context.Background()))

	accountNames, err := db.ListAccountNames(context.Background(), 1)
	require.NoError(t, err)

	// Not quite sure about the order here, will they always be in order?
	assert.Equal(t, []string{
		"eoscanada1",
		"eoscanada2",
		"eoscanada3",
	}, accountNames)
}

func TestBigtable_createAccountRowSets(t *testing.T) {
	assert.Equal(t, []bigtable.RowSet{
		bigtable.NewRange("a:", "a:3fffffffffffffff"),
		bigtable.NewRange("a:3fffffffffffffff", "a:7ffffffffffffffe"),
		bigtable.NewRange("a:7ffffffffffffffe", "a:bffffffffffffffd"),
		bigtable.NewRange("a:bffffffffffffffd", "a;"),
	}, createAccountRowSets(4))
}

func TestGetTransaction(t *testing.T) {
	testCases := []struct {
		name     string
		id       string
		rows     []DBInserter
		asserter func(t *testing.T, responses []*pbcodec.TransactionLifecycle)
	}{
		{
			name: "NotFound",
			id:   "01",
			rows: []DBInserter{
				executeTransaction("01", "1", true),
			},
			asserter: func(t *testing.T, responses []*pbcodec.TransactionLifecycle) {
				assert.Len(t, responses, 0)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			db, cleanup := newServer(t)
			defer cleanup()

			populateBigtableWithRows(t, db, test.rows...)
		})
	}
}

// FIXME: check if we still need this (hint: perhaps not? :)
//
// func TestGetTransaction_Found(t *testing.T) {
// 	db, cleanup := newServer(t)
// 	defer cleanup()

// 	populateBigtableWithRows(t, db,
// 		executeTransaction("02", "1a", false),
// 		executeTransaction("02", "2b", true),
// 	)

// 	trx, err := db.GetTransaction(context.Background(), "02", alwaysInChain)
// 	require.NoError(t, err)
// 	require.NotNil(t, trx)

// 	assert.Equal(t, "02:2b", transactionLifecycleKey(trx))
// }

//FIXME: check this one too:
//
// func TestGetTransaction_NotFoundError(t *testing.T) {
// 	db, cleanup := newServer(t)
// 	defer cleanup()

// 	_, err := db.GetTransaction(context.Background(), "trx_any", alwaysInChain)
// 	require.Equal(t, kvdb.ErrNotFound, err)
// }

func TestGetTransactionRowBatch(t *testing.T) {
	// TODO: clean this out.. it is not this lib's responsibility anymore to do that inference
	// as to which transaction row is the one the caller needs.
	t.Skip("delete me")

	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	testCases := []struct {
		name        string
		ids         []string
		expectedOut map[string]string
	}{
		{
			name:        "Empty",
			ids:         []string{},
			expectedOut: map[string]string{},
		},
		{
			name:        "One",
			ids:         []string{"03"},
			expectedOut: map[string]string{"03": "03:00000003deadbeef"},
		},
		{
			name: "Two",
			ids:  []string{"03", "04"},
			expectedOut: map[string]string{
				"03": "03:00000003deadbeef",
				"04": "04:00000004deadbeef",
			},
		},
		{
			name: "Irreversible first",
			ids:  []string{"01"},
			expectedOut: map[string]string{
				"01": "01:00000001deadbeef",
			},
		},
		{
			name: "HighestNum when not Irreversible",
			ids:  []string{"06"},
			expectedOut: map[string]string{
				"06": "06:00000006beefbeef",
			},
		},
		{
			name: "HighestNum when Irreversible",
			ids:  []string{"07"},
			expectedOut: map[string]string{
				"07": "07:00000004deadbeef",
			},
		},
		{
			name: "Irreversible even when not highest",
			ids:  []string{"05"},
			expectedOut: map[string]string{
				"05": "05:00000004deadbeef",
			},
		},
	}

	populateBigtableWithRows(t, db,
		blockInserter("00000001deadbeef", true, nil, []string{"01"}),
		blockInserter("00000002deadbeef", true, nil, []string{"02aa", "02bb"}),
		blockInserter("00000002beefdead", false, nil, []string{"01"}),
		blockInserter("00000003deadbeef", true, nil, []string{"03", "07"}),
		blockInserter("00000004deadbeef", true, nil, []string{"04", "05", "07"}),
		blockInserter("00000005beefdead", false, nil, []string{"05", "06"}),
		blockInserter("00000006beefbeef", false, nil, []string{"06"}),

		executeTransaction("01", "00000001deadbeef", true),
		executeTransaction("01", "00000002beefdead", false),
		executeTransaction("02aa", "00000002deadbeef", true),
		executeTransaction("02bb", "00000002deadbeef", true),
		executeTransaction("03", "00000003deadbeef", true),
		executeTransaction("04", "00000004deadbeef", true),
		executeTransaction("05", "00000004deadbeef", true),
		executeTransaction("05", "00000005beefdead", false),
		executeTransaction("06", "00000005beefdead", false),
		executeTransaction("06", "00000006beefbeef", false),

		executeTransaction("07", "00000003deadbeef", true),
		executeTransaction("07", "00000004deadbeef", true),
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			responses, err := db.GetTransactionEventsBatch(ctx, test.ids)
			require.NoError(t, err)

			require.Len(t, responses, len(test.expectedOut))

			respKeys := make(map[string]string)
			for k, v := range responses {
				fmt.Println("v is", k, v)
				// lifecycle, err := pbcodec.MergeTransactionEvents(v)
				// require.NoError(t, err)
				// respKeys[k] = lifecycle.Id + ":" + lifecycle.BlockId
			}
			assert.Equal(t, test.expectedOut, respKeys)
		})
	}
}

func TestGetTransactions(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()
	ctx := context.Background()

	testCases := []struct {
		name         string
		ids          []string
		expectedKeys []string
	}{
		{
			name:         "Empty",
			ids:          []string{},
			expectedKeys: nil,
		},
		{
			name:         "SingleKey_NoMatch",
			ids:          []string{"06"},
			expectedKeys: nil,
		},
		{
			name:         "SingleKey_SingleMatch",
			ids:          []string{"02aa"},
			expectedKeys: []string{"02aa:00000002deadbeef"},
		},
		{
			name:         "MultiKey_NoMatch",
			ids:          []string{"00", "06"},
			expectedKeys: nil,
		},
		{
			name:         "MultiKey_SingleMatch",
			ids:          []string{"00", "02aa", "06"},
			expectedKeys: []string{"02aa:00000002deadbeef"},
		},
		{
			name:         "MultiKey_SomeMatch",
			ids:          []string{"01", "02aa", "05"},
			expectedKeys: []string{"01:00000001deadbeef", "02aa:00000002deadbeef"},
		},
		{
			name:         "MultiKey_AllMatch_Consecutive",
			ids:          []string{"01", "02aa", "03"},
			expectedKeys: []string{"01:00000001deadbeef", "02aa:00000002deadbeef", "03:00000003deadbeef"},
		},
		{
			name:         "MultiKey_AllMatch_Disjoint",
			ids:          []string{"01", "04"},
			expectedKeys: []string{"01:00000001deadbeef", "04:00000004deadbeef"},
		},
	}

	populateBigtableWithRows(t, db,
		blockInserter("00000001deadbeef", true, nil, []string{"01"}),
		blockInserter("00000002deadbeef", true, nil, []string{"02aa", "02bb"}),
		blockInserter("00000002beefdead", false, nil, []string{"01"}),
		blockInserter("00000003deadbeef", true, nil, []string{"03"}),
		blockInserter("00000004deadbeef", true, nil, []string{"04"}),

		executeTransaction("01", "00000001deadbeef", true),
		executeTransaction("01", "00000002beefdead", false),
		executeTransaction("02aa", "00000002deadbeef", true),
		executeTransaction("02bb", "00000002deadbeef", true),
		executeTransaction("03", "00000003deadbeef", true),
		executeTransaction("04", "00000004deadbeef", true),
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			responses, err := db.GetTransactions(ctx, test.ids, alwaysInChain)
			require.NoError(t, err)

			require.Len(t, responses, len(test.expectedKeys))

			var keys []string
			for _, tx := range responses {
				keys = append(keys, transactionLifecycleKey(tx))
			}
			assert.Equal(t, test.expectedKeys, keys)
		})
	}
}

func newServer(t *testing.T) (db *EOSDatabase, cleanup func()) {
	srv, err := bttest.NewServer("localhost:0")
	require.NoError(t, err)
	conn, err := grpc.Dial(srv.Addr, grpc.WithInsecure())
	require.NoError(t, err)
	db, err = NewDriver("test", "dev", "dev", true, time.Second, 10, option.WithGRPCConn(conn))
	require.NoError(t, err)

	cleanup = func() {
		srv.Close()
		db.Close()
	}

	return db, cleanup
}

func transactionLifecycleKey(trxLifecycle *pbcodec.TransactionLifecycle) string {
	// We use the `Previous` field as the blockID part of the key. It's not intuitive, but
	// when we build the executionTransaction, we use this field to save the blockID. See
	// comment there on `putTransaction` for more info.
	return trxLifecycle.Id + ":" + trxLifecycle.ExecutionBlockHeader.Previous
}

func populateBigtableWithRows(t *testing.T, db *EOSDatabase, rows ...DBInserter) {
	for _, inserter := range rows {
		inserter(db)
	}

	err := db.FlushAllMutations(context.Background())
	require.NoError(t, err)
}

type DBInserter = func(db *EOSDatabase)

func mustDecodeString(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("error decoding ID string: %s", err))
	}
	return out
}

func trxIDsToTrxRefs(ids []string) *pbcodec.TransactionRefs {
	out := &pbcodec.TransactionRefs{}
	for _, id := range ids {
		out.Hashes = append(out.Hashes, mustDecodeString(id))
	}
	return out
}

func blockInserter(blockID string, irreversible bool, trxIDs, trxTraceIDs []string) DBInserter {
	return func(db *EOSDatabase) {
		putBlock(db, blockID, fmt.Sprintf("producer-%s", blockID), irreversible, trxIDsToTrxRefs(trxIDs), trxIDsToTrxRefs(trxTraceIDs))
	}
}

func executeTransaction(id string, blockID string, irreversible bool) DBInserter {
	return func(db *EOSDatabase) { putTransaction(db, id, blockID, irreversible) }
}

func putBlock(
	db *EOSDatabase,
	blockID string,
	producer string,
	irreversible bool,
	transactionRefs *pbcodec.TransactionRefs,
	transactionTraceRefs *pbcodec.TransactionRefs,
) {
	key := Keys.Block(blockID)

	db.Blocks.PutBlock(key, &pbcodec.Block{Id: blockID})
	db.Blocks.PutMetaIrreversible(key, irreversible)

	if transactionRefs != nil {
		db.Blocks.PutTransactionRefs(key, transactionRefs)
	}
	if transactionTraceRefs != nil {
		db.Blocks.PutTransactionTraceRefs(key, transactionTraceRefs)
	}
	db.Blocks.PutMetaWritten(key)
}

func putTransaction(db *EOSDatabase, id string, blockID string, irreversible bool) {
	key := Keys.Transaction(id, blockID)

	db.Transactions.PutTrx(key, &pbcodec.SignedTransaction{
		Transaction: &pbcodec.Transaction{
			Header: &pbcodec.TransactionHeader{DelaySec: 1},
		},
	})
	db.Transactions.PutTrace(key, &pbcodec.TransactionTrace{
		Id:       id,
		BlockNum: uint64(eos.BlockNum(blockID)),
		Receipt: &pbcodec.TransactionReceiptHeader{
			Status: pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
		},
	})

	// In some tests, we have only executed transactions but from different blocks. The tests checks
	// which version is picked by the sticher. However, TransactionLifecycle does not have the Id.
	// As such, in tests, we hijack the `Previous` field an put the `blockID` into. Later, we will be
	// able to inspect the `TransactionLifecycle.ExecutionBlockHeader` and use the `Previous` as our
	// trx block ID.
	db.Transactions.PutBlockHeader(key, &pbcodec.BlockHeader{Previous: blockID, Producer: "eosio"})
	db.Transactions.PutMetaIrreversible(key, irreversible)
	db.Transactions.PutPublicKeys(key, []string{"any"})
	db.Transactions.PutDTrxCreatedBy(key, &pbcodec.ExtDTrxOp{BlockId: blockID})
	db.Transactions.PutDTrxCanceledBy(key, &pbcodec.ExtDTrxOp{BlockId: blockID})
	db.Transactions.PutMetaWritten(key, true)
}

func putTimelineBlock(db *EOSDatabase, blockTime time.Time, blockID string) {
	key := Keys.TimelineBlockForward(blockTime, blockID)
	db.Timeline.PutMetaExists(key)
	key = Keys.TimelineBlockReverse(blockTime, blockID)
	db.Timeline.PutMetaExists(key)
}

func timelineBlock(blockTime time.Time, blockID string) DBInserter {
	return func(db *EOSDatabase) { putTimelineBlock(db, blockTime, blockID) }
}

func TestTimeline(t *testing.T) {
	db, cleanup := newServer(t)
	defer cleanup()

	t1, err := time.Parse(time.RFC3339, "2012-11-01T22:08:41+00:00")
	require.NoError(t, err)
	t2 := t1.Add(time.Millisecond * 500)
	t3 := t2.Add(time.Millisecond * 500)
	t4 := t3.Add(time.Millisecond * 500)

	populateBigtableWithRows(t, db,
		timelineBlock(t1, "00000001a"),
		timelineBlock(t2, "00000002a"),
		timelineBlock(t3, "00000003a"),
		timelineBlock(t4, "00000004a"),
	)

	b, ft, err := db.BlockIDBefore(context.Background(), t2, true)
	assert.NoError(t, err)
	assert.Equal(t, "00000002a", b)
	assert.True(t, t2.Equal(ft))

	b, err = db.BlockIDAt(context.Background(), t1)
	assert.NoError(t, err)
	assert.Equal(t, "00000001a", b)

	b, err = db.BlockIDAt(context.Background(), t4)
	assert.NoError(t, err)
	assert.Equal(t, "00000004a", b)

	b, ft, err = db.BlockIDBefore(context.Background(), t2, false)
	assert.NoError(t, err)
	assert.Equal(t, "00000001a", b)
	assert.True(t, t1.Equal(ft))

	b, ft, err = db.BlockIDAfter(context.Background(), t2, false)
	assert.NoError(t, err)
	assert.Equal(t, "00000003a", b)
	assert.True(t, t3.Equal(ft))

	b, _, err = db.BlockIDAfter(context.Background(), t2, true)
	assert.NoError(t, err)
	assert.Equal(t, "00000002a", b)

	b, _, err = db.BlockIDBefore(context.Background(), t2.Add(-time.Millisecond*123), true)
	assert.NoError(t, err)
	assert.Equal(t, "00000001a", b)

	b, _, err = db.BlockIDAfter(context.Background(), t2.Add(time.Millisecond*123), true)
	assert.NoError(t, err)
	assert.Equal(t, "00000003a", b)

	_, _, err = db.BlockIDAfter(context.Background(), t4.Add(time.Millisecond*123), true)
	assert.Error(t, err)

	_, _, err = db.BlockIDBefore(context.Background(), t1.Add(-time.Millisecond*123), true)
	assert.Error(t, err)
}
