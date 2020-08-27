package accounthist

import (
	"context"
	"encoding/binary"
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

	if ws.stopBlockNum != 0 && blk.Num() >= ws.stopBlockNum {
		// FLUSH all the things
		if err := ws.forceFlush(ctx, blk.Num()); err != nil {
			ws.Shutdown(err)
			return fmt.Errorf("flushing when stopping: %w", err)
		}
		ws.Shutdown(nil)
		return nil
	}

	block := blk.ToNative().(*pbcodec.Block)
	fObj := obj.(*forkable.ForkableObject)
	rawTraceMap := fObj.Obj.(map[uint64][]byte)

	for _, tx := range block.TransactionTraces() {
		if tx.HasBeenReverted() {
			continue
		}

		for _, act := range tx.ActionTraces {
			if act.Receipt == nil {
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

				if acctSeqData.historySeqNum+1 > ws.maxEntriesPerAccount {
					err := ws.deleteAction(ctx, acctUint, acctSeqData.historySeqNum-ws.maxEntriesPerAccount)
					if err != nil {
						return fmt.Errorf("error while deleting action: %w", err)
					}
				}

				//fmt.Println("Writing action", acct, acctSeqData.historySeqNum)

				rawTrace := rawTraceMap[act.Receipt.GlobalSequence]

				if err = ws.writeAction(ctx, acctUint, acctSeqData.historySeqNum, act, rawTrace); err != nil {
					return fmt.Errorf("error while writing action to store: %w", err)
				}

				acctSeqData.Increment(act.Receipt.GlobalSequence)
				ws.updateHistorySeq(acctUint, acctSeqData)
			}
		}
	}

	// save block progress
	if err := ws.writeLastProcessedBlock(ctx, blk.Num()); err != nil {
		return fmt.Errorf("error while saving block checkpoint")
	}

	if err := ws.flush(ctx, blk.Num()); err != nil {
		return fmt.Errorf("error while flushing: %w", err)
	}

	return nil
}

func (ws *Service) getSequenceData(ctx context.Context, account uint64) (out sequenceData, err error) {
	out, ok := ws.historySeqMap[account]
	if ok {
		return
	}

	out, err = ws.readSequenceData(ctx, account)
	if err == store.ErrNotFound {
		out = sequenceData{}
	} else if err != nil {
		err = fmt.Errorf("error while fetching sequence data: %w", err)
		return
	}
	return
}

func (ws *Service) GetLastProcessedBlock(ctx context.Context) (uint64, error) {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key, ws.shardNum)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	val, err := ws.kvStore.Get(ctx, key)
	if err == store.ErrNotFound {
		return uint64(0), nil
	} else if err != nil {
		return uint64(0), fmt.Errorf("error while last processed block: %w", err)
	}

	return binary.LittleEndian.Uint64(val), nil
}

func (ws *Service) writeLastProcessedBlock(ctx context.Context, blockNumber uint64) error {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key, ws.shardNum)

	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, blockNumber)

	return ws.kvStore.Put(ctx, key, value)
}

func (ws *Service) updateHistorySeq(account uint64, seqData sequenceData) {
	ws.historySeqMap[account] = seqData
}

func (ws *Service) readSequenceData(ctx context.Context, account uint64) (out sequenceData, err error) {

	// TWO GOALS:
	// * for the current `shardNum`, pick up where `lastGlobalSeq` was stopped INSIDE this `shardNum`
	// * get the TOP-MOST shardNum (== 0), or even a few of the top-most shard-nums, to know
	//   if I should not simply ignore that account going forward (say I'm in a very old shard)

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
	}
	if it.Err() != nil {
		err = it.Err()
		return
	}

	return
}

func (ws *Service) GetActions(req *pbaccounthist.GetActionsRequest, stream pbaccounthist.AccountHistory_GetActionsServer) error {
	ctx := stream.Context()
	account := req.Account
	accountName := eos.NameToString(account)

	// TODO: triple check that `account` is an EOS Name (encode / decode and check for ==, otherwise, BadRequest), perhaps at the DGraphQL level plz

	queryShardNum := byte(255)
	querySeqNum := uint64(math.MaxUint64)
	if req.Cursor != nil {
		// TODO: we could check that the Cursor.ShardNum doesn't go above 255
		queryShardNum = byte(req.Cursor.ShardNum)
		querySeqNum = req.Cursor.SequenceNumber - 1 // FIXME: CHECK BOUNDARIES, this is EXCLUSIVE, so do we -1, +1 ?
	}

	if req.Limit < 0 {
		return fmt.Errorf("negative limit is not valid")
	}

	startKey := make([]byte, actionKeyLen)
	encodeActionKey(startKey, account, queryShardNum, querySeqNum)
	endKey := make([]byte, actionKeyLen)
	encodeActionKey(endKey, account, 0, 0)

	zlog.Info("scanning actions",
		zap.String("account", accountName),
		zap.String("start_key", hex.EncodeToString(startKey)), // TODO: turn into a hex Stringer(), instead of encoding it all the time
		zap.String("end_key", hex.EncodeToString(endKey)),     // TODO: turn into a hex Stringer(), instead of encoding it all the time
	)

	limit := int(ws.maxEntriesPerAccount)
	if req.Limit != 0 && int(req.Limit) < limit {
		limit = int(req.Limit)
	}
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()
	it := ws.kvStore.Scan(ctx, startKey, endKey, limit)
	for it.Next() {
		newact := &pbaccounthist.ActionRow{}
		err := proto.Unmarshal(it.Item().Value, newact)
		if err != nil {
			return err
		}

		newresp := &pbaccounthist.ActionResponse{
			Cursor:      actionKeyToCursor(account, it.Item().Key),
			ActionTrace: newact.ActionTrace,
		}

		if err := stream.Send(newresp); err != nil {
			return err
		}
	}
	if err := it.Err(); err != nil {
		return fmt.Errorf("error while fetching actions from store: %w", err)
	}

	return nil
}

func (ws *Service) writeAction(ctx context.Context, account uint64, sequenceNumber uint64, actionTrace *pbcodec.ActionTrace, rawTrace []byte) error {
	key := make([]byte, actionKeyLen)
	encodeActionKey(key, account, ws.shardNum, sequenceNumber)

	zlog.Debug("writing action",
		zap.Uint64("account", account),
		zap.Stringer("action", actionTrace),
		//zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

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

func (ws *Service) forceFlush(ctx context.Context, blkNum uint64) error {
	zlog.Info("flushed block", zap.Uint64("block_num", blkNum))
	return ws.kvStore.FlushPuts(ctx)
}
func (ws *Service) flush(ctx context.Context, blkNum uint64) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	if blkNum%ws.flushBlocksInterval == 0 {
		if !ws.lastWrite.IsZero() {
			blocks := blkNum - ws.lastBlockWritten
			timeDelta := time.Since(ws.lastWrite)
			deltaInSeconds := float64(timeDelta) / float64(time.Second)
			blocksPerSec := float64(blocks) / deltaInSeconds
			zlog.Info("block throughput", zap.Float64("blocks_per_secs", blocksPerSec))
		}
		ws.lastWrite = time.Now()
		ws.lastBlockWritten = blkNum
		return ws.forceFlush(ctx, blkNum)
	}

	return nil
}
