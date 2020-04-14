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
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/eosdb/mdl"
	"github.com/dfuse-io/kvdb"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

var latestCellOnly = bigtable.LatestNFilter(1)
var latestCellFilter = bigtable.RowFilter(latestCellOnly)

func (b *EOSDatabase) GetBlock(ctx context.Context, blockID string) (*pbdeos.BlockWithRefs, error) {
	ctx, span := b.StartSpan(ctx, "get block", trace.StringAttribute("block_id", blockID))
	defer span.End()

	rowRange := bigtable.PrefixRange(Keys.Block(blockID))
	responses, err := b.Blocks.ReadRows(ctx, rowRange, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("get block: %s", err)
	}

	if len(responses) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return responses[0], nil
}

func (b *EOSDatabase) GetLastWrittenBlockID(ctx context.Context) (out string, err error) {
	ctx, span := b.StartSpan(ctx, "get last written block id")
	defer span.End()

	col := bigtable.ColumnRangeFilter("meta", "written", "written;")
	val := bigtable.ValueRangeFilter([]byte{0x01}, []byte{0x02})
	filter := bigtable.RowFilter(bigtable.ChainFilters(col, val))
	blocks, err := b.Blocks.ReadRows(ctx, bigtable.InfiniteRange(""), latestCellFilter, filter, bigtable.LimitRows(1))
	if err != nil {
		return "", fmt.Errorf("get block by num: %s", err)
	}

	if len(blocks) == 0 {
		return "", kvdb.ErrNotFound
	}

	return blocks[0].Id, nil
}

func (b *EOSDatabase) GetBlockByNum(ctx context.Context, blockNum uint32) ([]*pbdeos.BlockWithRefs, error) {
	ctx, span := b.StartSpan(ctx, "get block by num", trace.Int64Attribute("block_num", int64(blockNum)))
	defer span.End()

	key := kvdb.HexRevBlockNum(blockNum)
	rowRange := bigtable.PrefixRange(key)

	responses, err := b.Blocks.ReadRows(ctx, rowRange, latestCellFilter)
	if err != nil {
		return responses, fmt.Errorf("get block by num: %s", err)
	}

	if len(responses) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return responses, nil
}

// GetClosestIrreversibleIDAtBlockNum retrieves the CLOSEST
// irreversible block from that block num, INCLUSIVELY.
//
// WARN: a previous version of this function was EXCLUSIVE
// (GetIrreversibleIDAtBlockNum), make sure the caller does
// `blockNum-1` if it wants to keep that behavior.
func (b *EOSDatabase) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, blockNum uint32) (bstream.BlockRef, error) {
	ctx, span := b.StartSpan(ctx, "get irreversible id at block num", trace.Int64Attribute("block_num", int64(blockNum)))
	defer span.End()

	switch blockNum {
	case 2:
		return bstream.NewBlockRefFromID("0000000100000000000000000000000000000000000000000000000000000000"), nil // should work with forkdb exceptions
	case 1:
		return bstream.NewBlockRefFromID("0000000000000000000000000000000000000000000000000000000000000000"), nil
	case 0:
		return nil, fmt.Errorf("cannot get irreversible block before or at 0")
	}

	val := bigtable.ValueRangeFilter([]byte{0x01}, []byte{0x02})
	col1 := bigtable.ColumnRangeFilter("meta", "irreversible", "irreversible;")
	filter := bigtable.ChainFilters(col1, val)
	blocks, err := b.Blocks.ReadIrrCell(ctx, bigtable.InfiniteRange(kvdb.HexRevBlockNum(blockNum)), latestCellFilter, bigtable.RowFilter(filter), bigtable.LimitRows(1))
	if err != nil {
		return nil, fmt.Errorf("get block by num: %s", err)
	}

	if len(blocks) == 0 {
		return nil, kvdb.ErrNotFound
	}
	if len(blocks) > 1 {
		return nil, errors.New("more than one block returned")
	}

	return bstream.NewBlockRefFromID(bstream.BlockRefFromID(blocks[0].Id)), nil
}

