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

package eosws

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/streamingfast/bstream"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	"github.com/streamingfast/logging"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/kvdb"
	"github.com/streamingfast/opaque"
	"go.uber.org/zap"
)

type DB interface {
	trxdb.DBReader

	GetTransactionWithExpectedBlockID(ctx context.Context, id string, expectedBlockID string) (*pbcodec.TransactionLifecycle, error)
	GetTransaction(ctx context.Context, id string) (*pbcodec.TransactionLifecycle, error)
	GetTransactions(ctx context.Context, ids []string) ([]*pbcodec.TransactionLifecycle, error)
	ListTransactionsForBlockID(ctx context.Context, blockId string, startKey string, limit int) (*mdl.TransactionList, error)
	ListMostRecentTransactions(ctx context.Context, startKey string, limit int) (*mdl.TransactionList, error)
}

// TRXDB

type TRXDB struct {
	trxdb.DBReader

	chainDiscriminator func(blockID string) bool
}

func NewTRXDB(dbReader trxdb.DBReader) *TRXDB {
	return &TRXDB{
		DBReader: dbReader,
		// TODO: implement real discriminator
		chainDiscriminator: func(blockID string) bool {
			return true
		},
	}
}

func (db *TRXDB) GetTransactionWithExpectedBlockID(ctx context.Context, id string, expectedBlockID string) (out *pbcodec.TransactionLifecycle, err error) {
	evs, err := db.GetTransactionEvents(ctx, id)
	if err != nil && err != kvdb.ErrNotFound {
		return nil, err
	}
	if len(evs) == 0 {
		return nil, DBTrxNotFoundError(ctx, id)
	}
	seenBlockID := false
	for _, ev := range evs {
		if ev.BlockId == expectedBlockID {
			seenBlockID = true
			break
		}
	}
	if !seenBlockID {
		return nil, DBTrxNotFoundError(ctx, id)
	}

	out = pbcodec.MergeTransactionEvents(evs, db.chainDiscriminator)
	return
}

func (db *TRXDB) GetTransaction(ctx context.Context, id string) (out *pbcodec.TransactionLifecycle, err error) {
	evs, err := db.GetTransactionEvents(ctx, id)
	if err != nil && err != kvdb.ErrNotFound {
		return nil, err
	}
	if len(evs) == 0 {
		return nil, DBTrxNotFoundError(ctx, id)
	}

	out = pbcodec.MergeTransactionEvents(evs, db.chainDiscriminator)

	return
}

func (db *TRXDB) GetTransactions(ctx context.Context, ids []string) (out []*pbcodec.TransactionLifecycle, err error) {
	evs, err := db.GetTransactionEventsBatch(ctx, ids)
	if err != nil {
		return nil, err
	}

	for idx, ev := range evs {
		trxID := ids[idx]
		if len(ev) == 0 {
			return nil, DBTrxNotFoundError(ctx, trxID)
		}

		out = append(out, pbcodec.MergeTransactionEvents(ev, db.chainDiscriminator))
	}

	return
}

func (db *TRXDB) ListTransactionsForBlockID(ctx context.Context, blockID string, startKey string, limit int) (*mdl.TransactionList, error) {
	if limit < 1 {
		return &mdl.TransactionList{
			Cursor: opaqueCursor(startKey),
		}, nil
	}

	block, err := db.GetBlock(ctx, blockID)
	if err != nil {
		return nil, err
	}

	var startTrxIndex uint16
	if startKey != "" {
		startTrxIndex, err = kvdb.FromHexUint16(startKey)
		if err != nil {
			return nil, err
		}
	}

	var trxIDs []string
	var nextTrxIndex uint16

	allRefsHashes := append(block.ImplicitTransactionRefs.Hashes, block.TransactionTraceRefs.Hashes...)

	for trxIndex, trxIDBytes := range allRefsHashes {
		trxID := hex.EncodeToString(trxIDBytes)
		if uint16(trxIndex) < startTrxIndex {
			continue
		}

		nextTrxIndex = uint16(trxIndex)

		// We add 1 to the limit to have more ids than requested since when we will
		// actually retrieved the trx down below, we account for the `onblock` trx
		// that will be left out (if in the ids list).
		if len(trxIDs) >= limit+1 {
			break
		}

		trxIDs = append(trxIDs, trxID)
	}

	trxList, err := db.GetTransactionEventsBatch(ctx, trxIDs)
	if err != nil {
		return nil, err
	}

	var lifecycles []*v1.TransactionLifecycle
	for _, evs := range trxList {
		if len(evs) == 0 {
			return nil, fmt.Errorf("transactions list for block ID: a transaction was not found")
		}
		lc, err := mdl.ToV1TransactionLifecycle(pbcodec.MergeTransactionEvents(evs, db.chainDiscriminator))
		if err != nil {
			return nil, fmt.Errorf("transactions list for block ID: %w", err)
		}
		lifecycles = append(lifecycles, lc)
	}

	upperBound := limit
	if len(lifecycles) < upperBound {
		upperBound = len(lifecycles)
	}

	return &mdl.TransactionList{
		Cursor:       opaqueCursor(kvdb.HexUint16(nextTrxIndex)),
		Transactions: lifecycles[0:upperBound],
	}, nil
}

