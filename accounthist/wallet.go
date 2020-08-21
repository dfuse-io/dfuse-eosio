package wallet

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	_ "github.com/dfuse-io/dfuse-eosio/accounthist/codec"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/hub"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

const (
	maxEntriesPerAccount = 10
	databaseTimeout      = 5 * time.Second
)

type Store struct {
	kvstore            store.KVStore
	accountSequenceMap map[string]*sequenceData
	seqMutex           *sync.RWMutex

	subscriptionHub *hub.SubscriptionHub
}

func NewStore(kvdb store.KVStore) *Store {
	store := &Store{
		kvstore:            kvdb,
		accountSequenceMap: make(map[string]*sequenceData),
		seqMutex:           &sync.RWMutex{},
	}
	return store
}

func (ws *Store) GetTransactions(ctx context.Context, account string) ([]string, error) {
	return ws.scanTransactions(ctx, account)
}

func (ws *Store) GetBlockHandler(ctx context.Context) bstream.Handler {
	return bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
		block, ok := blk.ToNative().(*pbcodec.Block)
		if !ok {
			return fmt.Errorf("could not decode block to native *pbcodec.Block")
		}

		currentBlockNumber := uint64(block.GetNumber())
		for _, tx := range block.TransactionTraces() {
			for _, act := range tx.ActionTraces {
				sequenceData, err := ws.getSequenceData(ctx, act.Account())
				if err != nil {
					return fmt.Errorf("error while getting sequence data for account %v: %w", act.Account(), err)
				}

				if currentBlockNumber <= sequenceData.PreviousBlockNumber {
					zlog.Debug("this block has already been processed for this account", zap.Uint64("block", currentBlockNumber), zap.String("account", act.Account()))
					continue
				}

				if sequenceData.CurrentSequenceID+1 > maxEntriesPerAccount {
					//TODO: batch these?
					err := ws.deleteAction(ctx, act.Account(), sequenceData.CurrentSequenceID-maxEntriesPerAccount)
					if err != nil {
						return fmt.Errorf("error while deleting transaction: %w", err)
					}
				}

				if err = ws.writeAction(ctx, act.Account(), sequenceData.CurrentSequenceID, act); err != nil {
					return fmt.Errorf("error while writing transaction to store: %w", err)
				}

				ws.incrementSequence(act.Account(), sequenceData)
			}

			// before saving checkpoints in sequence data, make sure all transactions are safely written to store
			if err := ws.flush(ctx); err != nil {
				return fmt.Errorf("error while flushing transaction writes to store: %w", err)
			}

			// save checkpoints for accounts
			// TODO: only save checkpoints for accounts which were modified in this block
			if err := ws.setSequenceCheckpoints(ctx, currentBlockNumber); err != nil {
				return fmt.Errorf("error while saving checkpoints")
			}
			if err := ws.flush(ctx); err != nil {
				return fmt.Errorf("error while flushing checkpoint writes to store: %w", err)
			}
		}

		// save block progress
		if err := ws.writeLastProcessedBlock(ctx, currentBlockNumber); err != nil {
			return fmt.Errorf("error while saving block checkpoint")
		}
		if err := ws.flush(ctx); err != nil {
			return fmt.Errorf("error while flushing block checkpoint write to store: %w", err)
		}

		return nil
	})
}

func (ws *Store) getSequenceData(ctx context.Context, account string) (*sequenceData, error) {
	ws.seqMutex.Lock()
	defer ws.seqMutex.Unlock()

	seqData, ok := ws.accountSequenceMap[account]
	if !ok {
		var err error
		seqData, err = ws.readSequenceData(ctx, account)
		if err == store.ErrNotFound {
			seqData = &sequenceData{}
		} else if err != nil {
			return nil, fmt.Errorf("error while fetching sequence data: %w", err)
		}
	}

	return seqData, nil
}

