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

func (ws *Service) Terminate() {

}

func (ws *Service) deleteStaleRows(ctx context.Context, account uint64, acctSeqData sequenceData) (lastDeletedSeq uint64, err error) {

	// a) value of nextOrdinal:  5 (i.e. you have written 4 actions)
	// b) value of nextOrdinal:  6 (i.e. you have written 5 actions)
	// c) value of nextOrdinal:  12 (i.e. you have written 5 actions the last one being the 11th you saw) 11-10-9-8-7
	// max entries is 5
	// number of written actions is = nextOrdinal - 1
	//if (acctSeqData.nextOrdinal - 1) > acctSeqData.maxEntries {
	//
	//}
	if (acctSeqData.nextOrdinal - 1) > acctSeqData.maxEntries {
		// j'écris 2001, va effacer 1001 (last deleted devrait être 1000, si y'avait aucune config change)
		// si le dernier que j'ai effacé était 1500, perds pas ton temps
		// si le dernier que j'ai effacé était 500, delete 501 jusqu'à 1001

		deleteSeq := acctSeqData.nextOrdinal - acctSeqData.maxEntries
		// THERE'S AN OFF BY ONE ISSUE HERE, WE NEVER DELETE THE historySeqNum == 0
		// but I thought we didn't even WRITE that historySeqNum.. because we start that
		// history at `1` when we create a new `sequenceData` (!)
		if acctSeqData.lastDeletedOrdinal >= deleteSeq {
			return acctSeqData.lastDeletedOrdinal, nil
		}

		for i := acctSeqData.lastDeletedOrdinal + 1; i <= deleteSeq; i++ {
			// from acctSeqData.lastDeletedSeq up to `deleteSeq`
			err := ws.deleteAction(ctx, account, i)
			if err != nil {
				return 0, fmt.Errorf("error while deleting action: %w", err)
			}
		}
		return deleteSeq, nil
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
		out = sequenceData{
			nextOrdinal: 1, // FIXME: where is this initialized? How come are we writing some actions at index 0 ?!?
			maxEntries:  ws.maxEntriesPerAccount,
		}
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

	zlog.Debug("max entries for account", zap.Int("shard_num", int(ws.shardNum)), zap.Stringer("account", EOSName(account)), zap.Uint64("max_entries", maxEntriesForAccount))

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
	endKey := encodeActionKey(account+1, 0, 0)

	zlog.Debug("reading sequence data",
		zap.Stringer("account", EOSName(account)),
		zap.Int("shard_num", int(ws.shardNum)),
		zap.Stringer("start_key", Key(startKey)),
		zap.Stringer("end_key", Key(endKey)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := ws.kvStore.Scan(ctx, startKey, endKey, 1)
	for it.Next() {
		newact := &pbaccounthist.ActionRow{}
		if err = proto.Unmarshal(it.Item().Value, newact); err != nil {
			return
		}
		// unique identifying global sequence per account
		out.lastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
		// internal sequence number for a given shard
		_, out.nextOrdinal = decodeActionKeySeqNum(it.Item().Key)
		out.nextOrdinal++
		out.lastDeletedOrdinal = newact.LastDeletedSeq
	}
	if it.Err() != nil {
		err = it.Err()
		return
	}

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

	zlog.Debug("deleting action",
		zap.Uint64("sequence", sequenceNumber),
		zap.Stringer("account", EOSName(account)),
		zap.Stringer("key", Key(key)),
	)

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
