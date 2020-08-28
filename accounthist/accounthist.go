package accounthist

import (
	"context"
	"encoding/hex"
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

func NewService(kvdb store.KVStore, blocksStore dstore.Store, blockFilter func(blk *bstream.Block) error, shardNum byte, maxEntriesPerAccount, flushBlocksInterval uint64, startBlockNum, stopBlockNum uint64, tracker *bstream.Tracker) *Service {
	return &Service{
		maxEntriesPerAccount: maxEntriesPerAccount,
		flushBlocksInterval:  flushBlocksInterval,
		Shutter:              shutter.New(),
		shardNum:             shardNum,
		kvStore:              kvdb,
		blocksStore:          blocksStore,
		blockFilter:          blockFilter,
		historySeqMap:        make(map[uint64]sequenceData),
		startBlockNum:        startBlockNum,
		stopBlockNum:         stopBlockNum,
		tracker:              tracker,
	}
}

func (ws *Service) Launch() {
	ws.source.OnTerminating(ws.Shutdown)
	ws.OnTerminating(ws.source.Shutdown)
	ws.source.Run()
}

func (ws *Service) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	ctx := context.Background()

	block := blk.ToNative().(*pbcodec.Block)
	fObj := obj.(*forkable.ForkableObject)
	rawTraceMap := fObj.Obj.(map[uint64][]byte)
	isLastInStreak := fObj.StepIndex+1 == fObj.StepCount

	if ws.stopBlockNum != 0 && blk.Num() >= ws.stopBlockNum {
		// FLUSH all the things
		if err := ws.forceFlush(ctx, block); err != nil {
			ws.Shutdown(err)
			return fmt.Errorf("flushing when stopping: %w", err)
		}
		ws.Shutdown(nil)
		return nil
	}

	for _, tx := range block.TransactionTraces() {
		if tx.HasBeenReverted() {
			continue
		}

		for _, act := range tx.ActionTraces {
			if act.Receipt == nil {
				continue
			}

			if block.FilteringApplied && !act.FilteringMatched {
				continue
			}

			accts := map[string]bool{act.Receiver: true}
			for _, v := range act.Action.Authorization {
				accts[v.Actor] = true
			}

			for acct := range accts {
				acctUint := eos.MustStringToName(acct)
				acctSeqData, err := ws.getSequenceData(ctx, acctUint)
				if err != nil {
					return fmt.Errorf("error while getting sequence data for account %v: %w", acct, err)
				}

				if act.Receipt.GlobalSequence <= acctSeqData.lastGlobalSeq {
					zlog.Debug("this block has already been processed for this account", zap.Uint64("block", blk.Num()), zap.String("account", acct))
					continue
				}

				lastDeletedSeq, err := ws.deleteStaleRows(ctx, acctUint, acctSeqData)
				if err != nil {
					return err
				}

				acctSeqData.lastDeletedSeq = lastDeletedSeq
				rawTrace := rawTraceMap[act.Receipt.GlobalSequence]

				if err = ws.writeAction(ctx, acctUint, acctSeqData, act, rawTrace); err != nil {
					return fmt.Errorf("error while writing action to store: %w", err)
				}

				acctSeqData.Increment(act.Receipt.GlobalSequence)
				ws.updateHistorySeq(acctUint, acctSeqData)
			}
		}
	}

	// save block progress
	if err := ws.writeLastProcessedBlock(ctx, block); err != nil {
		return fmt.Errorf("error while saving block checkpoint")
	}

	if err := ws.flush(ctx, block, isLastInStreak); err != nil {
		return fmt.Errorf("error while flushing: %w", err)
	}

	return nil
}

