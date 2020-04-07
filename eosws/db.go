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

	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/dfuse-io/bstream"
	"github.com/eoscanada/eos-go"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/kvdb/eosdb"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

type DB interface {
	eosdb.DBReader

	// GetLastWrittenBlockID(ctx context.Context) (out string, err error)
	// GetBlock(ctx context.Context, id string) (*mdl.BlockRow, error)
	//GetBlocksByNum(ctx context.Context, num uint32) ([]*mdl.BlockRow, error)
	//ListBlocks(ctx context.Context, startBlockNum uint32, limit int) ([]*mdl.BlockRow, error)
	//ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*mdl.BlockRow, error)
	GetTransaction(ctx context.Context, id string) (*pbdeos.TransactionLifecycle, error)
	GetTransactions(ctx context.Context, ids []string) ([]*pbdeos.TransactionLifecycle, error)
	ListTransactionsForBlockID(ctx context.Context, blockId string, startKey string, limit int) (*mdl.TransactionList, error)
	ListMostRecentTransactions(ctx context.Context, startKey string, limit int) (*mdl.TransactionList, error)
	// GetIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (blockID string, err error)
	// GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (blockID string, err error)
	// GetAccount(ctx context.Context, name string) (account *mdl.AccountResponse, err error)
	// ListAccountNames(ctx context.Context, concurrentReadCount uint32) (accountNames []eos.AccountName, err error)
}

// EOSDB

type EOSDB struct {
	eosdb.DBReader

	chainDiscriminator func(blockID string) bool
}

func NewEOSDB(dbReader eosdb.DBReader) *EOSDB {
	return &EOSDB{
		DBReader: dbReader,
		// TODO: implement real discriminator
		chainDiscriminator: func(blockID string) bool {
			return true
		},
	}
}

