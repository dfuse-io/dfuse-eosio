package wallet

import (
	"encoding/binary"

	"go.uber.org/zap/zapcore"
)

const sequenceDataValueLength = 24

type sequenceData struct {
	CurrentSequenceID   uint64
	PreviousSequenceID  uint64
	PreviousBlockNumber uint64
}

func (sqd *sequenceData) Increment() {
	sqd.CurrentSequenceID++
}

//set data ready to be saved once a block is processed
func (sqd *sequenceData) SetCheckpoint(blockNumber uint64) {
	sqd.PreviousSequenceID = sqd.CurrentSequenceID
	sqd.PreviousBlockNumber = blockNumber
}

func (sqd *sequenceData) Encode(bytes []byte) {
	_ = bytes[sequenceDataValueLength-1] // bounds check

	binary.LittleEndian.PutUint64(bytes[0:8], sqd.CurrentSequenceID)
	binary.LittleEndian.PutUint64(bytes[8:16], sqd.PreviousSequenceID)
	binary.LittleEndian.PutUint64(bytes[16:20], sqd.PreviousBlockNumber)
}

func (sqd *sequenceData) Decode(bytes []byte) {
	_ = bytes[sequenceDataValueLength-1] // bounds check

	sqd.CurrentSequenceID = binary.LittleEndian.Uint64(bytes[0:8])
	sqd.PreviousSequenceID = binary.LittleEndian.Uint64(bytes[8:16])
	sqd.PreviousBlockNumber = binary.LittleEndian.Uint64(bytes[16:20])
}

func (sqd *sequenceData) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("current sequence number", sqd.CurrentSequenceID)
	encoder.AddUint64("previous sequence number", sqd.PreviousSequenceID)
	encoder.AddUint64("previous block number", sqd.PreviousBlockNumber)
	return nil
}
