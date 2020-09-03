package accounthist

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

const (
	databaseTimeout = 10 * time.Minute
)

type Service struct {
	*shutter.Shutter

	shardNum             byte // 0 is live
	maxEntriesPerAccount uint64
	flushBlocksInterval  uint64
	blockFilter          func(blk *bstream.Block) error

	blocksStore dstore.Store
	kvStore     store.KVStore

	historySeqMap map[uint64]sequenceData
	source        bstream.Source

	rwCache *RWCache

	startBlockNum uint64
	stopBlockNum  uint64

	tracker *bstream.Tracker

	lastWrite        time.Time
	lastCheckpoint   *pbaccounthist.ShardCheckpoint
	lastBlockWritten uint64
}

func NewService(
	kvdb store.KVStore,
	blocksStore dstore.Store,
	blockFilter func(blk *bstream.Block) error,
	shardNum byte,
	maxEntriesPerAccount uint64,
	flushBlocksInterval uint64,
	startBlockNum uint64,
	stopBlockNum uint64,
	tracker *bstream.Tracker,
) *Service {
	return &Service{
		Shutter: shutter.New(),

		kvStore:              kvdb,
		blocksStore:          blocksStore,
		blockFilter:          blockFilter,
		shardNum:             shardNum,
		maxEntriesPerAccount: maxEntriesPerAccount,
		flushBlocksInterval:  flushBlocksInterval,
		startBlockNum:        startBlockNum,
		stopBlockNum:         stopBlockNum,
		tracker:              tracker,

		historySeqMap: make(map[uint64]sequenceData),
	}
}

func (ws *Service) Launch() {
	ws.source.OnTerminating(func(err error) {
		zlog.Info("block source shutted down, notifying service about its termination")
		ws.Shutdown(err)
	})

	ws.OnTerminating(func(_ error) {
		zlog.Info("service shutted down, shutting down block source")
		ws.source.Shutdown(nil)
	})

	ws.source.Run()
}

func (ws *Service) Shutdown(err error) {
	zlog.Info("service shutting down, about to terminate child services")
	ws.Shutter.Shutdown(err)
}

func (ws *Service) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	ctx := context.Background()

	block := blk.ToNative().(*pbcodec.Block)
	fObj := obj.(*forkable.ForkableObject)
	rawTraceMap := fObj.Obj.(map[uint64][]byte)
	isLastInStreak := fObj.StepIndex+1 == fObj.StepCount

	if ws.stopBlockNum != 0 && blk.Num() >= ws.stopBlockNum {
		zlog.Info("stop block num reached, flushing all writes",
			zap.Uint64("stop_block_num", ws.stopBlockNum),
			zap.Uint64("current_block_num", blk.Num()),
		)
		if err := ws.forceFlush(ctx, block); err != nil {
			ws.Shutdown(err)
			return fmt.Errorf("flushing when stopping: %w", err)
		}

		ws.Shutdown(nil)
		return nil
	}

	for _, trxTrace := range block.TransactionTraces() {
		if trxTrace.HasBeenReverted() {
			continue
		}

		actionMatcher := block.FilteringActionMatcher(trxTrace)
		for _, act := range trxTrace.ActionTraces {
			if !actionMatcher.Matched(act.ExecutionIndex) || act.Receipt == nil {
				continue
			}

			accts := map[string]bool{
				act.Receiver: true,
			}
			for _, v := range act.Action.Authorization {
				accts[v.Actor] = true
			}

			for acct := range accts {
				acctUint := eos.MustStringToName(acct)
				acctSeqData, err := ws.getSequenceData(ctx, acctUint)
				if err != nil {
					return fmt.Errorf("error while getting sequence data for account %v: %w", acct, err)
				}

				if acctSeqData.maxEntries == 0 {
					continue
				}

				// when shard 1 starts it will based the first seen action on values in shard 0. the last aciotn for an account
				// will always have a greater last global seq
				if act.Receipt.GlobalSequence <= acctSeqData.lastGlobalSeq {
					zlog.Debug("this block has already been processed for this account",
						zap.Stringer("block", blk),
						zap.String("account", acct),
					)
					continue
				}

				lastDeletedSeq, err := ws.deleteStaleRows(ctx, acctUint, acctSeqData)
				if err != nil {
					return err
				}

				acctSeqData.lastDeletedOrdinal = lastDeletedSeq
				rawTrace := rawTraceMap[act.Receipt.GlobalSequence]

				if err = ws.writeAction(ctx, acctUint, acctSeqData, act, rawTrace); err != nil {
					return fmt.Errorf("error while writing action to store: %w", err)
				}

				acctSeqData.nextOrdinal++
				acctSeqData.lastGlobalSeq = act.Receipt.GlobalSequence

				ws.updateHistorySeq(acctUint, acctSeqData)
			}
		}
	}

	if err := ws.writeLastProcessedBlock(ctx, block); err != nil {
		return fmt.Errorf("error while saving block checkpoint")
	}

	if err := ws.flush(ctx, block, isLastInStreak); err != nil {
		return fmt.Errorf("error while flushing: %w", err)
	}

	return nil
}