// FIXME: Put that in CONSUMING code. It uses only public APIs from this. Belongs elsewhere.
func (b *EOSDatabase) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (bstream.BlockRef, error) {
	ctx, span := b.StartSpan(ctx, "get irreversible id at block id", trace.StringAttribute("block_id", ID))
	defer span.End()

	// FIXME: This should only retrieve the block column we care about and not everything!
	blk, err := b.GetBlock(ctx, ID)
	if err != nil {
		return nil, err
	}

	responses, err := b.GetBlockByNum(ctx, blk.Block.DposIrreversibleBlocknum)
	if err != nil {
		return nil, err
	}

	for _, b := range responses {
		if b.Irreversible {
			return bstream.NewBlockRefFromID(bstream.BlockRefFromID(b.Id)), nil
		}
	}

	return nil, kvdb.ErrNotFound
}

// FIXME: this belongs to the caller, not to the KVDB interface.  This is client code, using
// `GetBlock` and `GetTransactionTraces`, outside the app context.  The `mdl.TransactionList`
// belongs to `eosws` and not here.
func (b *EOSDatabase) ListTransactionsForBlockID(
	ctx context.Context,
	blockID string,
	startKey string,
	limit int,
) (out *mdl.TransactionList, err error) {
	ctx, span := b.StartSpan(ctx, "list transactions for block id",
		trace.StringAttribute("block_id", blockID),
		trace.StringAttribute("start_key", startKey),
		trace.Int64Attribute("limit", int64(limit)),
	)
	defer span.End()

	if limit < 1 {
		return &mdl.TransactionList{
			NextCursor: startKey,
		}, nil
	}

	block, err := b.GetBlock(ctx, blockID)
	if err == kvdb.ErrNotFound {
		return nil, kvdb.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	// TODO: use block.Irreversible to distinguish behavior of the stitcher..
	// context here is: if you're asking for a block PAST irreversibility barrier, but that
	// block is STALE or reversible, we'd want a chainDiscriminator that is able
	// to navigate the block_id -> header.previous_id FROM Bigtable instead of from a
	// ForkDB, this way we could make sure the dtrx creation was from the same chain, and
	// ignore the creations that were from the irreversible chain.
	//
	// That chain discriminator could:
	//
	// * query the `meta:header` columns from the `blockID` passed to
	//   `ListTransactionsForBlockID`, until parameter until a *block
	//   num* has NO stale block for that number: means we're sure
	//   there were no forks .. means the fork we're asking to see has
	//   found its junction point.
	//
	// * we can then navigate that little chain like we would in a
	//   ForkDB, checking if those blocks include the `blockID` passed
	//   to the chainDiscriminator.
	//
	// * We can also, in the chainDiscriminator, validate right away
	//   if the passed `blockID` was indeed irreversible, avoiding the
	//   first query altogether.
	//
	// * this situation is only true for the duration of a fork, and
	//   the uncertainty (when requesting a stale block) only goes back
	//   to the irreversible junction point... so even, for those tiny
	//   cases where someone explicitly calls the list of transactions
	//   of a stale block, we would have a clear view.

	var startTrxIndex uint16
	if startKey != "" {
		startTrxIndex, err = kvdb.FromHexUint16(startKey)
		if err != nil {
			return nil, err
		}
	}

	var trxIDs []string
	var nextTrxIndex uint16
	for trxIndex, trxIDBytes := range block.TransactionTraceRefs.Hashes {
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

	trxList, err := b.GetTransactionEventsBatch(ctx, trxIDs)
	if err != nil {
		return nil, err
	}

	upperBound := limit
	if len(trxList) < upperBound {
		upperBound = len(trxList)
	}

	return &mdl.TransactionList{
		NextCursor:   kvdb.HexUint16(nextTrxIndex),
		Transactions: trxList[0:upperBound],
	}, nil
}

func (b *EOSDatabase) ListBlocks(ctx context.Context, startBlockNum uint32, limit int) ([]*pbdeos.BlockWithRefs, error) {
	ctx, span := b.StartSpan(ctx, "list blocks",
		trace.Int64Attribute("start_block_num", int64(startBlockNum)),
		trace.Int64Attribute("limit", int64(limit)),
	)
	defer span.End()

	if limit < 1 {
		return nil, nil
	}

	revBlockNum := kvdb.HexRevBlockNum(startBlockNum)
	rowRange := bigtable.InfiniteRange(revBlockNum)

	responses, err := b.Blocks.ReadRows(ctx, rowRange, latestCellFilter, bigtable.LimitRows(int64(limit)))
	if err != nil {
		return nil, fmt.Errorf("list blocks: %s", err)
	}

	return responses, nil
}

func (b *EOSDatabase) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbdeos.BlockWithRefs, error) {
	ctx, span := b.StartSpan(ctx, "list siblings blocks",
		trace.Int64Attribute("block_num", int64(blockNum)),
		trace.Int64Attribute("spread", int64(spread)),
	)
	defer span.End()

	startBlockNum := blockNum + spread
	endBlockNum := blockNum - (spread + 1)
	if spread >= blockNum {
		endBlockNum = blockNum - 1
	}

	startKey := kvdb.HexRevBlockNum(startBlockNum)
	endKey := kvdb.HexRevBlockNum(endBlockNum)
	rowRange := bigtable.NewRange(startKey, endKey)

	responses, err := b.Blocks.ReadRows(ctx, rowRange, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("list sibling blocks: %s", err)
	}

	return responses, nil
}

func (b *EOSDatabase) ListBlocksRange(ctx context.Context, blockNumStart uint32, blockNumEnd uint32) ([]*pbdeos.BlockWithRefs, error) {
	ctx, span := b.StartSpan(ctx, "list blocks range",
		trace.Int64Attribute("block_num_start", int64(blockNumStart)),
		trace.Int64Attribute("block_num_end", int64(blockNumEnd)),
	)
	defer span.End()

	startKey := kvdb.HexRevBlockNum(blockNumStart)
	endKey := kvdb.HexRevBlockNum(blockNumEnd)
	rowRange := bigtable.NewRange(startKey, endKey)

	responses, err := b.Blocks.ReadRows(ctx, rowRange, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("list blocks range: %s", err)
	}

	return responses, nil
}

func (b *EOSDatabase) ListAccountNames(ctx context.Context, concurrentReadCount uint32) ([]string, error) {
	if concurrentReadCount < 1 {
		return nil, fmt.Errorf("invalid concurrent read")
	}
	ctx, span := b.StartSpan(ctx, "list account names", trace.Int64Attribute("concurrent_read_count", int64(concurrentReadCount)))
	defer span.End()

	var accountNames []string
	lock := sync.Mutex{}
	addAccountName := func(accountName string) {
		lock.Lock()
		defer lock.Unlock()

		accountNames = append(accountNames, accountName)
	}

	err := b.Accounts.ParallelStreamRows(ctx, createAccountRowSets(concurrentReadCount), concurrentReadCount, func(response *AccountResponse) bool {
		addAccountName(string(response.Name))
		return true
	}, bigtable.RowFilter(bigtable.CellsPerRowLimitFilter(1)))

	if err != nil {
		return nil, fmt.Errorf("list account names: %s", err)
	}

	zlog.Debug("completed assembling account names list", zap.Int("count", len(accountNames)))
	return accountNames, nil
}

func createAccountRowSets(concurrentReadCount uint32) []bigtable.RowSet {
	step := math.MaxUint64 / uint64(concurrentReadCount)
	startPrefix := "a:"
	var endPrefix string

	rowRanges := make([]bigtable.RowSet, concurrentReadCount)
	for i := uint64(0); i < uint64(concurrentReadCount)-1; i++ {
		endPrefix = Keys.Account((i + 1) * step)
		rowRanges[i] = bigtable.NewRange(startPrefix, endPrefix)

		startPrefix = endPrefix
	}

	// FIXME: Find a way to get up to last possible keys of `a:` set without copying the `prefixSuccessor` method from bigtable
	//        Hard-coded for now.
	rowRanges[concurrentReadCount-1] = bigtable.NewRange(startPrefix, "a;")

	return rowRanges
}

func (b *EOSDatabase) GetAccount(ctx context.Context, accountName string) (*pbdeos.AccountCreationRef, error) {
	ctx, span := b.StartSpan(ctx, "get account", trace.StringAttribute("account_name", accountName))
	defer span.End()

	name, err := eos.StringToName(accountName)
	if err != nil {
		return nil, fmt.Errorf("string to name: %s", err)
	}
	key := Keys.Account(name)

	row, err := b.Accounts.ReadRow(ctx, key, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("read row: %s", err)
	}

	if len(row) == 0 {
		return nil, kvdb.ErrNotFound
	}

	parsed, err := b.Accounts.parseRowAs(row)
	if err != nil {
		return nil, err
	}

	out := &pbdeos.AccountCreationRef{
		Account: string(parsed.Name),
		Creator: string(parsed.CreatorName),
	}
	if parsed.Creator != nil {
		blockTime, _ := ptypes.TimestampProto(parsed.Creator.BlockTime)
		out.BlockTime = blockTime
		out.BlockId = parsed.Creator.BlockID
		out.BlockNum = uint64(parsed.Creator.BlockNum)
		out.TransactionId = parsed.Creator.TrxID
	}
	return out, nil
}

// FIXME: delete this
// // ListMostRecentTransactions has a cursor that looks like: `block_id`:`trx_index:uint16`
// func (b *EOSDatabase) ListMostRecentTransactions(ctx context.Context, startKey string, limit int, chainDiscriminator eosdb.ChainDiscriminator) (*mdl.TransactionList, error) {
// 	ctx, span := b.StartSpan(ctx, "list most recent transactions",
// 		trace.StringAttribute("start_key", startKey),
// 		trace.Int64Attribute("limit", int64(limit)),
// 	)
// 	defer span.End()

// 	// MOVED TO `eosws`
// }

func (b *EOSDatabase) GetTransactionTraces(ctx context.Context, idPrefix string) (out []*pbdeos.TransactionEvent, err error) {
	events, err := b.GetTransactionEvents(ctx, idPrefix)
	if err != nil {
		return nil, err
	}

	for _, ev := range events {
		if _, ok := ev.Event.(*pbdeos.TransactionEvent_Execution); ok {
			out = append(out, ev)
		}
	}

	// OPTIM: later, we can avoid deserializing all the TransactionEvents based on the rows, in
	// order to fetch the Trace only, and faster.

	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}
	return out, nil
}

