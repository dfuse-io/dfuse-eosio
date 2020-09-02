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

func encodeActionPrefixKey(account uint64) []byte {
	key := make([]byte, actionPrefixKeyLen)

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], account)
	return key
}

func encodeActionKey(account uint64, shardNum byte, ordinalNumber uint64) []byte {
	key := make([]byte, actionKeyLen)

	key[0] = prefixAction

	binary.LittleEndian.PutUint64(key[1:], account)

	// We want the rows to be sorted by shard ascending 0 -> n
	key[9] = shardNum
	binary.BigEndian.PutUint64(key[10:], ^ordinalNumber)

	return key
}

func decodeActionKeySeqNum(key []byte) (byte, uint64) {
	_ = key[actionKeyLen-1] //bounds check

	shardNum := key[9]
	seqNum := binary.BigEndian.Uint64(key[10:])
	return ^shardNum, ^seqNum
}

func encodeLastProcessedBlockKey(shardNum byte) []byte {
	key := make([]byte, lastBlockKeyLen)
	key[0] = prefixLastBlock
	key[1] = shardNum
	return key
}
