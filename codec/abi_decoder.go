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
package codec

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dhammer"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

var noActiveBlockNum uint64 = math.MaxUint64

// ABIDecoder holds the ABI cache, controls it and process transaction
// on the fly parallel decoding the various elements that needs ABI decoding
// in-place in the data structure.
//
// The ABI decoder is the one that controls the locking of the cache, so all
// locking of the cache should be performed by the decoder. The idea here is to
// have full control of the locking, so we can write-lock the cache and add
// multiple ABIs inside a single locking session, than resume with the read.
// That is to improve lock-contention.
type ABIDecoder struct {
	cache  *ABICache
	hammer *dhammer.Hammer

	hammerFeederWg   sync.WaitGroup
	hammerConsumerWg sync.WaitGroup

	// The logic of truncation is the following. We assume we will always receives
	// blocks in sequential order, expect when there is a fork, we could go back
	// in the past or changing the actual block. Assume a single block fork situation,
	// it means we would received `1a`, then `2b` then `2a` or in a multi blocks
	// situation like `1a`, then `2b` - `3b` - `4b` then `2a` - `3a` - `4a`.
	//
	// The idea at the point is that the decoder received signals when a block starts
	// and ends. Each time we finish a full block, we record it's block num. When a new
	// block arrives, it should stricly sequentially follow our last seen block num.
	// This is never respected in a fork situation, assuming last block is `2b`, when we
	// received `2a`, it's not following it, if last block was `4b`, same thing.
	//
	// Now, we are in a fork situation, this means we must removed any previously defined
	// ABI. The trick here is to leverage the global sequence number. When we detect the
	// fork, we flag the decoder that it needs to peform a truncation. On the next transaction
	// that arrive, we extract the first global sequence we can find. This is our truncation
	// value. Any ABI set after or equal to this value must be truncated, for each and every
	// account.
	//
	// In the event no valid transaction is in the block, the flag remains and we continue
	// on, until we are actually able to find our first new global sequence value. This is
	// ok because the global sequence while there cannot move on if no action is executed.
	activeBlockNum               uint64
	lastSeenBlockRef             bstream.BlockRef
	truncateOnNextGlobalSequence bool
}

type actionDecodingJob struct {
	actionTrace *pbcodec.ActionTrace
	actionIndex int
	localCache  *ABICache
}

func newABIDecoder() *ABIDecoder {
	decoder := &ABIDecoder{
		cache:            newABICache(),
		activeBlockNum:   noActiveBlockNum,
		lastSeenBlockRef: bstream.BlockRefEmpty,
	}

	decoder.hammer = dhammer.NewHammer(1, 8, decoder.executeDecodingJob)

	return decoder
}

func (c *ABIDecoder) startBlock(ctx context.Context, blockNum uint64) error {
	zlog.Debug("starting a new block", zap.Uint64("block_num", blockNum), zap.Stringer("previous_block", c.lastSeenBlockRef))
	if c.activeBlockNum != noActiveBlockNum {
		return fmt.Errorf("start block for block #%d received while already processing block #%d", blockNum, c.activeBlockNum)
	}

	c.activeBlockNum = blockNum

	// If the last seen block is not stricly preceding the block newly started, we are in a fork situation
	if c.lastSeenBlockRef != bstream.BlockRefEmpty && c.lastSeenBlockRef.Num()+1 != blockNum {
		zlog.Debug("starting block is not strictly following last processed one, setting truncation required flag")
		c.truncateOnNextGlobalSequence = true
	}

	// FIXME: Replace 8 with as many CPUs available minus one (or two) we have. Will need some profiling to see the best value for EOS Mainnet
	c.hammer = dhammer.NewHammer(1, 8, c.executeDecodingJob)
	c.hammer.Start(ctx)

	// This is going to drain continuously hammer so new jobs can execute
	c.hammerConsumerWg.Add(1)
	go c.consumeCompletedDecodingJob(blockNum)

	return nil
}

func (c *ABIDecoder) endBlock(blockRef bstream.BlockRef) error {
	zlog.Debug("ending active block", zap.Stringer("block", blockRef))
	if c.activeBlockNum == noActiveBlockNum {
		return fmt.Errorf("end block for block %s received while no active block present", blockRef)
	}

	zlog.Debug("waiting for all transaction processing call that feeds hammer to complete")
	c.hammerFeederWg.Wait()

	zlog.Debug("waiting for hammer to complete all inflight requests")
	c.hammer.Close()
	c.hammerConsumerWg.Wait()

	if c.hammer.Err() != nil {
		return c.hammer.Err()
	}

	c.activeBlockNum = noActiveBlockNum
	c.lastSeenBlockRef = blockRef
	c.hammer = nil

	return nil
}