func (b *EOSDatabase) GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbdeos.TransactionEvent, error) {
	ctx, span := b.StartSpan(ctx, "get transaction", trace.StringAttribute("id_prefix", idPrefix))
	defer span.End()

	rowRange := bigtable.PrefixRange(idPrefix)
	events, err := b.Transactions.ReadEvents(ctx, rowRange, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %s", err)
	}

	if len(events) == 0 {
		return nil, kvdb.ErrNotFound
	}

	return events, nil
}

func (b *EOSDatabase) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbdeos.TransactionEvent, err error) {
	allEvents, err := b.GetTransactionEventsBatch(ctx, idPrefixes)
	if err != nil {
		return nil, err
	}

	out = make([][]*pbdeos.TransactionEvent, len(allEvents))
	for idx, events := range allEvents {
		for _, ev := range events {
			if _, ok := ev.Event.(*pbdeos.TransactionEvent_Execution); ok {
				out[idx] = append(out[idx], ev)
			}
		}
	}

	return
}

// GetTransactionEventsBatch retrieves all events for each transaction
// listed in `idPrefixes`.  It is the caller's responsibility to
// decide whether it wants Irreversible only, by using
// `pbdeos.MergeTransactionEvents()` for example.
func (b *EOSDatabase) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbdeos.TransactionEvent, error) {
	ctx, span := b.StartSpan(ctx, "get transaction row batch trace", trace.Int64Attribute("len_id_prefixes", int64(len(idPrefixes))))
	defer span.End()

	var ranges []bigtable.RowRange
	prefixToIndex := make(map[string]int)
	for idx, idPrefix := range idPrefixes {
		prefixToIndex[idPrefix] = idx
		ranges = append(ranges, bigtable.PrefixRange(idPrefix))
	}

	events, err := b.Transactions.ReadEvents(ctx, bigtable.RowRangeList(ranges), latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %s", err)
	}

	m := idToPrefix{}
	out := make([][]*pbdeos.TransactionEvent, len(idPrefixes))
	for _, ev := range events {
		prefix, err := m.prefix(idPrefixes, ev.Id)
		if err != nil {
			return nil, err
		}

		idx := prefixToIndex[prefix]
		out[idx] = append(out[idx], ev)
	}

	return out, nil
}

