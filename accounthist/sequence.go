package accounthist

import (
	"go.uber.org/zap/zapcore"
)

const sequenceDataValueLength = 24

type sequenceData struct {
	historySeqNum  uint64 // while in memory, this value is the NEXT history seq num that should be attributed, in this shard.
	lastGlobalSeq  uint64 // taken from the top-most action stored in this shard
	lastDeletedSeq uint64 // taken from the top-most action stored in this shard
	maxEntries     uint64 // initialized with the process' max entries per account, but can be reduced if some more recent shards covered this account
}

func (sqd *sequenceData) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("history_seq_num", sqd.historySeqNum)
	encoder.AddUint64("last_global_seq", sqd.lastGlobalSeq)
	return nil
}
