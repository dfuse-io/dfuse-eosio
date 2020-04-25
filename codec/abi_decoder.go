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
	"time"

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
	action *pbcodec.ActionTrace
	index  int
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

	if err := c.addABIsFromTransaction(trxTrace); err != nil {
		return fmt.Errorf("unable to update abis from transaction: %w", err)
	}

	// The wait group `Add` **must** be done before launching the go routine, otherwise, race conditions can happen
	c.hammerFeederWg.Add(1)
	go func() {
		zlog.Debug("abi decoding transaction traces", zap.Uint64("block_num", c.activeBlockNum), zap.String("id", trxTrace.Id))
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

			c.hammer.In <- actionDecodingJob{actionTrace, i}
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

func (c *ABIDecoder) addABIsFromTransaction(trxTrace *pbcodec.TransactionTrace) error {
	zlog.Debug("adding abis from transaction", zap.String("id", trxTrace.Id), zap.Uint64("block_num", c.activeBlockNum))

	if trxTrace.HasBeenReverted() {
		zlog.Debug("skipping transaction since it was reverted")
		return nil
	}

	// FIXME: Add support for failed_dtrx_trace, think about the correct meaning. Answers the
	//        following questions/use cases:
	//        - Assumes dtrx that fails with 3 actions in it. Action@450 (setabi) Action@451 (data with new ABI) Action@0 (fails)
	//          We are currently building the full cache for the block, does it mean we cannot do it? Maybe we should only accumulated
	//          committed block state and for failure causes, we resolve in the transaction trace it self?.
	//        - Think and test weird case that a `eosio:setabi` is called in a successufl `onerror` handler.
	//
	//        One important thing to note, the failed deferred transaction will always be followed by
	//        an `onerror` handler. Both could be in failure state. In the original failure, no abi should be
	//        comitted and we need to deal with the setabi only within the transaction. While in the onerror,
	//        it could have committed some `setabi`.

	var abiOperations []abiOperation
	for i, actionTrace := range trxTrace.ActionTraces {
		if actionTrace.FullName() == "eosio:eosio:setabi" {
			setABI := &system.SetABI{}
			err := eos.UnmarshalBinary(actionTrace.Action.RawData, setABI)
			if err != nil {
				return fmt.Errorf("unable to read action trace 'setabi' at index %d in transaction %s: %w", i, trxTrace.Id, err)
			}

			// All sort of garbage can be in this field, skip if we cannot properly decode to an eos.ABI object
			abi := &eos.ABI{}
			err = eos.UnmarshalBinary(setABI.ABI, abi)
			if err != nil {
				zlog.Info("skipping action trace 'setabi' since abi content cannot be unmarshalled correctly", zap.Int("action_index", i), zap.String("trx_id", trxTrace.Id))
				continue
			}

			fmt.Printf("actiontrace not nil %t, receipt %t\n", actionTrace != nil, actionTrace.Receipt != nil)
			abiOperations = append(abiOperations, abiOperation{string(setABI.Account), i, actionTrace.Receipt.GlobalSequence, abi})
		}
	}

	if len(abiOperations) <= 0 {
		return nil
	}

	zlog.Debug("updating cache with abis from transaction", zap.String("id", trxTrace.Id), zap.Uint64("block_num", c.activeBlockNum))
	c.cache.Lock()
	defer c.cache.Unlock()

	for _, operation := range abiOperations {
		if err := c.cache.addABI(operation.account, operation.globalSequence, operation.abi); err != nil {
			return fmt.Errorf("failed to add ABI in action trace at index %d in transaction %s: %w", operation.actionIndex, trxTrace.Id, err)
		}
	}

	return nil
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
		return actionJobTypeResult, c.decodeActionTrace(v.action, v.index)
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

func (c *ABIDecoder) decodeActionTrace(actionTrace *pbcodec.ActionTrace, actionIndex int) error {
	globalSequence := uint64(math.MaxUint64)
	if actionTrace.Receipt != nil && actionTrace.Receipt.GlobalSequence != 0 {
		globalSequence = actionTrace.Receipt.GlobalSequence
	}

	if traceEnabled {
		zlog.Debug("abi decoding action trace", zap.Int("action_index", actionIndex), zap.Uint64("global_sequence", globalSequence))
	}

	action := actionTrace.Action
	if len(action.RawData) <= 0 {
		if traceEnabled {
			zlog.Debug("skipping action since no hex data found", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	c.cache.RLock()
	defer c.cache.RUnlock()

	abi := c.cache.findABI(action.Account, globalSequence)
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

type ABICache struct {
	sync.RWMutex

	// Represents the actual cache information. The map structure is for each `contract`, keep a mapping
	// of `globalSequenceNumber` to `ABI` (i.e. `map[<contract>]map[<globalSequenceNumber>]<ABI>`). Here
	// an actual map content example to get a better idea.
	//
	// ```
	// {
	// 	"eosio": {
	// 		4000: `ABI #3`,
	// 		0: `ABI #1`,
	// 		100: `ABI #2`,
	// 	},
	// 	"eosio.token": {
	// 		0: `ABI #1`,
	// 		1000: `ABI #2`,
	// 	}
	// }
	// ```
	//
	// **Important** The second inner map is un-ordered, to retrieve correct ABI based on sequential
	//               ordering, you must use `abisOrdering` element.
	abis map[string]map[uint64]*eos.ABI

	// Represents the ABIs ordering values based on `globalSequence`. The map structure is for each
	// `contract`, keep a slice of ordered ABI global sequence number
	// (i.e. `map[<contract>][<index>]<globalSequenceNumber>`). By using this ordering structure, we
	// can inside the `[]uint64` perfrom a binary search to find the correct `<globalSequenceNumber>`
	// then retrieve the corresponding ABI in `abis` element.
	abisOrdering map[string][]uint64
}

func newABICache() *ABICache {
	return &ABICache{
		abis:         map[string]map[uint64]*eos.ABI{},
		abisOrdering: map[string][]uint64{},
	}
}

// addABI adds the ABI to cache assuming it follows the latest stored ABI for this
// contract. For example, assuming a series of ABI for which the latest ABI
// change was peformed at global sequence #450, then it's assumed that the receive `globalSequence`
// argument is greater than 450.
//
// If the invariant is not respected, an error is returned.
func (c *ABICache) addABI(contract string, globalSequence uint64, abi *eos.ABI) error {
	zlog.Debug("adding new abi", zap.String("account", contract), zap.Uint64("global_sequence", globalSequence))

	contractOrdering, found := c.abisOrdering[contract]
	if found && len(contractOrdering) > 0 && contractOrdering[len(contractOrdering)-1] > globalSequence {
		return fmt.Errorf("abi is not sequential against latest ABI's global sequence, latest is %d and trying to add %d which is in the past", contractOrdering[len(contractOrdering)-1], globalSequence)
	}

	contractAbis, found := c.abis[contract]
	if !found {
		contractAbis = map[uint64]*eos.ABI{}
		c.abis[contract] = contractAbis
	}

	contractAbis[globalSequence] = abi
	c.abisOrdering[contract] = append(contractOrdering, globalSequence)

	return nil
}

// findABI for the given `contract` at which `globalSequence` was the most
// recent active ABI.
func (c *ABICache) findABI(contract string, globalSequence uint64) *eos.ABI {
	contractOrdering := c.abisOrdering[contract]
	if len(contractOrdering) <= 0 {
		return nil
	}

	// Walk the active global sequence in reverse order, and pick the first one that was
	// set before the request `globalSequence` (`x <= globalSequence`) but never set after.
	for i := len(contractOrdering) - 1; i >= 0; i-- {
		activeGlobalSequence := contractOrdering[i]
		if activeGlobalSequence <= globalSequence {
			return c.abis[contract][activeGlobalSequence]
		}
	}

	return nil
}

func (c *ABICache) truncateAfterOrEqualTo(globalSequence uint64) {
	startTime := time.Now()
	removedCount := 0

	for contract, contractOrdering := range c.abisOrdering {
		if len(contractOrdering) <= 0 {
			continue
		}

		pivot, preservedSet, cutSet := truncateAfterOrEqual(contractOrdering, globalSequence)
		if traceEnabled {
			zlog.Debug("truncating contract abi",
				zap.String("contract", contract),
				zap.Int("pivot", pivot),
				zap.Int("preserved_count", len(preservedSet)),
				zap.Int("cut_count", len(cutSet)),
			)
		}

		if len(cutSet) <= 0 {
			continue
		}

		contractAbis := c.abis[contract]
		for _, removedGlobalSequence := range cutSet {
			delete(contractAbis, removedGlobalSequence)
		}

		if len(contractAbis) <= 0 {
			delete(c.abis, contract)
		}

		if len(preservedSet) <= 0 {
			delete(c.abisOrdering, contract)
		} else {
			c.abisOrdering[contract] = preservedSet
		}

		removedCount += len(cutSet)
	}

	zlog.Debug("completed cache truncation",
		zap.Duration("elapsed", time.Since(startTime)),
		zap.Int("removed_abi", removedCount),
		zap.Uint64("truncated_at", globalSequence),
	)
}

func truncateAfterOrEqual(slice []uint64, element uint64) (pivot int, preservedSet, cutSet []uint64) {
	pivot = -1
	count := len(slice)
	if count <= 0 {
		return pivot, nil, nil
	}

	for i := count - 1; i >= 0; i-- {
		if slice[i] < element {
			pivot = i
			break
		}
	}

	// Every elements were before the searched element, everything must be preserved
	if pivot == count-1 {
		return pivot, slice, nil
	}

	// Every elements were after or equal to the searched element, everything must be removed
	if pivot == -1 {
		return pivot, nil, slice
	}

	return pivot, slice[:pivot+1], slice[pivot+1:]
}