func (db *EOSDB) GetTransaction(ctx context.Context, id string) (out *pbdeos.TransactionLifecycle, err error) {
	evs, err := db.GetTransactionEvents(ctx, id)
	if err == kvdb.ErrNotFound {
		return nil, DBTrxNotFoundError(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	out = pbdeos.MergeTransactionEvents(evs, db.chainDiscriminator)

	return
}

func (db *EOSDB) GetTransactions(ctx context.Context, ids []string) (out []*pbdeos.TransactionLifecycle, err error) {
	evs, err := db.GetTransactionEventsBatch(ctx, ids)
	if err != nil {
		return nil, err
	}

	for idx, ev := range evs {
		trxID := ids[idx]
		if len(ev) == 0 {
			return nil, DBTrxNotFoundError(ctx, trxID)
		}

		out = append(out, pbdeos.MergeTransactionEvents(ev, db.chainDiscriminator))
	}

	return
}

func (db *EOSDB) ListTransactionsForBlockID(ctx context.Context, blockID string, startKey string, limit int) (*mdl.TransactionList, error) {
	if limit < 1 {
		return &mdl.TransactionList{
			Cursor: startKey,
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
		lc, err := mdl.ToV1TransactionLifecycle(pbdeos.MergeTransactionEvents(evs, db.chainDiscriminator))
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
		Cursor:       kvdb.HexUint16(nextTrxIndex),
		Transactions: lifecycles[0:upperBound],
	}, nil
}

func (db *EOSDB) ListMostRecentTransactions(ctx context.Context, startKey string, limit int) (*mdl.TransactionList, error) {
	if limit < 1 {
		return &mdl.TransactionList{
			Cursor: startKey,
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
		//nextPrefix = kvdb.ReversedBlockID(blockID)
		startBlockID = blockID
		startTrxIndex = trxIndex
	}

	// TODO: fetch ListBlocks(nextPrefix, limit) BlockWithRefs
	// on a les
	// On fetch seulement la colonne `trxs:executed-ids`
	// De ceux-là, on prends les `limit` premiers, et on forge le `NextCursor` à partir soit du prochain index, ou du prochain

	var list [][]*pbdeos.TransactionEvent
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

		// rowRange := bigtable.InfiniteRange(nextPrefix)
		// filter := bigtable.RowFilter(bigtable.FamilyFilter("trxs")) // we only want `trxs:executed-ids` and block ID (from the key)
		// // FIXME: THIS WILL FAIL BECAUSE WE DON'T HAVE A WRITTEN FIELD AGAIN
		// blocks, err := b.Blocks.ReadRows(ctx, rowRange, filter, latestCellFilter, bigtable.LimitRows(int64(limit)))
		// if err != nil {
		// 	return nil, fmt.Errorf("list transactions: %s", err)
		// }

		for _, blk := range blks {
			if len(list) > limit+1 {
				break
			}

			if seenBlock[blk.Id] {
				// in case `nextPrefix` was not properly increased
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

			for idx, resp := range responses {
				if len(resp) == 0 {
					return nil, fmt.Errorf("transaction not found: %q", fetchTransactions[idx])
				}
				list = append(list, resp)
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
		out.Cursor = seenData.blockID + ":" + kvdb.HexUint16(seenData.transactionIndex)
	}

	var keep [][]*pbdeos.TransactionEvent
	if len(list) < limit {
		keep = list
	} else {
		keep = list[:limit]
	}

	for _, events := range keep {
		lc, err := mdl.ToV1TransactionLifecycle(pbdeos.MergeTransactionEvents(events, db.chainDiscriminator))
		if err != nil {
			return nil, fmt.Errorf("most recent transactions: %w", err)
		}
		out.Transactions = append(out.Transactions, lc)
	}

	return out, nil
}

func (db *EOSDB) GetBlock(ctx context.Context, id string) (out *pbdeos.BlockWithRefs, err error) {
	out, err = db.DBReader.GetBlock(ctx, id)
	if err == eos.ErrNotFound {
		return nil, DBBlockNotFoundError(ctx, id)
	}

	return
}

func (db *EOSDB) GetBlockByNum(ctx context.Context, num uint32) (out []*pbdeos.BlockWithRefs, err error) {
	out, err = db.DBReader.GetBlockByNum(ctx, num)
	if err == kvdb.ErrNotFound {
		return nil, DBBlockNotFoundError(ctx, string(num))
	}
	if err != nil {
		logging.Logger(ctx, zlog).Error("cannot get blocks by number", zap.Uint32("block_num", num), zap.Error(err))
	}

	return
}

func (db *EOSDB) GetAccount(ctx context.Context, name string) (account *pbdeos.AccountCreationRef, err error) {
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
	eosdb.TimelineExplorer
	eosdb.TransactionsReader
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

func (db *MockDB) GetTransaction(ctx context.Context, id string) (out *pbdeos.TransactionLifecycle, err error) {
	filename := filepath.Join(db.path, "transactions", fmt.Sprintf("%s.json", id))
	err = readFromFile(filename, out)
	return
}

func (db *MockDB) GetTransactions(ctx context.Context, ids []string) (out []*pbdeos.TransactionLifecycle, err error) {
	for _, id := range ids {
		filename := filepath.Join(db.path, "transactions", fmt.Sprintf("%s.json", id))
		var el *pbdeos.TransactionLifecycle
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

func (db *MockDB) GetBlock(ctx context.Context, id string) (out *pbdeos.BlockWithRefs, err error) {
	filename := filepath.Join(db.path, "blocks", fmt.Sprintf("%s.json", id))
	err = readFromFile(filename, out)
	return
}

func (db *MockDB) ListBlocks(ctx context.Context, startBlockNum uint32, limit int) ([]*pbdeos.BlockWithRefs, error) {
	panic("implement me")
}

func (db *MockDB) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbdeos.BlockWithRefs, error) {
	panic("implement me")
}

func (db *MockDB) GetAccount(ctx context.Context, name string) (out *pbdeos.AccountCreationRef, err error) {
	return &pbdeos.AccountCreationRef{
		Account: "eoscanadacom",
		Creator: "bozo",
	}, nil
}

func (db *MockDB) ListAccountNames(ctx context.Context, concurrentReadCount uint32) (out []string, err error) {
	return []string{"eoscanadacom"}, nil
}

func (db *MockDB) GetBlockByNum(ctx context.Context, num uint32) (out []*pbdeos.BlockWithRefs, err error) {
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