func (ws *Store) GetLastProcessedBlock(ctx context.Context) (uint64, error) {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	val, err := ws.kvstore.Get(ctx, key)
	if err == store.ErrNotFound {
		return uint64(0), nil
	} else if err != nil {
		return uint64(0), fmt.Errorf("error while last processed block: %w", err)
	}

	return binary.LittleEndian.Uint64(val), nil
}

func (ws *Store) writeLastProcessedBlock(ctx context.Context, blockNumber uint64) error {
	key := make([]byte, lastBlockKeyLen)
	encodeLastProcessedBlockKey(key)

	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, blockNumber)

	return ws.kvstore.Put(ctx, key, value)
}

func (ws *Store) incrementSequence(account string, seqData *sequenceData) {
	ws.seqMutex.Lock()
	defer ws.seqMutex.Unlock()

	seqData.Increment()
	ws.accountSequenceMap[account] = seqData
}

func (ws *Store) setSequenceCheckpoints(ctx context.Context, blockNumber uint64) error {
	ws.seqMutex.Lock()
	defer ws.seqMutex.Unlock()

	for account, sequenceData := range ws.accountSequenceMap {
		sequenceData.SetCheckpoint(blockNumber)

		err := ws.writeSequenceData(ctx, account, sequenceData)
		if err != nil {
			return fmt.Errorf("error while writing sequence number checkpoints to store: %w", err)
		}
		ws.accountSequenceMap[account] = sequenceData
	}

	return nil
}

func (ws *Store) writeSequenceData(ctx context.Context, account string, sequenceData *sequenceData) error {
	key := make([]byte, sequenceKeyLen)
	encodeSequenceKey(key, account)

	value := make([]byte, sequenceDataValueLength)
	sequenceData.Encode(value)

	zlog.Debug("writing sequence number",
		zap.String("account", account),
		zap.Object("sequence data", sequenceData),
		zap.String("key", hex.EncodeToString(key)),
		zap.String("value", hex.EncodeToString(value)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvstore.Put(ctx, key, value)
}

func (ws *Store) readSequenceData(ctx context.Context, account string) (*sequenceData, error) {
	key := make([]byte, sequenceKeyLen)
	encodeSequenceKey(key, account)

	zlog.Debug("reading sequence data",
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	res, err := ws.kvstore.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(res) < sequenceDataValueLength {
		return nil, fmt.Errorf("invalid sequence data from database. data must be at least %v bytes", sequenceDataValueLength)
	}

	sequenceData := &sequenceData{}
	sequenceData.Decode(res)

	return sequenceData, nil
}

func (ws *Store) scanTransactions(ctx context.Context, account string) ([]string, error) {
	key := make([]byte, actionPrefixKeyLen)
	encodeTransactionPrefixKey(key, account)

	zlog.Debug("scanning transactions",
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	transactions := make([]string, 0, maxEntriesPerAccount)
	it := ws.kvstore.Prefix(ctx, key, maxEntriesPerAccount)
	for it.Next() {
		transactions = append(transactions, string(it.Item().Value))
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("error while fetching transactions from store: %w", err)
	}

	return transactions, nil
}

func (ws *Store) writeAction(ctx context.Context, account string, sequenceNumber uint64, actionTrace *pbcodec.ActionTrace) error {
	key := make([]byte, actionKeyLen)
	encodeTransactionKey(key, account, sequenceNumber)

	value := actionTrace.Action.RawData

	zlog.Debug("writing transaction",
		zap.String("account", account),
		zap.String("action", actionTrace.String()),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvstore.Put(ctx, key, value)
}

func (ws *Store) deleteAction(ctx context.Context, account string, sequenceNumber uint64) error {
	key := make([]byte, actionKeyLen)
	encodeTransactionKey(key, account, sequenceNumber)

	keys := make([][]byte, 1)
	keys[0] = key

	zlog.Debug("deleting transaction",
		zap.Uint64("sequence", sequenceNumber),
		zap.String("account", account),
		zap.String("key", hex.EncodeToString(key)),
	)

	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvstore.BatchDelete(ctx, keys)
}

func (ws *Store) flush(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	return ws.kvstore.FlushPuts(ctx)
}
