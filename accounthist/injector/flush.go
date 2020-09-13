package injector

import (
	"context"
	"time"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

func (i *Injector) flush(ctx context.Context, blk *pbcodec.Block, lastInStreak bool) error {
	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	realtimeFlush := time.Since(blk.MustTime()) < 20*time.Minute && lastInStreak
	onFlushIntervalBoundary := blk.Num()%i.flushBlocksInterval == 0
	if realtimeFlush || onFlushIntervalBoundary {

		if i.lastWrittenBlock != nil {
			blocks := blk.Num() - i.lastWrittenBlock.blockNum
			timeDelta := time.Since(i.lastWrittenBlock.writtenAt)
			deltaInSeconds := float64(timeDelta) / float64(time.Second)
			blocksPerSec := float64(blocks) / deltaInSeconds
			zlog.Info("block throughput",
				zap.Float64("blocks_per_secs", blocksPerSec),
				zap.Uint64("last_written_block_num", i.lastWrittenBlock.blockNum),
				zap.Uint64("current_block_num", blk.Num()),
				zap.Time("last_written_block_at", i.lastWrittenBlock.writtenAt),
			)
		}
		i.lastWrittenBlock = &lastWrittenBlock{
			blockNum:  blk.Num(),
			writtenAt: time.Now(),
		}
		zlog.Info("starting force flush", zap.Uint64("block_num", blk.Num()))
		return i.ForceFlush(ctx)
	}
	return nil
}

func (i *Injector) ForceFlush(ctx context.Context) error {
	return i.KvStore.FlushPuts(ctx)
}
