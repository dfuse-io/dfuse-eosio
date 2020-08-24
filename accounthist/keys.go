package accounthist

import (
	"encoding/binary"

	"github.com/eoscanada/eos-go"
)

const (
	//prefixSequenceNumber = byte(0x01) // unused now
	prefixAction    = byte(0x02)
	prefixLastBlock = byte(0x03)

	actionPrefixKeyLen = 9
	actionKeyLen       = 18
	lastBlockKeyLen    = 10
)

func encodeActionPrefixKey(key []byte, account string) {
	_ = key[actionPrefixKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], encodeAccountName(account))
}

func encodeActionKey(key []byte, account string, shardNum byte, sequenceNumber uint64) {
	_ = key[actionKeyLen-1] //bounds check

	key[0] = prefixAction
	binary.LittleEndian.PutUint64(key[1:], encodeAccountName(account))
	key[9] = ^shardNum
	binary.BigEndian.PutUint64(key[10:], ^sequenceNumber)
}

func decodeActionKeySeqNum(key []byte) (byte, uint64) {
	_ = key[actionKeyLen-1] //bounds check

	//binary.LittleEndian.ReadUint64(key[1:], encodeAccountName(account))
	shardNum := key[9]
	seqNum := binary.BigEndian.Uint64(key[10:])
	return ^shardNum, ^seqNum
}

func encodeLastProcessedBlockKey(key []byte, shardNum byte) {
	_ = key[lastBlockKeyLen-1] //bounds check
	key[0] = prefixLastBlock
	key[1] = shardNum
}

func encodeAccountName(account string) uint64 {
	return eos.MustStringToName(account)
}
