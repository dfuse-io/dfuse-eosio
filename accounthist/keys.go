package accounthist

import (
	"encoding/binary"
)

const (
	//prefixSequenceNumber = byte(0x01) // unused now
	prefixAction    = byte(0x02)
	prefixLastBlock = byte(0x03)

	actionPrefixKeyLen = 9
	actionKeyLen       = 18
	lastBlockKeyLen    = 2
)

func encodeActionPrefixKey(key []byte, account uint64) {
	_ = key[actionPrefixKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], account)
}

func encodeActionKey(key []byte, account uint64, shardNum byte, sequenceNumber uint64) {
	_ = key[actionKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], account)
	key[9] = ^shardNum
	binary.BigEndian.PutUint64(key[10:], ^sequenceNumber)
}

func decodeActionKeySeqNum(key []byte) (byte, uint64) {
	_ = key[actionKeyLen-1] //bounds check

	shardNum := key[9]
	seqNum := binary.BigEndian.Uint64(key[10:])
	return ^shardNum, ^seqNum
}

func encodeLastProcessedBlockKey(key []byte, shardNum byte) {
	_ = key[lastBlockKeyLen-1] //bounds check
	key[0] = prefixLastBlock
	key[1] = shardNum
}