// FIXME: delete this, no one calls it anymore. The TransactionEvent took prcedence over that.
func (b *EOSDatabase) GetTransactionRow(ctx context.Context, idPrefix string) (*TransactionRow, error) {
	ctx, span := b.StartSpan(ctx, "get raw transaction trace", trace.StringAttribute("id_prefix", idPrefix))
	defer span.End()

	rowRange := bigtable.PrefixRange(idPrefix)
	rows, err := b.Transactions.ReadRows(ctx, rowRange, latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %s", err)
	}

	if len(rows) == 0 {
		return nil, kvdb.ErrNotFound
	}

	var rowInLonguestChain *TransactionRow
	var largestBlockNum uint64
	for _, row := range rows {
		if !row.Written {
			zlog.Debug("ignoring row that is not fully written yet", zap.String("row_key", row.Key))
			continue
		}

		if row.Irreversible {
			if row.TransactionTrace != nil && row.TransactionTrace.BlockNum > largestBlockNum {
				rowInLonguestChain = row
				largestBlockNum = row.TransactionTrace.BlockNum
			}
		}
	}

	if rowInLonguestChain != nil {
		return rowInLonguestChain, nil
	}

	return nil, kvdb.ErrNotFound
}

// GetTransactions returns aggregated TransactionResponse. It might
// return less transactions than what is specified by `ids` if some
// transactions are not in current chain (or irreversible).

