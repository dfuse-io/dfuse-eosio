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

package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (db *DB) GetLastWrittenBlockID(ctx context.Context) (blockID string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	it := db.irrBlockStore.Scan(ctx, Keys.StartOfIrrBlockTable(), Keys.EndOfIrrBlockTable(), 1)
	found := it.Next()
	if err := it.Err(); err != nil {
		return "", err
	}
	if !found {
		return "", kvdb.ErrNotFound
	}
	key := it.Item().Key
	db.logger.Debug("retrieved key", zap.ByteString("packed_key", key))
	blockID = Keys.UnpackIrrBlocksKey(key)
	return
}

func (db *DB) GetBlock(ctx context.Context, id string) (blk *pbcodec.BlockWithRefs, err error) {
	value, err := db.blksReadStore.Get(ctx, Keys.PackBlocksKey(id))

	if err == store.ErrNotFound {
		return nil, kvdb.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("unable to get block: %w", err)
	}

	blockRow := &pbtrxdb.BlockRow{}
	db.dec.MustInto(value, blockRow)
	return db.blockRowToBlockWithRef(ctx, blockRow)
}

func (db *DB) GetBlockByNum(ctx context.Context, num uint32) (out []*pbcodec.BlockWithRefs, err error) {
	db.logger.Debug("get block by num", zap.Uint32("block_num", num))
	it := db.blksReadStore.Scan(ctx, Keys.PackBlockNumPrefix(num), Keys.PackBlockNumPrefix(num-1), 0)
	for it.Next() {
		kv := it.Item()

		blockRow := &pbtrxdb.BlockRow{}
		db.dec.MustInto(kv.Value, blockRow)
		blk, err := db.blockRowToBlockWithRef(ctx, blockRow)
		if err != nil {
			return nil, fmt.Errorf("item value: to block with ref: %w", err)
		}
		out = append(out, blk)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return
}

func (db *DB) blockRowToBlockWithRef(ctx context.Context, blockRow *pbtrxdb.BlockRow) (*pbcodec.BlockWithRefs, error) {
	blk := &pbcodec.BlockWithRefs{
		Id:                      blockRow.Block.Id,
		Block:                   blockRow.Block,
		ImplicitTransactionRefs: blockRow.ImplicitTrxRefs,
		TransactionRefs:         blockRow.TrxRefs,
		TransactionTraceRefs:    blockRow.TrxRefs,
	}

	//todo: add a test to check the irreversibility
	_, err := db.blksReadStore.Get(ctx, Keys.PackIrrBlocksKey(blockRow.Block.Id))
	if err != nil && err != store.ErrNotFound {
		return nil, fmt.Errorf("get irr block: txn get: %w", err)
	}

	blk.Irreversible = err == nil

	return blk, nil
}

func (db *DB) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (ref bstream.BlockRef, err error) {
	db.logger.Debug("get closest irr id at block num", zap.Uint32("block_num", num))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	it := db.blksReadStore.Scan(ctx, Keys.PackIrrBlockNumPrefix(num), Keys.EndOfIrrBlockTable(), 1)
	found := it.Next()
	if err := it.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, kvdb.ErrNotFound
	}

	blockID := Keys.UnpackIrrBlocksKey(it.Item().Key)
	return bstream.NewBlockRefFromID(bstream.BlockRefFromID(blockID)), nil
}

func (db *DB) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (ref bstream.BlockRef, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	blk, err := db.GetBlock(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("get irreversible id at block id: get block: %w", err)
	}

	dposIrrNum := blk.Block.DposIrreversibleBlocknum

	db.logger.Debug("get irr block by num", zap.Uint32("block_num", dposIrrNum))
	it := db.blksReadStore.Scan(ctx, Keys.PackIrrBlockNumPrefix(dposIrrNum), Keys.PackIrrBlockNumPrefix(dposIrrNum-1), 1)
	found := it.Next()
	if err := it.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, kvdb.ErrNotFound
	}

	blockID := Keys.UnpackIrrBlocksKey(it.Item().Key)
	ref = bstream.NewBlockRefFromID(bstream.BlockRefFromID(blockID))

	if ref.Num() != uint64(dposIrrNum) {
		db.logger.Debug("get irr block by num: block num mismatch")
		return nil, kvdb.ErrNotFound
	}

	return ref, nil
}

