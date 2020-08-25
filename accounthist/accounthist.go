package accounthist

import (
	"context"
	"encoding/binary"
	"encoding/hex"
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
	databaseTimeout = 5 * time.Second
)

type Service struct {
	*shutter.Shutter

	shardNum             byte // 0 is live
	maxEntriesPerAccount uint64
	flushBlocksInterval  uint64
	blockFilter          func(blk *bstream.Block) error

	blocksStore dstore.Store
	kvStore     store.KVStore

	historySeqMap map[string]sequenceData
	source        bstream.Source

	rwCache *RWCache
}

func NewService(kvdb store.KVStore, blocksStore dstore.Store, blockFilter func(blk *bstream.Block) error, shardNum byte, maxEntriesPerAccount, flushBlocksInterval uint64) *Service {
	return &Service{
		maxEntriesPerAccount: maxEntriesPerAccount,
		flushBlocksInterval:  flushBlocksInterval,
		Shutter:              shutter.New(),
		shardNum:             shardNum,
		kvStore:              kvdb,
		blocksStore:          blocksStore,
		blockFilter:          blockFilter,
		historySeqMap:        make(map[string]sequenceData),
	}
}

func (ws *Service) Launch() {
	ws.source.OnTerminating(ws.Shutdown)
	ws.OnTerminating(ws.source.Shutdown)
	ws.source.Run()
}

func (ws *Service) GetActions(ctx context.Context, account string) ([]*pbaccounthist.ActionData, error) {
	return ws.scanActions(ctx, account)
}

func (ws *Service) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	ctx := context.Background()

	block, ok := blk.ToNative().(*pbcodec.Block)
	if !ok {
		return fmt.Errorf("could not decode block to native *pbcodec.Block")
	}

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
				acctSeqData, err := ws.getSequenceData(ctx, acct)
				if err != nil {
					return fmt.Errorf("error while getting sequence data for account %v: %w", acct, err)
				}

				if act.Receipt.GlobalSequence <= acctSeqData.lastGlobalSeq {
					zlog.Debug("this block has already been processed for this account", zap.Uint64("block", blk.Num()), zap.String("account", acct))
					continue
				}

				if acctSeqData.historySeqNum+1 > ws.maxEntriesPerAccount {
					err := ws.deleteAction(ctx, acct, acctSeqData.historySeqNum-ws.maxEntriesPerAccount)
					if err != nil {
						return fmt.Errorf("error while deleting action: %w", err)
					}
				}

				//fmt.Println("Writing action", acct, acctSeqData.historySeqNum)

				if err = ws.writeAction(ctx, acct, acctSeqData.historySeqNum, act); err != nil {
					return fmt.Errorf("error while writing action to store: %w", err)
				}

				acctSeqData.Increment(act.Receipt.GlobalSequence)
				ws.updateHistorySeq(acct, acctSeqData)
			}
		}
	}

	// save block progress
	if err := ws.writeLastProcessedBlock(ctx, blk.Num()); err != nil {
		return fmt.Errorf("error while saving block checkpoint")
	}

	if err := ws.flush(ctx, blk.Num()); err != nil {
		return fmt.Errorf("error while flushing block checkpoint write to store: %w", err)
	}

	return nil
}

func (ws *Service) getSequenceData(ctx context.Context, account string) (out sequenceData, err error) {
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

func (ws *Service) updateHistorySeq(account string, seqData sequenceData) {
	ws.historySeqMap[account] = seqData
}

func (ws *Service) readSequenceData(ctx context.Context, account string) (out sequenceData, err error) {

	// TWO GOALS:
	// * for the current `shardNum`, pick up where `lastGlobalSeq` was stopped INSIDE this `shardNum`
	// * get the TOP-MOST shardNum (== 0), or even a few of the top-most shard-nums, to know
	//   if I should not simply ignore that account going forward (say I'm in a very old shard)

	key := make([]byte, actionPrefixKeyLen)
	encodeActionPrefixKey(key, account)

	zlog.Debug("reading sequence data",
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	it := ws.kvStore.Prefix(ctx, key, 1)
	for it.Next() {
		newact := &pbaccounthist.ActionData{}
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

func (ws *Service) scanActions(ctx context.Context, account string) ([]*pbaccounthist.ActionData, error) {
	key := make([]byte, actionPrefixKeyLen)
	encodeActionPrefixKey(key, account)

	zlog.Debug("scanning actions",
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	actions := make([]*pbaccounthist.ActionData, 0, ws.maxEntriesPerAccount)
	it := ws.kvStore.Prefix(ctx, key, int(ws.maxEntriesPerAccount))
	for it.Next() {
		newact := &pbaccounthist.ActionData{}
		err := proto.Unmarshal(it.Item().Value, newact)
		if err != nil {
			return nil, err
		}
		newact.Key = it.Item().Key
		actions = append(actions, newact)
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("error while fetching actions from store: %w", err)
	}

	return actions, nil
}

func (ws *Service) writeAction(ctx context.Context, account string, sequenceNumber uint64, actionTrace *pbcodec.ActionTrace) error {
	key := make([]byte, actionKeyLen)
	encodeActionKey(key, account, ws.shardNum, sequenceNumber)

	d := &pbaccounthist.ActionData{}
	d.ActionTrace = actionTrace
	d.SequenceNumber = sequenceNumber

	rawTrace, err := proto.Marshal(d)
	if err != nil {
		return err
	}

	zlog.Debug("writing action",
		zap.String("account", account),
		zap.String("action", actionTrace.String()),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvStore.Put(ctx, key, rawTrace)
}

func (ws *Service) deleteAction(ctx context.Context, account string, sequenceNumber uint64) error {
	key := make([]byte, actionKeyLen)
	encodeActionKey(key, account, ws.shardNum, sequenceNumber)

	zlog.Debug("deleting action",
		zap.Uint64("sequence", sequenceNumber),
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvStore.BatchDelete(ctx, [][]byte{key})
}

func (ws *Service) flush(ctx context.Context, blkNum uint64) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	if blkNum%ws.flushBlocksInterval == 0 {
		zlog.Info("flushed block", zap.Uint64("block_num", blkNum))
		return ws.kvStore.FlushPuts(ctx)
	}

	return nil
}