// FIXME: delete this, it's used by `eosws`, but see how it can NOT use it anymore.
// This should be replaced by `GetTransactionEventsBatch` and Merged like the other ones.
// Hopefully there's no difference (check in `eosws`).
func (b *EOSDatabase) GetTransactions(ctx context.Context, ids []string, chainDiscriminator eosdb.ChainDiscriminator) (out []*pbdeos.TransactionLifecycle, err error) {
	ctx, span := b.StartSpan(ctx, "get transactions", trace.Int64Attribute("transaction_count", int64(len(ids))))
	defer span.End()

	prefixLength := 64
	var ranges []bigtable.RowRange
	for _, id := range ids {
		ranges = append(ranges, bigtable.PrefixRange(id))

		if len(id) < prefixLength {
			prefixLength = len(id)
		}
	}

	rows, err := b.Transactions.ReadRows(ctx, bigtable.RowRangeList(ranges), latestCellFilter)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %s", err)
	}

	groupedTrxIDs := make(map[string][]*TransactionRow)
	var trxIDs []string
	for _, row := range rows {
		if !row.Written {
			zlog.Debug("ignoring row that is not fully written yet", zap.String("row_key", row.Key))
			continue
		}

		trxID, _, err := Keys.ReadTransaction(row.Key)
		if err != nil {
			return nil, err
		}
		trxPrefix := trxID[:prefixLength]

		if _, found := groupedTrxIDs[trxPrefix]; !found {
			trxIDs = append(trxIDs, trxID)
		}

		groupedTrxIDs[trxPrefix] = append(groupedTrxIDs[trxPrefix], row)
	}

	for _, trxID := range ids {
		groupedIDs, found := groupedTrxIDs[trxID[:prefixLength]]
		if !found {
			continue
		}
		response, err := b.Transactions.stitchTransaction(groupedIDs, chainDiscriminator)
		if err != nil {
			return nil, err
		}

		if response != nil {
			out = append(out, response)
		}
	}

	return
}