func (ws *Service) deleteStaleRows(ctx context.Context, account uint64, acctSeqData sequenceData) (lastDeletedSeq uint64, err error) {
	// The `nextOrdinal` represents the next ordinal to be written, it's not written yet, so the
	// actual last written value is the next ordinal minus one. The ordinal starts at 1, so it
	// represents the count directly (until the ordinal goes over the limit).
	lastWrittenOrdinal := acctSeqData.nextOrdinal - 1

	// If the last written ordinal is bigger than the max allowed entries for this account, adjust our sliding window by deleting anything below least recent ordinal
	if lastWrittenOrdinal > acctSeqData.maxEntries {
		// Don't forget, we are in a sliding window setup, so if last written was 12, assuming max entry of 5,
		// we have normally a window composed of ordinals [8, 9, 10, 11, 12], so anything from 7 and downwards should
		// be deleted.
		leastRecentOrdinal := lastWrittenOrdinal - acctSeqData.maxEntries

		// Assuming for this account that the last deleted ordinal is already higher or equal to our least
		// recent ordinal, it means there is nothing to do since everything below this last deleted ordinal
		// should already be gone.
		if acctSeqData.lastDeletedOrdinal >= leastRecentOrdinal {
			return acctSeqData.lastDeletedOrdinal, nil
		}

		// Let's assume our last deleted ordinal was 5, let's delete everything from 5 up and including 7
		zlog.Debug("deleting all actions between last deleted and now least recent ordinal", zap.Uint64("last_deleted_ordinal", acctSeqData.lastDeletedOrdinal), zap.Uint64("least_recent_ordinal", leastRecentOrdinal))
		for i := acctSeqData.lastDeletedOrdinal + 1; i <= leastRecentOrdinal; i++ {
			err := ws.deleteAction(ctx, account, i)
			if err != nil {
				return 0, fmt.Errorf("error while deleting action: %w", err)
			}
		}
		return leastRecentOrdinal, nil
	}

	return acctSeqData.lastDeletedOrdinal, nil
}

func (ws *Service) getSequenceData(ctx context.Context, account uint64) (out sequenceData, err error) {
	out, ok := ws.historySeqMap[account]
	if ok {
		return
	}

	out, err = ws.readSequenceData(ctx, account)
	if err == store.ErrNotFound {
		zlog.Info("account never seen before, initializing a new sequence data",
			zap.Stringer("account", EOSName(account)),
		)
		out.nextOrdinal = 1
		out.maxEntries = ws.maxEntriesPerAccount
	} else if err != nil {
		err = fmt.Errorf("error while fetching sequence data: %w", err)
		return
	}

	maxEntriesForAccount, err := ws.readMaxEntries(ctx, account)
	if err != nil {
		err = fmt.Errorf("error fetching max entries: %w", err)
		return
	}

	out.maxEntries = maxEntriesForAccount
	zlog.Debug("account sequence data setup",
		zap.Int("shard_num", int(ws.shardNum)),
		zap.Stringer("account", EOSName(account)),
		zap.Uint64("max_entries", maxEntriesForAccount),
		zap.Uint64("next_ordinal", out.nextOrdinal),
		zap.Uint64("last_global_sequence", out.lastGlobalSeq),
	)

	return
}

func (ws *Service) GetShardCheckpoint(ctx context.Context) (*pbaccounthist.ShardCheckpoint, error) {
	key := encodeLastProcessedBlockKey(ws.shardNum)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	val, err := ws.kvStore.Get(ctx, key)
	if err == store.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while last processed block: %w", err)
	}

	// Decode val as `pbaccounthist.ShardCheckpoint`
	out := &pbaccounthist.ShardCheckpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (ws *Service) updateHistorySeq(account uint64, seqData sequenceData) {
	ws.historySeqMap[account] = seqData
}

func (ws *Service) readMaxEntries(ctx context.Context, account uint64) (maxEntries uint64, err error) {
	nextShardNum := byte(0)
	var seenActions uint64
	for i := 0; i < 5; i++ {
		if nextShardNum >= ws.shardNum {
			// we'll stop writing only if FUTURE shards have covered our `maxEntriesPerAccount`.
			break
		}

		startKey := encodeActionKey(account, nextShardNum, math.MaxUint64)
		endKey := encodeActionKey(account+1, 0, 0)

		zlog.Debug("reading sequence data",
			zap.Stringer("account", EOSName(account)),
			zap.Int("shard_num", int(nextShardNum)),
			zap.Stringer("start_key", Key(startKey)),
			zap.Stringer("end_key", Key(endKey)),
		)

		ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
		defer cancel()

		it := ws.kvStore.Scan(ctx, startKey, endKey, 1)
		var rows int
		for it.Next() {
			rows++
			currentShardNum, historySeqNum := decodeActionKeySeqNum(it.Item().Key)
			seenActions += historySeqNum

			if seenActions >= ws.maxEntriesPerAccount {
				return 0, nil
			}

			nextShardNum = currentShardNum + 1
		}
		if it.Err() != nil {
			err = it.Err()
			return
		}
		if rows == 0 {
			break
		}
	}

	return ws.maxEntriesPerAccount - seenActions, nil
}

