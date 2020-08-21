package accounthist

import (
	"encoding/binary"

	"github.com/eoscanada/eos-go"
)

const (
	//prefixSequenceNumber = byte(0x01) // unused now
	prefixAction    = byte(0x02)
	prefixLastBlock = byte(0x03)

	//sequenceKeyLen     = 9
	actionPrefixKeyLen = 9
	actionKeyLen       = 17
	lastBlockKeyLen    = 9
)

func encodeActionPrefixKey(key []byte, account string) {
	_ = key[actionPrefixKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], encodeAccountName(account))
}

func encodeActionKey(key []byte, account string, sequenceNumber uint64) {
	_ = key[actionKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], encodeAccountName(account))
	binary.BigEndian.PutUint64(key[9:], ^sequenceNumber)
}

func decodeActionKeySeqNum(key []byte) uint64 {
	_ = key[actionKeyLen-1] //bounds check

	//binary.LittleEndian.ReadUint64(key[1:], encodeAccountName(account))
	seqNum := binary.BigEndian.Uint64(key[9:])
	return ^seqNum
}

func encodeLastProcessedBlockKey(key []byte) {
	_ = key[lastBlockKeyLen-1] //bounds check
	key[0] = prefixLastBlock
}

func encodeAccountName(account string) uint64 {
	return eos.MustStringToName(account)
}