func (b *EOSDatabase) BlockIDAt(ctx context.Context, blockTime time.Time) (id string, err error) {
	ctx, span := b.StartSpan(ctx, "find blockID at or before a timestamp")
	trace.Int64Attribute("blockTime", blockTime.UnixNano())
	defer span.End()

	prefix := Keys.TimelineBlockReverse(blockTime, "")
	keys, err := b.Timeline.ReadRows(ctx, bigtable.PrefixRange(prefix), latestCellFilter, bigtable.LimitRows(1))
	if err != nil {
		return
	}
	if len(keys) >= 1 {
		_, id, err = Keys.ReadTimelineBlockReverse(keys[0])
		return id, err
	}
	return "", kvdb.ErrNotFound
}

func (b *EOSDatabase) BlockIDBefore(ctx context.Context, blockTime time.Time, inclusive bool) (id string, foundTime time.Time, err error) {
	ctx, span := b.StartSpan(ctx, "find blockID at or before a timestamp")
	trace.Int64Attribute("blockTime", blockTime.UnixNano())
	defer span.End()

	nextPrefix := Keys.TimelineBlockReverse(blockTime, "")
	keys, err := b.Timeline.ReadRows(ctx, bigtable.InfiniteRange(nextPrefix), latestCellFilter, bigtable.LimitRows(2))
	if err != nil {
		zlog.Info("time prefix", zap.Error(err))
		return "", time.Time{}, kvdb.ErrNotFound
	}

	err = kvdb.ErrNotFound
	for _, key := range keys {
		foundTime, id, err = Keys.ReadTimelineBlockReverse(key)
		if err != nil {
			zlog.Info("time prefix", zap.Error(err))
			return "", time.Time{}, kvdb.ErrNotFound
		}

		if foundTime.Before(blockTime) {
			return
		}

		if inclusive && foundTime.Equal(blockTime) {
			return
		}
	}
	return "", time.Time{}, kvdb.ErrNotFound
}

func (b *EOSDatabase) BlockIDAfter(ctx context.Context, blockTime time.Time, inclusive bool) (id string, foundTime time.Time, err error) {
	ctx, span := b.StartSpan(ctx, "find blockID at or after a timestamp")
	trace.Int64Attribute("blockTime", blockTime.UnixNano())
	defer span.End()

	nextPrefix := Keys.TimelineBlockForward(blockTime, "")

	keys, err := b.Timeline.ReadRows(ctx, bigtable.InfiniteRange(nextPrefix), latestCellFilter, bigtable.LimitRows(2)) // expect no more with same timestamp
	if err != nil {
		zlog.Info("time prefix", zap.Error(err))
		return "", time.Time{}, kvdb.ErrNotFound
	}

	err = kvdb.ErrNotFound
	for _, key := range keys {
		foundTime, id, err = Keys.ReadTimelineBlockForward(key)
		if err != nil {
			zlog.Info("time prefix", zap.Error(err))
			return "", time.Time{}, kvdb.ErrNotFound
		}
		if inclusive || foundTime.After(blockTime) {
			break
		}
	}
	return
}
