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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dhammer"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

var mostRecentActiveABI uint64 = math.MaxUint64
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
	cache *ABICache
	queue *decodingQueue

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

func newABIDecoder() *ABIDecoder {
	return &ABIDecoder{
		cache:            newABICache(),
		activeBlockNum:   noActiveBlockNum,
		lastSeenBlockRef: bstream.BlockRefEmpty,
	}
}

func (c *ABIDecoder) resetCache() {
	c.cache = newABICache()
}

func (c *ABIDecoder) addInitialABI(contract string, b64ABI string) error {
	rawABI, err := base64.StdEncoding.DecodeString(b64ABI)
	if err != nil {
		return fmt.Errorf("unable to decode ABI hex data for contract %s: %w", contract, err)
	}

	abi := &eos.ABI{}
	abi.SetFitNodeos(true)

	err = eos.UnmarshalBinary(rawABI, abi)
	if err != nil {
		zlog.Info("skipping initial ABI since content cannot be unmarshalled correctly", zap.String("contract", contract))
		return nil
	}

	c.cache.Lock()
	defer c.cache.Unlock()

	return c.cache.addABI(contract, 0, abi)
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

	c.queue = newDecodingQueue(ctx, blockNum, c.cache)

	return nil
}

func (c *ABIDecoder) endBlock(block *pbcodec.Block) error {
	blockRef := block.AsRef()
	zlog.Debug("post-processing block", zap.Stringer("block", blockRef))
	if c.activeBlockNum == noActiveBlockNum {
		return fmt.Errorf("end block for block %s received while no active block present", block)
	}

	zlog.Debug("processing implicit transactions", zap.Int("trx_op_count", len(block.UnfilteredImplicitTransactionOps)))
	err := c.processImplicitTransactions(block.UnfilteredImplicitTransactionOps)
	if err != nil {
		return fmt.Errorf("unable to process implicit transactions: %w", err)
	}

	zlog.Debug("waiting for decoding queue to drain completely")
	err = c.queue.drain()
	if err != nil {
		return fmt.Errorf("unable to correctly drain decoding queue: %w", err)
	}

	zlog.Debug("ending block processing", zap.String("block", block.Id))
	c.activeBlockNum = noActiveBlockNum
	c.lastSeenBlockRef = blockRef
	c.queue = nil

	return nil
}

func (c *ABIDecoder) processTransaction(trxTrace *pbcodec.TransactionTrace) error {
	zlog.Debug("processing transaction for decoding", zap.String("trx_id", trxTrace.Id))

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

	var decodingJobs []decodingJob

	// FIXME: Optimization: We could optimize notification inside a transaction. We could have a two-pass algorithm.
	//                      In the first pass we loop on all `non-notification` action, decoding them against the ABI.
	//                      In the second pass, we loop on all `notification` action this time and now instead of
	//                      decoding them, we find the action that created the notification and use it's already decoded
	//                      action. This would save us 2 decoding for each `eosio.token` for example.
	//
	//                      Now that we run that in parallel, two-pass it a little bit harder. Implementation wise, I
	//                      suggest we perform a final serialize phase in the `endBlock` method, after having done all
	//                      decoding jobs. This way, we are sure that all parent action are properly decoded.
	for _, actionTrace := range trxTrace.ActionTraces {
		globalSequence := actionTraceGlobalSequence(actionTrace)

		decodingJobs = append(decodingJobs, actionDecodingJob{actionTrace.Action, c.activeBlockNum, trxTrace.Id, globalSequence, localCache})
	}

	for _, dtrxOp := range trxTrace.DtrxOps {
		// A deferred transaction push in the blockchain (using CLI and `--delay-sec`) does not have any action trace,
		// let's use the most recent active ABI global sequence in those cases.
		globalSequence := mostRecentActiveABI
		if dtrxOp.Operation != pbcodec.DTrxOp_OPERATION_PUSH_CREATE {
			globalSequence = actionTraceGlobalSequence(trxTrace.ActionTraces[dtrxOp.ActionIndex])
		}

		for _, action := range dtrxOp.Transaction.Transaction.ContextFreeActions {
			decodingJobs = append(decodingJobs, dtrxDecodingJob{actionDecodingJob{action, c.activeBlockNum, trxTrace.Id, globalSequence, localCache}})
		}

		for _, action := range dtrxOp.Transaction.Transaction.Actions {
			decodingJobs = append(decodingJobs, dtrxDecodingJob{actionDecodingJob{action, c.activeBlockNum, trxTrace.Id, globalSequence, localCache}})
		}
	}

	zlog.Debug("queuing transaction trace decoding jobs", zap.Uint64("block_num", c.activeBlockNum), zap.String("id", trxTrace.Id), zap.Int("job_count", len(decodingJobs)))
	return c.queue.addJobs(decodingJobs)
}

