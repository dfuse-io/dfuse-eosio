package accounthist

import (
	"time"

	"go.uber.org/zap"
)

type blockBatchMetrics struct {
	batchStartTime            time.Time
	blockCount                uint64
	actionCount               int
	accountCacheMiss          uint64
	accountCacheHit           uint64
	totalReadSeqDuration      time.Duration
	readSeqCallCount          uint64
	totalReadMaxEntryDuration time.Duration
	readMaxEntryCallCount     uint64
}

func (m blockBatchMetrics) dump() (out []zap.Field) {
	out = append(out, []zap.Field{
		zap.Duration("processed_blocks_duration", time.Since(m.batchStartTime)),
		zap.Float64("block_rate", float64(m.blockCount)/(float64(time.Since(m.batchStartTime))/float64(time.Second))),
		zap.Int("action_count", m.actionCount),
		zap.Uint64("cache_miss", m.accountCacheMiss),
		zap.Uint64("cache_hit", m.accountCacheHit),
	}...)

	if m.readSeqCallCount > 0 {
		out = append(out, []zap.Field{
			zap.Duration("total_read_seq_duration", m.totalReadSeqDuration),
			zap.Duration("avg_read_seq_duration", m.totalReadSeqDuration/time.Duration(m.readSeqCallCount)),
			zap.Uint64("read_seq_call_count", m.readSeqCallCount),
		}...)
	}

	if m.readMaxEntryCallCount > 0 {
		out = append(out, []zap.Field{
			zap.Duration("total_max_entry_seq_duration", m.totalReadMaxEntryDuration),
			zap.Duration("avg_max_entry_duration", m.totalReadMaxEntryDuration/time.Duration(m.readMaxEntryCallCount)),
			zap.Uint64("read_max_entry_count", m.readMaxEntryCallCount),
		}...)
	}
	return out

}
