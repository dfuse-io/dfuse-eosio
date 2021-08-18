package injector

import (
	"context"
	"fmt"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

func (i *Injector) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	ctx := context.Background()
	block := blk.ToNative().(*pbcodec.Block)
	fObj := obj.(*forkable.ForkableObject)

	switch fObj.Step {
	case forkable.StepNew:

		// metrics
		if i.headBlockNumber != nil {
			i.headBlockNumber.SetUint64(block.Num())
		}
		if i.headBlockTimeDrift != nil {
			blkTime := block.MustTime()
			i.headBlockTimeDrift.SetBlockTime(blkTime)
		}

	case forkable.StepIrreversible:
		rawTraceMap := fObj.Obj.(map[uint64][]byte)
		isLastInStreak := fObj.StepIndex+1 == fObj.StepCount

		if i.stopBlockNum != 0 && blk.Num() >= i.stopBlockNum {
			zlog.Info("stop block num reached, flushing all writes",
				zap.Uint64("stop_block_num", i.stopBlockNum),
				zap.Uint64("block_num", blk.Num()),
			)
			if err := i.ForceFlush(ctx); err != nil {
				i.Shutdown(err)
				return fmt.Errorf("flushing when stopping: %w", err)
			}

			i.Shutdown(nil)
			return nil
		}
		for _, trxTrace := range block.TransactionTraces() {
			if trxTrace.HasBeenReverted() {
				continue
			}

			actionMatcher := block.FilteringActionMatcher(trxTrace)

			i.currentBatchMetrics.actionCount += len(trxTrace.ActionTraces)
			for _, act := range trxTrace.ActionTraces {
				if !actionMatcher.Matched(act.ExecutionIndex) || act.Receipt == nil {
					continue
				}

				err := i.processAction(ctx, blk, act, rawTraceMap)
				if err != nil {
					return err
				}
			}
		}

		if err := i.writeCheckpoint(ctx, block); err != nil {
			return fmt.Errorf("error while saving block checkpoint")
		}

		if err := i.flush(ctx, block, isLastInStreak); err != nil {
			return fmt.Errorf("error while flushing: %w", err)
		}

		i.currentBatchMetrics.blockCount++
		if (blk.Number % 1000) == 0 {
			opts := i.currentBatchMetrics.dump()
			opts = append(opts, []zap.Field{
				zap.String("block_id", block.Id),
				zap.Uint32("block_num", block.Number),
				zap.Int("cache_size", len(i.cacheSeqData)),
			}...)
			zlog.Info("processed blk 1/1000",
				opts...,
			)
			i.currentBatchMetrics = blockBatchMetrics{
				batchStartTime: time.Now(),
			}
		}
	}

	return nil
}