func (db *TRXDB) ListMostRecentTransactions(ctx context.Context, startKey string, limit int) (*mdl.TransactionList, error) {
	if limit < 1 {
		return &mdl.TransactionList{
			Cursor: opaqueCursor(startKey),
		}, nil
	}

	fromCursorKey := func(startKey string) (blockID string, trxIndex uint16, err error) {
		chunks := strings.Split(startKey, ":")
		if len(chunks) != 2 {
			err = fmt.Errorf("invalid cursor key %q, expected two chunks separated by ':'", startKey)
			return
		}

		blockID = chunks[0]

		rawTrxIndex, err := strconv.ParseUint(chunks[1], 16, 16)
		if err != nil {
			err = fmt.Errorf("parsing transaction index %q: %s", chunks[1], err)
		}

		trxIndex = uint16(rawTrxIndex)

		return
	}

	//var nextPrefix string
	var nextBlockNum uint32
	nextBlockNum = math.MaxUint32
	var startBlockID string
	var startTrxIndex uint16
	if startKey != "" {
		blockID, trxIndex, err := fromCursorKey(startKey)
		if err != nil {
			return nil, err
		}
		nextBlockNum = eos.BlockNum(blockID)
		startBlockID = blockID
		startTrxIndex = trxIndex
	}

	// TODO: fetch ListBlocks(nextPrefix, limit) BlockWithRefs
	// on a les
	// On fetch seulement la colonne `trxs:executed-ids`
	// De ceux-là, on prends les `limit` premiers, et on forge le `NextCursor` à partir soit du prochain index, ou du prochain

	var list [][]*pbcodec.TransactionEvent
	type seenData struct {
		transactionIndex uint16
		blockID          string
	}
	seenTrx := make(map[string]*seenData)
	seenBlock := make(map[string]bool)

	for {
		if len(list) >= limit+1 /* +1 for the next cursor */ {
			break
		}

		const blockCount = 5
		blks, err := db.ListBlocks(ctx, nextBlockNum, blockCount) // for a max of 5 forked blocks
		if err != nil {
			return nil, fmt.Errorf("list blocks: %s", err)
		}

		for _, blk := range blks {
			if len(list) > limit+1 {
				break
			}

			if seenBlock[blk.Id] {
				break
			}

			seenBlock[blk.Id] = true

			if !db.chainDiscriminator(blk.Id) {
				continue
			}

			// Accumulate transactions once the transaction
			var fetchTransactions []string
			allRefsHashes := append(blk.ImplicitTransactionRefs.Hashes, blk.TransactionTraceRefs.Hashes...)

			for trxIndex := len(allRefsHashes) - 1; trxIndex >= 0; trxIndex-- {
				trxIDBytes := allRefsHashes[trxIndex]
				trxID := hex.EncodeToString(trxIDBytes)

				if blk.Id == startBlockID && trxIndex > int(startTrxIndex) /* fetch one more, to know the next cursor */ {
					continue
				}

				if _, found := seenTrx[trxID]; !found {
					// save here the  `blockID` + `trxIndex
					fetchTransactions = append(fetchTransactions, trxID)
					seenTrx[trxID] = &seenData{
						transactionIndex: uint16(trxIndex),
						blockID:          blk.Id,
					}
				}
			}

			// FIXME: optimize this, as we don't need to get ALL the transactions if we know we're going to need only 25..
			responses, err := db.GetTransactionEventsBatch(ctx, fetchTransactions)
			if err != nil {
				return nil, err
			}

			for _, resp := range responses {
				if len(resp) != 0 {
					list = append(list, resp)
					//} else {return nil, fmt.Errorf("transaction not found: %q", fetchTransactions[idx]) // soft failing only
				}

			}

			nextBlockNum = eos.BlockNum(blk.Id) - 1
			//nextPrefix = kvdb.ReversedBlockID(kvdb.IncreaseBlockIDSuffix(blk.Id))
		}

		if len(blks) < blockCount {
			// reached tip of history
			break
		}

	}

	out := &mdl.TransactionList{}
	if len(list) > limit {
		trxRowAtBoundary := list[limit]

		seenData := seenTrx[trxRowAtBoundary[0].Id]
		if seenData == nil {
			return nil, fmt.Errorf("hmm, we haven't seen this but we added it? %s", trxRowAtBoundary[0].Id)
		}

		out.Cursor = opaqueCursor(seenData.blockID + ":" + kvdb.HexUint16(seenData.transactionIndex))
	}

	var keep [][]*pbcodec.TransactionEvent
	if len(list) < limit {
		keep = list
	} else {
		keep = list[:limit]
	}

	for _, events := range keep {
		lc, err := mdl.ToV1TransactionLifecycle(pbcodec.MergeTransactionEvents(events, db.chainDiscriminator))
		if err != nil {
			return nil, fmt.Errorf("most recent transactions: %w", err)
		}
		out.Transactions = append(out.Transactions, lc)
	}

	return out, nil
}

