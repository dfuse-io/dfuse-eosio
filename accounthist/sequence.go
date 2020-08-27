package accounthist

import (
	"go.uber.org/zap/zapcore"
)

const sequenceDataValueLength = 24

type sequenceData struct {
	historySeqNum uint64
	lastGlobalSeq uint64
}

func (sqd *sequenceData) Increment(globalSeq uint64) {
	sqd.historySeqNum++
	sqd.lastGlobalSeq = globalSeq
}

func (sqd *sequenceData) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("history_seq_num", sqd.historySeqNum)
	encoder.AddUint64("last_global_seq", sqd.lastGlobalSeq)
	return nil
}
