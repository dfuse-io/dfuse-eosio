package wallet

import (
	"encoding/binary"
)

const (
	prefixSequenceNumber = byte(0x01)
	prefixAction         = byte(0x02)
	prefixLastBlock      = byte(0x03)

	sequenceKeyLen     = 9
	actionPrefixKeyLen = 9
	actionKeyLen       = 17
	lastBlockKeyLen    = 4
)

func encodeSequenceKey(key []byte, account string) {
	_ = key[sequenceKeyLen-1] //bounds check

	key[0] = prefixSequenceNumber
	binary.LittleEndian.PutUint64(key[1:9], encodeAccountName(account))
}

func encodeTransactionPrefixKey(key []byte, account string) {
	_ = key[actionPrefixKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:9], encodeAccountName(account))
}

func encodeTransactionKey(key []byte, account string, sequenceNumber uint64) {
	_ = key[actionKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:9], encodeAccountName(account))
	binary.LittleEndian.PutUint64(key[9:17], ^sequenceNumber)
}

func encodeLastProcessedBlockKey(key []byte) {
	_ = key[lastBlockKeyLen-1] //bounds check
	key[0] = prefixLastBlock
}

func encodeAccountName(account string) uint64 {
	// TODO: is this correct?
	return binary.LittleEndian.Uint64([]byte(account))
}