func (db *TRXDB) GetBlock(ctx context.Context, id string) (out *pbcodec.BlockWithRefs, err error) {
	out, err = db.DBReader.GetBlock(ctx, id)
	if err == eos.ErrNotFound {
		return nil, DBBlockNotFoundError(ctx, id)
	}

	return
}

func (db *TRXDB) GetBlockByNum(ctx context.Context, num uint32) (out []*pbcodec.BlockWithRefs, err error) {
	out, err = db.DBReader.GetBlockByNum(ctx, num)
	if err == kvdb.ErrNotFound {
		return nil, DBBlockNotFoundError(ctx, strconv.FormatUint(uint64(num), 10))
	}
	if err != nil {
		logging.Logger(ctx, zlog).Error("cannot get blocks by number", zap.Uint32("block_num", num), zap.Error(err))
	}

	return
}

func (db *TRXDB) GetAccount(ctx context.Context, name string) (account *pbcodec.AccountCreationRef, err error) {
	account, err = db.DBReader.GetAccount(ctx, name)
	if err == kvdb.ErrNotFound {
		return nil, DBAccountNotFoundError(ctx, name)
	}
	if err != nil {
		logging.Logger(ctx, zlog).Error("cannot get account", zap.String("account_name", name), zap.Error(err))
	}

	return
}

// MOCK MOCK

type MockDB struct {
	trxdb.TimelineExplorer
	trxdb.TransactionsReader
	path string
}

func NewMockDB(path string) *MockDB {
	return &MockDB{path: path}
}

func (db *MockDB) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (out bstream.BlockRef, err error) {
	return bstream.NewBlockRef("123", 123), nil
}

func (db *MockDB) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (out bstream.BlockRef, err error) {
	return bstream.NewBlockRef("123", 123), nil
}

func (db *MockDB) GetLastWrittenBlockID(ctx context.Context) (string, error) {
	return "0123456700000000000000000000000000000000000000000000000000000000", nil
}

func (db *MockDB) GetTransactionWithExpectedBlockID(ctx context.Context, id string, _ string) (out *pbcodec.TransactionLifecycle, err error) {
	return db.GetTransaction(ctx, id)
}

func (db *MockDB) GetTransaction(ctx context.Context, id string) (out *pbcodec.TransactionLifecycle, err error) {
	filename := filepath.Join(db.path, "transactions", fmt.Sprintf("%s.json", id))
	err = readFromFile(filename, out)
	return
}

func (db *MockDB) GetTransactions(ctx context.Context, ids []string) (out []*pbcodec.TransactionLifecycle, err error) {
	for _, id := range ids {
		filename := filepath.Join(db.path, "transactions", fmt.Sprintf("%s.json", id))
		var el *pbcodec.TransactionLifecycle
		err = readFromFile(filename, el)
		out = append(out, el)
	}
	return
}
func (db *MockDB) ListMostRecentTransactions(ctx context.Context, startKey string, limit int) (*mdl.TransactionList, error) {
	panic("Implement me!")
}

func (db *MockDB) ListTransactionsForBlockID(ctx context.Context, blockId string, startKey string, limit int) (*mdl.TransactionList, error) {
	panic("Implement me!")
}

func (db *MockDB) GetBlock(ctx context.Context, id string) (out *pbcodec.BlockWithRefs, err error) {
	filename := filepath.Join(db.path, "blocks", fmt.Sprintf("%s.json", id))
	err = readFromFile(filename, out)
	return
}

func (db *MockDB) ListBlocks(ctx context.Context, startBlockNum uint32, limit int) ([]*pbcodec.BlockWithRefs, error) {
	panic("implement me")
}

func (db *MockDB) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbcodec.BlockWithRefs, error) {
	panic("implement me")
}

func (db *MockDB) GetAccount(ctx context.Context, name string) (out *pbcodec.AccountCreationRef, err error) {
	return &pbcodec.AccountCreationRef{
		Account: "eoscanadacom",
		Creator: "bozo",
	}, nil
}

func (db *MockDB) ListAccountNames(ctx context.Context) (out []string, err error) {
	return []string{"eoscanadacom"}, nil
}

func (db *MockDB) GetBlockByNum(ctx context.Context, num uint32) (out []*pbcodec.BlockWithRefs, err error) {
	filename := filepath.Join(db.path, "blocks", fmt.Sprintf("%s.json", fmt.Sprintf("%d", num)))
	err = readFromFile(filename, out)
	return
}

func readFromFile(filename string, out interface{}) (err error) {
	buffer, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(buffer, &out)
	if err != nil {
		return fmt.Errorf("unmarshal: %s", err)
	}

	return
}

func opaqueCursor(key string) string {
	// The implementation always returns nil! So it's ok to panic here, if it changes and there is an error,
	// change the code throughtout.
	out, err := opaque.ToOpaque(key)
	if err != nil {
		panic(fmt.Errorf("unable to transform key %q to opaque cursor: %w", key, err))
	}

	return out
}