func (c *ABIDecoder) processImplicitTransactions(trxOps []*pbcodec.TrxOp) error {
	var decodingJobs []decodingJob

	for _, trxOp := range trxOps {
		for _, action := range trxOp.Transaction.Transaction.ContextFreeActions {
			decodingJobs = append(decodingJobs, dtrxDecodingJob{actionDecodingJob{action, c.activeBlockNum, trxOp.TransactionId, mostRecentActiveABI, emptyCache}})
		}

		for _, action := range trxOp.Transaction.Transaction.Actions {
			decodingJobs = append(decodingJobs, dtrxDecodingJob{actionDecodingJob{action, c.activeBlockNum, trxOp.TransactionId, mostRecentActiveABI, emptyCache}})
		}
	}

	zlog.Debug("queuing implicit transactions decoding jobs", zap.Uint64("block_num", c.activeBlockNum), zap.Int("job_count", len(decodingJobs)))
	return c.queue.addJobs(decodingJobs)
}

func actionTraceGlobalSequence(actionTrace *pbcodec.ActionTrace) uint64 {
	globalSequence := mostRecentActiveABI
	if actionTrace.Receipt != nil && actionTrace.Receipt.GlobalSequence != 0 {
		globalSequence = actionTrace.Receipt.GlobalSequence
	}

	return globalSequence
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
		// If the action trace receipt is `nil`, it means the action failed, in which case, we don't care about those `setabi`
		if actionTrace.FullName() == "eosio:eosio:setabi" && actionTrace.Receipt != nil {
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
			abi.SetFitNodeos(true)
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

type decodingQueue struct {
	blockNum    uint64
	globalCache *ABICache
	hammer      *dhammer.Hammer
}

type decodingJob interface {
	blockNum() uint64
	kind() string
	trxID() string
}

func (j actionDecodingJob) blockNum() uint64 { return j.actualblockNum }
func (j dtrxDecodingJob) blockNum() uint64   { return j.actionDecodingJob.actualblockNum }

func (j actionDecodingJob) trxID() string { return j.actualTrxID }
func (j dtrxDecodingJob) trxID() string   { return j.actionDecodingJob.actualTrxID }

func (j actionDecodingJob) kind() string { return "action" }
func (j dtrxDecodingJob) kind() string   { return "dtrx" }

type actionDecodingJob struct {
	action         *pbcodec.Action
	actualblockNum uint64
	actualTrxID    string
	globalSequence uint64
	localCache     *ABICache
}

type dtrxDecodingJob struct {
	actionDecodingJob
}

func newDecodingQueue(ctx context.Context, blockNum uint64, globalCache *ABICache) *decodingQueue {
	queue := &decodingQueue{
		blockNum:    blockNum,
		globalCache: globalCache,
	}

	// FIXME: Replace 8 with as many CPUs available minus one (or two) we have. Will need some profiling to see the best value for EOS Mainnet
	queue.hammer = dhammer.NewHammer(1, 8, queue.executeDecodingJob, dhammer.SetInChanSize(10000))
	queue.hammer.Start(ctx)

	go queue.drainQueueFully(blockNum)

	return queue
}

func (q *decodingQueue) addJobs(jobs []decodingJob) error {
	for _, job := range jobs {
		if traceEnabled {
			zlog.Debug("adding decoding job to queue", zap.String("kind", job.kind()))
		}

		select {
		case <-q.hammer.Terminating():
			zlog.Debug("decoding queue hammer terminating, stopping queuer routine")
			return fmt.Errorf("unable to add job, hammer is terminating: %w", q.hammer.Err())
		case q.hammer.In <- job:
		}
	}

	return nil
}

func (q *decodingQueue) drain() error {
	zlog.Debug("closing dhammer")
	q.hammer.Close()

	zlog.Debug("waiting for dhammer termination")
	<-q.hammer.Terminated()
	if q.hammer.Err() != nil {
		return fmt.Errorf("dhammer unexpected termination: %w", q.hammer.Err())
	}

	if len(q.hammer.In) != 0 || len(q.hammer.Out) != 0 {
		return fmt.Errorf("dhammer terminated without being fully drained, still %d elements in In and %d elements in Out", len(q.hammer.In), len(q.hammer.Out))
	}

	zlog.Debug("dhammer terminated")
	return nil
}

func (c *decodingQueue) executeDecodingJob(ctx context.Context, batch []interface{}) ([]interface{}, error) {
	if len(batch) != 1 {
		return nil, fmt.Errorf("expecting batch to have a single element, got %d", len(batch))
	}

	job := batch[0].(decodingJob)
	if traceEnabled {
		zlog.Debug("executing decoding job", zap.String("kind", job.kind()))
	}

	if job.blockNum() != c.blockNum {
		return nil, fmt.Errorf("trying to decode a job for block num %d while decoding queue block num is %d", job.blockNum(), c.blockNum)
	}

	switch v := batch[0].(type) {
	case actionDecodingJob:
		return []interface{}{job.kind()}, c.decodeAction(v.action, v.globalSequence, job.trxID(), job.blockNum(), v.localCache)
	case dtrxDecodingJob:
		return []interface{}{job.kind()}, c.decodeAction(v.action, v.globalSequence, job.trxID(), job.blockNum(), v.localCache)
	default:
		return nil, fmt.Errorf("unknown decoding job kind %s", job.kind())
	}
}

func (q *decodingQueue) drainQueueFully(blockNum uint64) {
	zlog.Debug("queue drainer routine started", zap.Uint64("block_num", blockNum))
	defer func() {
		zlog.Debug("queue drainer routine terminated", zap.Uint64("block_num", blockNum))
	}()

	for {
		select {
		case jobKind, ok := <-q.hammer.Out:
			if !ok {
				zlog.Debug("queue is now closed", zap.Uint64("block_num", blockNum))
				return
			}

			if traceEnabled {
				zlog.Debug("queue job completed", zap.String("kind", jobKind.(string)))
			}
		}
	}
}

func (q *decodingQueue) decodeAction(action *pbcodec.Action, globalSequence uint64, trxID string, blockNum uint64, localCache *ABICache) error {
	if traceEnabled {
		zlog.Debug("decoding action", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
	}

	if len(action.RawData) <= 0 {
		if traceEnabled {
			zlog.Debug("skipping action since no hex data found", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	// Transfer raw data min length is 8 bytes for `from`, 8 bytes for `to`, 16 bytes for `quantity` and `1 byte` for memo length
	if action.Account == "eosio.token" && action.Name == "transfer" && len(action.RawData) >= 33 {
		if traceEnabled {
			zlog.Debug("decoding action using pre-built eosio.token:transfer decoder", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
		}

		var err error
		action.JsonData, err = decodeTransfer(action.RawData)
		if err != nil {
			zlog.Error("skipping transfer action since we were not able to decode it against ABI",
				zap.Uint64("block_num", blockNum),
				zap.String("trx_id", trxID),
				zap.String("action", action.SimpleName()),
				zap.Uint64("global_sequence", globalSequence),
				zap.Error(err),
			)
			return nil
		}

		return nil
	}

	abi := q.findABI(action.Account, globalSequence, localCache)
	if abi == nil {
		if traceEnabled {
			zlog.Debug("skipping action since no ABI found for it", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	actionDef := abi.ActionForName(eos.ActionName(action.Name))
	if actionDef == nil {
		if traceEnabled {
			zlog.Debug("skipping action since action was not in ABI", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
		}
		return nil
	}

	if traceEnabled {
		zlog.Debug("found ABI and action definition, performing decoding", zap.String("action", action.SimpleName()), zap.Uint64("global_sequence", globalSequence))
	}

	decoder := eos.NewDecoder(action.RawData)
	jsonData, err := abi.Decode(decoder, actionDef.Type)
	if err != nil {
		// Sadly, anything can make up the hex data of an action, so it could be pure garbage that does not fit against
		// the ABI. Even more, the ABI iteself while valid in structure could be wrong, image a struct field type that point to
		// an alias that itself is not defined in the ABI. This would prevent decoding. We simply cannot error out here, it must
		// be logged and we must track the difference somehow.
		//
		// FIXME: Probably that logging an error is too much, it's being done like this for now while we
		//        tweak. Will probably move to INFO (depending on occurrences) or DEBUG.
		zlog.Error("skipping action since we were not able to decode it against ABI",
			zap.Uint64("block_num", blockNum),
			zap.String("trx_id", trxID),
			zap.String("action", action.SimpleName()),
			zap.Uint64("global_sequence", globalSequence),
			zap.Error(err),
		)
		return nil
	}

	action.JsonData = string(jsonData)

	return nil
}

func (q *decodingQueue) findABI(contract string, globalSequence uint64, localCache *ABICache) *eos.ABI {
	if localCache != emptyCache {
		localCache.RLock()
		defer localCache.RUnlock()

		abi := localCache.findABI(contract, globalSequence)
		if abi != nil {
			return abi
		}
	}

	q.globalCache.RLock()
	defer q.globalCache.RUnlock()

	return q.globalCache.findABI(contract, globalSequence)
}

type eosioTokenTransfer struct {
	From     eos.Name  `json:"from"`
	To       eos.Name  `json:"to"`
	Quantity eos.Asset `json:"quantity"`
	Memo     string    `json:"memo"`
}

func decodeTransfer(data []byte) (string, error) {
	decoder := eos.NewDecoder(data)
	from, err := decoder.ReadName()
	if err != nil {
		return "", fmt.Errorf(`unable to read transfer field "from": %w`, err)
	}

	to, err := decoder.ReadName()
	if err != nil {
		return "", fmt.Errorf(`unable to read transfer field "to": %w`, err)
	}

	quantity, err := decoder.ReadAsset()
	if err != nil {
		return "", fmt.Errorf(`unable to read transfer field "quantity": %w`, err)
	}

	memo, err := decoder.ReadString()
	if err != nil {
		return "", fmt.Errorf(`unable to read transfer field "memo": %w`, err)
	}

	serialized, err := json.Marshal(&eosioTokenTransfer{
		From:     from,
		To:       to,
		Quantity: quantity,
		Memo:     memo,
	})
	if err != nil {
		return "", fmt.Errorf(`unable to Marshal transfer to JSON: %w`, err)
	}

	return string(serialized), nil
}