func (c *ABIDecoder) processTransaction(trxTrace *pbcodec.TransactionTrace) error {
	// Optimization: The truncation and ABI addition just below could share the same
	//               write lock. In the current code form, the lock is acquired/released
	//               twice. We could make them together but it adds a fair amount of logic
	//               because we don't want to lock if we don't really have to. So maybe later.
	if c.truncateOnNextGlobalSequence {
		// It's possible that no sequence number is found. The only case possible is if
		// the transaction did nothing or failed. In the failure case, we still need
		// to decode it, so we must not quit just yet.
		truncateAt, found := c.findFirstGlobalSequence(trxTrace)
		if found {
			c.truncateCache(truncateAt)
		}
	}

	abiOperations, err := c.extractABIOperations(trxTrace)
	if err != nil {
		return fmt.Errorf("unable to extract abis: %w", err)
	}

	// We only commit ABIs if the transaction was recored in the blockchain, failure is handled locally
	if len(abiOperations) > 0 && !trxTrace.HasBeenReverted() {
		if err := c.commitABIs(trxTrace.Id, abiOperations); err != nil {
			return fmt.Errorf("unable to commit abis: %w", err)
		}
	}

	// When a transaction fails, the ABIs cannot be committed since they were not recorded in the
	// blockchain. Instead, we build a local cache that will be passed to each decoding job.
	// The decoder will then lookup the local cache prior the global one to search for the correct
	// ABI.
	localCache := emptyCache
	if len(abiOperations) > 0 && trxTrace.HasBeenReverted() {
		localCache, err = c.createLocalABICache(trxTrace.Id, abiOperations)
		if err != nil {
			return fmt.Errorf("unable to create local abi cache: %w", err)
		}
	}

	// The wait group `Add` **must** be done before launching the go routine, otherwise, race conditions can happen
	c.hammerFeederWg.Add(1)
	go func() {
		zlog.Debug("abi decoding of transaction", zap.Uint64("block_num", c.activeBlockNum), zap.String("id", trxTrace.Id))
		defer c.hammerFeederWg.Done()

		// FIXME: Optimization: We could optimize notification inside a transaction. We could have a two-pass algorithm.
		//                      In the first pass we loop on all `non-notification` action, decoding them against the ABI.
		//                      In the second pass, we loop on all `notification` action this time and now instead of
		//                      decoding them, we find the action that created the notification and use it's already decoded
		//                      action. This would save us 2 decoding for each `eosio.token` for example.
		//
		//                      Now that we run that in parallel, two-pass it a little bit harder. Implementation wise, I
		//                      suggest we perform a final serialize phase in the `endBlock` method, after having done all
		//                      decoding jobs. This way, we are sure that all parent action are properly decoded.
		for i, actionTrace := range trxTrace.ActionTraces {
			if traceEnabled {
				zlog.Debug("adding action decoding job", zap.Int("action_index", i))
			}

			c.hammer.In <- actionDecodingJob{actionTrace, i, localCache}
		}
	}()

	// FIXME: Performed also for `dtrxOps` and `trxOps`
	// FIXME: How about `dbOps`, do we check them right now?

	return nil
}

func (c *ABIDecoder) findFirstGlobalSequence(trxTrace *pbcodec.TransactionTrace) (uint64, bool) {
	if trxTrace.HasBeenReverted() || len(trxTrace.ActionTraces) <= 0 {
		return 0, false
	}

	return trxTrace.ActionTraces[0].Receipt.GlobalSequence, true
}

type abiOperation struct {
	account        string
	actionIndex    int
	globalSequence uint64
	abi            *eos.ABI
}

func (c *ABIDecoder) commitABIs(trxID string, operations []abiOperation) error {
	zlog.Debug("updating cache with abis from transaction", zap.String("id", trxID), zap.Uint64("block_num", c.activeBlockNum), zap.Int("abi_count", len(operations)))
	c.cache.Lock()
	defer c.cache.Unlock()

	for _, operation := range operations {
		if err := c.cache.addABI(operation.account, operation.globalSequence, operation.abi); err != nil {
			return fmt.Errorf("failed to add ABI in action trace at index %d in transaction %s: %w", operation.actionIndex, trxID, err)
		}
	}

	return nil
}

func (c *ABIDecoder) createLocalABICache(trxID string, operations []abiOperation) (*ABICache, error) {
	zlog.Debug("creating local abi cache from transaction", zap.String("id", trxID), zap.Uint64("block_num", c.activeBlockNum))

	abiCache := newABICache()
	for _, operation := range operations {
		if err := abiCache.addABI(operation.account, operation.globalSequence, operation.abi); err != nil {
			return nil, fmt.Errorf("failed to add local ABI in action trace at index %d in transaction %s: %w", operation.actionIndex, trxID, err)
		}
	}

	return abiCache, nil
}

