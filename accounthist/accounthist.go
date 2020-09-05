package accounthist

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
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

	historySeqMap map[uint64]SequenceData
	source        bstream.Source

	rwCache *RWCache

	startBlockNum uint64
	stopBlockNum  uint64

	tracker *bstream.Tracker

	lastCheckpoint *pbaccounthist.ShardCheckpoint

	lastWrittenBlock *lastWrittenBlock
}

type lastWrittenBlock struct {
	blockNum  uint64
	writtenAt time.Time
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

		historySeqMap: make(map[uint64]SequenceData),
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

func (ws *Service) deleteStaleRows(ctx context.Context, account uint64, acctSeqData SequenceData) (lastDeletedSeq uint64, err error) {
	// If the last current ordinal is bigger than the max allowed entries for this account,
	// adjust our sliding window by deleting anything below least recent ordinal
	if acctSeqData.CurrentOrdinal > acctSeqData.MaxEntries {
		// Don't forget, we are in a sliding window setup, so if last written was 12, assuming max entry of 5,
		// we have normally a window composed of ordinals [8, 9, 10, 11, 12], so anything from 7 and downwards should
		// be deleted.
		leastRecentOrdinal := acctSeqData.CurrentOrdinal - acctSeqData.MaxEntries

		// Assuming for this account that the last deleted ordinal is already higher or equal to our least
		// recent ordinal, it means there is nothing to do since everything below this last deleted ordinal
		// should already be gone.
		if acctSeqData.LastDeletedOrdinal >= leastRecentOrdinal {
			return acctSeqData.LastDeletedOrdinal, nil
		}

		// Let's assume our last deleted ordinal was 5, let's delete everything from 5 up and including 7
		zlog.Debug("deleting all actions between last deleted and now least recent ordinal", zap.Uint64("last_deleted_ordinal", acctSeqData.LastDeletedOrdinal), zap.Uint64("least_recent_ordinal", leastRecentOrdinal))
		for i := acctSeqData.LastDeletedOrdinal + 1; i <= leastRecentOrdinal; i++ {
			err := ws.deleteAction(ctx, account, i)
			if err != nil {
				return 0, fmt.Errorf("error while deleting action: %w", err)
			}
		}
		return leastRecentOrdinal, nil
	}

	return acctSeqData.LastDeletedOrdinal, nil
}

func (ws *Service) processSequenceDataKeyValue(item store.KV) (SequenceData, error) {
	s := SequenceData{}
	_, _, s.CurrentOrdinal = decodeActionKeySeqNum(item.Key)
	newact := &pbaccounthist.ActionRow{}
	if err := proto.Unmarshal(item.Value, newact); err != nil {
		return SequenceData{}, err
	}
	s.LastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
	s.LastDeletedOrdinal = newact.LastDeletedSeq

	return s, nil
}

func (ws *Service) getSequenceData(ctx context.Context, account uint64) (out SequenceData, err error) {
	out, ok := ws.historySeqMap[account]
	if ok {
		return
	}

	out, err = ws.shardNewestSequenceData(ctx, account, ws.shardNum, ws.processSequenceDataKeyValue)

	if err == store.ErrNotFound {
		zlog.Debug("account never seen before, initializing a new sequence data",
			zap.Stringer("account", EOSName(account)),
		)
		out.CurrentOrdinal = 0
		out.MaxEntries = ws.maxEntriesPerAccount
	} else if err != nil {
		err = fmt.Errorf("error while fetching sequence data: %w", err)
		return
	}
	maxEntriesForAccount, err := ws.readMaxEntries(ctx, account)
	if err != nil {
		err = fmt.Errorf("error fetching max entries: %w", err)
		return
	}

	out.MaxEntries = maxEntriesForAccount
	zlog.Debug("account sequence data setup",
		zap.Int("shard_num", int(ws.shardNum)),
		zap.Stringer("account", EOSName(account)),
		zap.Uint64("account_max_entries", ws.maxEntriesPerAccount),
		zap.Uint64("seq_data_max_entries", out.MaxEntries),
		zap.Uint64("seq_data_current_ordinal", out.CurrentOrdinal),
		zap.Uint64("seq_data_last_global_sequence", out.LastGlobalSeq),
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

func (ws *Service) updateHistorySeq(account uint64, seqData SequenceData) {
	ws.historySeqMap[account] = seqData
}

func (ws *Service) readMaxEntries(ctx context.Context, account uint64) (maxEntries uint64, err error) {
	return ws.maxEntriesPerAccount, nil
	//shardsToCheck := 0
	//nextShardNum := byte(0)
	//var seenActions uint64
	//for i := 0; i < shardsToCheck; i++ {
	//	if nextShardNum >= ws.shardNum {
	//		// we'll stop writing only if FUTURE shards have covered our `maxEntriesPerAccount`.
	//		break
	//	}
	//
	//	startKey := encodeActionKey(account, nextShardNum, math.MaxUint64)
	//	endKey := encodeActionKey(account+1, 0, 0)
	//
	//	zlog.Debug("reading sequence data",
	//		zap.Stringer("account", EOSName(account)),
	//		zap.Int("shard_num", int(nextShardNum)),
	//		zap.Stringer("start_key", Key(startKey)),
	//		zap.Stringer("end_key", Key(endKey)),
	//	)
	//
	//	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	//	defer cancel()
	//
	//	it := ws.kvStore.Scan(ctx, startKey, endKey, 1)
	//	var rows int
	//	for it.Next() {
	//		rows++
	//		_, currentShardNum, historySeqNum := decodeActionKeySeqNum(it.Item().Key)
	//		seenActions += historySeqNum
	//
	//		if seenActions >= ws.maxEntriesPerAccount {
	//			return 0, nil
	//		}
	//
	//		nextShardNum = currentShardNum + 1
	//	}
	//	if it.Err() != nil {
	//		err = it.Err()
	//		return
	//	}
	//	if rows == 0 {
	//		break
	//	}
	//}
	//
	//return ws.maxEntriesPerAccount - seenActions, nil
}

func (ws *Service) writeAction(ctx context.Context, account uint64, acctSeqData SequenceData, actionTrace *pbcodec.ActionTrace, rawTrace []byte) error {
	key := encodeActionKey(account, ws.shardNum, acctSeqData.CurrentOrdinal)

	zlog.Debug("writing action", zap.Stringer("account", EOSName(account)), zap.Stringer("key", Key(key)))

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	if acctSeqData.LastDeletedOrdinal != 0 {
		// this is will append the protobuf pbaccounthist.ActionRowAppend to the protobuf pbaccounthist.ActionRow, since
		// both struct have the field last_deleted_seq with the same index (3), when unmarshalling the row
		// into an pbaccounthist.ActionRow the value of `last_deleted_seq` in the appended pbaccounthist.ActionRowAppend will
		// override the value defined in the pbaccounthist.ActionRow struct
		appendSeq := &pbaccounthist.ActionRowAppend{LastDeletedSeq: acctSeqData.LastDeletedOrdinal}
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

		if ws.lastWrittenBlock != nil {
			blocks := blk.Num() - ws.lastWrittenBlock.blockNum
			timeDelta := time.Since(ws.lastWrittenBlock.writtenAt)
			deltaInSeconds := float64(timeDelta) / float64(time.Second)
			blocksPerSec := float64(blocks) / deltaInSeconds
			zlog.Info("block throughput",
				zap.Float64("blocks_per_secs", blocksPerSec),
				zap.Uint64("last_written_block_num", ws.lastWrittenBlock.blockNum),
				zap.Uint64("current_block_num", blk.Num()),
				zap.Time("last_written_block_at", ws.lastWrittenBlock.writtenAt),
			)
		}
		ws.lastWrittenBlock = &lastWrittenBlock{
			blockNum:  blk.Num(),
			writtenAt: time.Now(),
		}
		zlog.Info("starting force flush", zap.Uint64("block_num", blk.Num()))
		return ws.forceFlush(ctx)
	}
	return nil
}

func (ws *Service) forceFlush(ctx context.Context) error {
	return ws.kvStore.FlushPuts(ctx)
}