func (db *DB) BlockIDAt(ctx context.Context, start time.Time) (id string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	it := db.blksReadStore.Scan(ctx, Keys.PackTimelinePrefix(true, start), Keys.EndOfTimelineIndex(true), 1)
	found := it.Next()
	if err := it.Err(); err != nil {
		return "", err
	}
	if !found {
		return "", kvdb.ErrNotFound
	}

	blockTime, blockID := Keys.UnpackTimelineKey(true, it.Item().Key)
	if start.Equal(blockTime) {
		return blockID, nil
	}
	return "", kvdb.ErrNotFound
}

func (db *DB) BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundTime time.Time, err error) {
	return db.blockIDAround(ctx, true, start, inclusive)
}

func (db *DB) BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundTime time.Time, err error) {
	return db.blockIDAround(ctx, false, start, inclusive)
}

func (db *DB) blockIDAround(ctx context.Context, fwd bool, start time.Time, inclusive bool) (id string, foundTime time.Time, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	it := db.blksReadStore.Scan(ctx, Keys.PackTimelinePrefix(fwd, start), Keys.EndOfTimelineIndex(fwd), 4) // supports 3 blocks at the *same* timestamp, should be pretty rare..

	for it.Next() {
		foundTime, id = Keys.UnpackTimelineKey(fwd, it.Item().Key)
		if !inclusive && foundTime.Equal(start) {
			continue
		}
		return
	}
	if err = it.Err(); err != nil {
		return
	}

	err = kvdb.ErrNotFound
	return
}

func (db *DB) ListBlocks(ctx context.Context, highBlockNum uint32, limit int) (out []*pbcodec.BlockWithRefs, err error) {
	db.logger.Debug("list blocks", zap.Uint32("high_block_num", highBlockNum), zap.Int("limit", limit))
	it := db.blksReadStore.Scan(ctx, Keys.PackBlockNumPrefix(highBlockNum), Keys.EndOfBlocksTable(), limit)
	for it.Next() {
		blockRow := &pbtrxdb.BlockRow{}
		db.dec.MustInto(it.Item().Value, blockRow)
		blk, err := db.blockRowToBlockWithRef(ctx, blockRow)
		if err != nil {
			return nil, fmt.Errorf("block with ref: %w", err)
		}
		out = append(out, blk)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	return
}

func (db *DB) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) (out []*pbcodec.BlockWithRefs, err error) {
	highBlockNum := blockNum + spread
	lowBlockNum := blockNum - (spread + 1)
	db.logger.Debug("list sibling blocks", zap.Uint32("high_block_num", highBlockNum), zap.Uint32("low_block_num", lowBlockNum))
	it := db.blksReadStore.Scan(ctx, Keys.PackBlockNumPrefix(highBlockNum), Keys.PackBlockNumPrefix(lowBlockNum), 0)
	for it.Next() {
		blockRow := &pbtrxdb.BlockRow{}
		db.dec.MustInto(it.Item().Value, blockRow)
		blk, err := db.blockRowToBlockWithRef(ctx, blockRow)
		if err != nil {
			return nil, fmt.Errorf("block with ref: %w", err)
		}
		out = append(out, blk)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	return
}

func (db *DB) GetAccount(ctx context.Context, accountName string) (*pbcodec.AccountCreationRef, error) {
	value, err := db.blksReadStore.Get(ctx, Keys.PackAccountKey(accountName))

	if err == store.ErrNotFound {
		return nil, kvdb.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("unable to get account: %w", err)
	}

	acctRow := &pbtrxdb.AccountRow{}
	db.dec.MustInto(value, acctRow)
	return &pbcodec.AccountCreationRef{
		Account:       acctRow.Name,
		Creator:       acctRow.Creator,
		BlockNum:      uint64(eos.BlockNum(acctRow.BlockId)),
		BlockId:       acctRow.BlockId,
		BlockTime:     acctRow.BlockTime,
		TransactionId: acctRow.TrxId,
	}, nil
}

func (db *DB) ListAccountNames(ctx context.Context, concurrentReadCount uint32) (out []string, err error) {
	if concurrentReadCount == 0 {
		return nil, fmt.Errorf("invalid concurrent read")
	}

	it := db.blksReadStore.Scan(ctx, Keys.StartOfAccountTable(), Keys.EndOfAccountTable(), 0)
	for it.Next() {
		acctRow := &pbtrxdb.AccountRow{}
		db.dec.MustInto(it.Item().Value, acctRow)
		out = append(out, acctRow.Name)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return
}