// readSequenceData returns sequenceData for a given account, for the current shard
func (ws *Service) readSequenceData(ctx context.Context, account uint64) (out sequenceData, err error) {

	// find the row with the biggest ordinal value for the current shard
	startKey := encodeActionKey(account, ws.shardNum, math.MaxUint64)
	endKey := store.Key(encodeActionPrefixKey(account)).PrefixNext()

	zlog.Info("reading last sequence data for current shard",
		zap.Stringer("account", EOSName(account)),
		zap.Int("current_shard_num", int(ws.shardNum)),
		zap.Stringer("start_key", Key(startKey)),
		zap.Stringer("end_key", Key(endKey)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := ws.kvStore.Scan(ctx, startKey, endKey, 1)
	hasNext := it.Next()
	if !hasNext && it.Err() != nil {
		return out, fmt.Errorf("scan last action: %w", it.Err())
	}

	if !hasNext {
		return out, store.ErrNotFound
	}

	newact := &pbaccounthist.ActionRow{}
	if err = proto.Unmarshal(it.Item().Value, newact); err != nil {
		return
	}

	_, out.nextOrdinal = decodeActionKeySeqNum(it.Item().Key)
	out.nextOrdinal++
	out.lastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
	out.lastDeletedOrdinal = newact.LastDeletedSeq

	return
}

func (ws *Service) writeAction(ctx context.Context, account uint64, acctSeqData sequenceData, actionTrace *pbcodec.ActionTrace, rawTrace []byte) error {
	key := encodeActionKey(account, ws.shardNum, acctSeqData.nextOrdinal)

	zlog.Debug("writing action", zap.Stringer("account", EOSName(account)), zap.Stringer("key", Key(key)))

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	if acctSeqData.lastDeletedOrdinal != 0 {
		// this is will append the protobuf pbaccounthist.ActionRowAppend to the protobuf pbaccounthist.ActionRow, since
		// both struct have the field last_deleted_seq with the same index (3), when unmarshalling the row
		// into an pbaccounthist.ActionRow the value of `last_deleted_seq` in the appended pbaccounthist.ActionRowAppend will
		// override the value defined in the pbaccounthist.ActionRow struct
		appendSeq := &pbaccounthist.ActionRowAppend{LastDeletedSeq: acctSeqData.lastDeletedOrdinal}
		encodedAppendSeq, _ := proto.Marshal(appendSeq)
		rawTrace = append(rawTrace, encodedAppendSeq...)
	}

	return ws.kvStore.Put(ctx, key, rawTrace)
}

func (ws *Service) deleteAction(ctx context.Context, account uint64, sequenceNumber uint64) error {
	key := encodeActionKey(account, ws.shardNum, sequenceNumber)

	if traceEnabled {
		zlog.Debug("deleting action",
			zap.Uint64("sequence", sequenceNumber),
			zap.Stringer("account", EOSName(account)),
			zap.Stringer("key", Key(key)),
		)
	}

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvStore.BatchDelete(ctx, [][]byte{key})
}

func (ws *Service) writeLastProcessedBlock(ctx context.Context, blk *pbcodec.Block) error {
	key := encodeLastProcessedBlockKey(ws.shardNum)

	ws.lastCheckpoint.LastWrittenBlockNum = blk.Num()
	ws.lastCheckpoint.LastWrittenBlockId = blk.ID()

	value, err := proto.Marshal(ws.lastCheckpoint)
	if err != nil {
		return err
	}

	return ws.kvStore.Put(ctx, key, value)
}

func (ws *Service) flush(ctx context.Context, blk *pbcodec.Block, lastInStreak bool) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	realtimeFlush := time.Since(blk.MustTime()) < 20*time.Minute && lastInStreak
	onFlushIntervalBoundary := blk.Num()%ws.flushBlocksInterval == 0
	if realtimeFlush || onFlushIntervalBoundary {
		if !ws.lastWrite.IsZero() {
			blocks := blk.Num() - ws.lastBlockWritten
			timeDelta := time.Since(ws.lastWrite)
			deltaInSeconds := float64(timeDelta) / float64(time.Second)
			blocksPerSec := float64(blocks) / deltaInSeconds
			zlog.Info("block throughput", zap.Float64("blocks_per_secs", blocksPerSec))
		}
		ws.lastWrite = time.Now()
		return ws.forceFlush(ctx, blk)
	}
	return nil
}

func (ws *Service) forceFlush(ctx context.Context, blk *pbcodec.Block) error {
	zlog.Info("flushed block", zap.Uint64("block_num", blk.Num()))
	return ws.kvStore.FlushPuts(ctx)
}