func (c *ABIDecoder) extractABIOperations(trxTrace *pbcodec.TransactionTrace) (out []abiOperation, err error) {
	for i, actionTrace := range trxTrace.ActionTraces {
		if actionTrace.FullName() == "eosio:eosio:setabi" {
			setABI := &system.SetABI{}
			err := eos.UnmarshalBinary(actionTrace.Action.RawData, setABI)
			if err != nil {
				return nil, fmt.Errorf("unable to read action trace 'setabi' at index %d in transaction %s: %w", i, trxTrace.Id, err)
			}

			// All sort of garbage can be in this field, skip if we cannot properly decode to an eos.ABI object
			abi := &eos.ABI{}
			err = eos.UnmarshalBinary(setABI.ABI, abi)
			if err != nil {
				zlog.Info("skipping action trace 'setabi' since abi content cannot be unmarshalled correctly", zap.Int("action_index", i), zap.String("trx_id", trxTrace.Id))
				continue
			}

			fmt.Printf("actiontrace not nil %t, receipt %t\n", actionTrace != nil, actionTrace.Receipt != nil)
			out = append(out, abiOperation{string(setABI.Account), i, actionTrace.Receipt.GlobalSequence, abi})
		}
	}

	return out, nil
}

func (c *ABIDecoder) truncateCache(truncateAt uint64) {
	zlog.Debug("truncating abi cache", zap.Uint64("truncate_at", truncateAt), zap.Uint64("block_num", c.activeBlockNum))
	c.cache.Lock()
	defer c.cache.Unlock()

	c.cache.truncateAfterOrEqualTo(truncateAt)

	c.truncateOnNextGlobalSequence = false
}

var actionJobTypeResult = []interface{}{"action"}

func (c *ABIDecoder) executeDecodingJob(ctx context.Context, batch []interface{}) ([]interface{}, error) {
	if len(batch) != 1 {
		return nil, fmt.Errorf("expecting batch to have a single element, got %d", len(batch))
	}

	if traceEnabled {
		zlog.Debug("executing decoding job", zap.String("type", fmt.Sprintf("%T", batch[0])))
	}

	switch v := batch[0].(type) {
	case actionDecodingJob:
		return actionJobTypeResult, c.decodeActionTrace(&v)
	}

	return nil, fmt.Errorf("unknown decoding job type %T", batch[0])
}

func (c *ABIDecoder) consumeCompletedDecodingJob(blockNum uint64) {
	zlog.Debug("consume completed decoding job starting", zap.Uint64("block_num", blockNum))
	defer func() {
		zlog.Debug("consuming completed decoding job terminated", zap.Uint64("block_num", blockNum))
		c.hammerConsumerWg.Done()
	}()

	for {
		select {
		case jobType, ok := <-c.hammer.Out:
			if !ok {
				zlog.Debug("hammer closed for block", zap.Uint64("block_num", blockNum))
				return
			}

			if traceEnabled {
				zlog.Debug("hammer job completed", zap.String("job_type", jobType.(string)))
			}
		}
	}
}

func (c *ABIDecoder) decodeActionTrace(job *actionDecodingJob) error {
	actionTrace := job.actionTrace

	globalSequence := uint64(math.MaxUint64)
	if actionTrace.Receipt != nil && actionTrace.Receipt.GlobalSequence != 0 {
		globalSequence = actionTrace.Receipt.GlobalSequence
	}

	if traceEnabled {
		zlog.Debug("abi decoding action trace", zap.Int("action_index", job.actionIndex), zap.Uint64("global_sequence", globalSequence))
	}

	action := actionTrace.Action
	if len(action.RawData) <= 0 {
		if traceEnabled {
			zlog.Debug("skipping action since no hex data found", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	abi := c.findABI(action.Account, globalSequence, job.localCache)
	if abi == nil {
		if traceEnabled {
			zlog.Debug("skipping action since no ABI found for it", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	actionDef := abi.ActionForName(eos.ActionName(action.Name))
	if actionDef == nil {
		if traceEnabled {
			zlog.Debug("skipping action since action was not in ABI", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	if traceEnabled {
		zlog.Debug("found ABI and action definition, performing decoding", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
	}

	decoder := eos.NewDecoder(action.RawData)
	jsonData, err := abi.Decode(decoder, actionDef.Type)
	if err != nil {
		return err
	}

	action.JsonData = string(jsonData)

	// FIXME: Do we need to keep both here? I'm not sure, reading `eos_to_proto` did not give me the final answer (it coded that late!)
	action.RawData = nil

	return nil
}

func (c *ABIDecoder) findABI(contract string, globalSequence uint64, localCache *ABICache) *eos.ABI {
	if localCache != emptyCache {
		localCache.RLock()
		defer localCache.RUnlock()

		abi := localCache.findABI(contract, globalSequence)
		if abi != nil {
			return abi
		}
	}

	c.cache.RLock()
	defer c.cache.RUnlock()

	return c.cache.findABI(contract, globalSequence)
}