func (ws *Service) deleteStaleRows(ctx context.Context, account uint64, acctSeqData sequenceData) (lastDeletedSeq uint64, err error) {
	if acctSeqData.historySeqNum+1 > acctSeqData.maxEntries {
		// j'écris 2001, va effacer 1001 (last deleted devrait être 1000, si y'avait aucune config change)
		// si le dernier que j'ai effacé était 1500, perds pas ton temps
		// si le dernier que j'ai effacé était 500, delete 501 jusqu'à 1001
		deleteSeq := acctSeqData.historySeqNum - acctSeqData.maxEntries

		if acctSeqData.lastDeletedSeq >= deleteSeq {
			return acctSeqData.lastDeletedSeq, nil
		} else {
			for i := acctSeqData.lastDeletedSeq + 1; i <= deleteSeq; i++ {
				// from acctSeqData.lastDeletedSeq up to `deleteSeq`
				err := ws.deleteAction(ctx, account, i)
				if err != nil {
					return 0, fmt.Errorf("error while deleting action: %w", err)
				}
			}
			return deleteSeq, nil
		}
	}
	return acctSeqData.lastDeletedSeq, nil
}

func (ws *Service) getSequenceData(ctx context.Context, account uint64) (out sequenceData, err error) {
	out, ok := ws.historySeqMap[account]
	if ok {
		return
	}

	out, err = ws.readSequenceData(ctx, account)
	if err == store.ErrNotFound {
		out = sequenceData{
			historySeqNum: 1,
			maxEntries:    ws.maxEntriesPerAccount,
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

	zlog.Debug("max entries for account", zap.Int("shard_num", int(ws.shardNum)), zap.Uint64("account", account), zap.Uint64("max_entries", maxEntriesForAccount))

	return
}

func (ws *Service) GetShardCheckpoint(ctx context.Context) (*pbaccounthist.ShardCheckpoint, error) {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key, ws.shardNum)

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

		startKey := make([]byte, actionKeyLen)
		encodeActionKey(startKey, account, nextShardNum, math.MaxUint64)
		endKey := make([]byte, actionKeyLen)
		encodeActionKey(endKey, account+1, 0, 0)

		zlog.Debug("reading sequence data",
			//zap.String("account", account),
			zap.Int("shard_num", int(nextShardNum)),
			zap.String("start_key", hex.EncodeToString(startKey)),
			zap.String("end_key", hex.EncodeToString(endKey)),
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
	key := make([]byte, actionPrefixKeyLen)
	encodeActionPrefixKey(key, account)

	zlog.Debug("reading sequence data",
		//zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := ws.kvStore.Prefix(ctx, key, 1)
	for it.Next() {
		newact := &pbaccounthist.ActionRow{}
		if err = proto.Unmarshal(it.Item().Value, newact); err != nil {
			return
		}
		out.lastGlobalSeq = newact.ActionTrace.Receipt.GlobalSequence
		_, out.historySeqNum = decodeActionKeySeqNum(it.Item().Key)
		out.historySeqNum++
		out.lastDeletedSeq = newact.LastDeletedSeq
	}
	if it.Err() != nil {
		err = it.Err()
		return
	}

	return
}

func (ws *Service) writeAction(ctx context.Context, account uint64, acctSeqData sequenceData, actionTrace *pbcodec.ActionTrace, rawTrace []byte) error {
	key := make([]byte, actionKeyLen)
	encodeActionKey(key, account, ws.shardNum, acctSeqData.historySeqNum)

	zlog.Debug("writing action",
		zap.Uint64("account", account),
		zap.Stringer("action", actionTrace),
		//zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	if acctSeqData.lastDeletedSeq != 0 {
		appendSeq := &pbaccounthist.ActionRowAppend{LastDeletedSeq: acctSeqData.lastDeletedSeq}
		encodedAppendSeq, _ := proto.Marshal(appendSeq)
		rawTrace = append(rawTrace, encodedAppendSeq...)
	}

	return ws.kvStore.Put(ctx, key, rawTrace)
}

func (ws *Service) deleteAction(ctx context.Context, account uint64, sequenceNumber uint64) error {
	key := make([]byte, actionKeyLen)
	encodeActionKey(key, account, ws.shardNum, sequenceNumber)

	zlog.Debug("deleting action",
		zap.Uint64("sequence", sequenceNumber),
		//zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvStore.BatchDelete(ctx, [][]byte{key})
}

func (ws *Service) writeLastProcessedBlock(ctx context.Context, blk *pbcodec.Block) error {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key, ws.shardNum)

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
