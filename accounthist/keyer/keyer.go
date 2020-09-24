package keyer

import "encoding/binary"

const (
	//prefixSequenceNumber = byte(0x01) // unused now
	PrefixAccount           = byte(0x02)
	PrefixAccountCheckpoint = byte(0x03)

	PrefixAccountContract           = byte(0x04)
	PrefixAccountContractCheckpoint = byte(0x05)

	TokenPrefixLen      = 17
	AccountPrefixKeyLen = 9
	AccountKeyLen       = 18
	TokenKeyLen         = 26
	CheckpointLen       = 2
)

func EncodeAccountWithPrefixKey(prefix byte, account uint64) []byte {
	key := make([]byte, AccountPrefixKeyLen)

	key[0] = prefix
	binary.BigEndian.PutUint64(key[1:], account)
	return key
}

func EncodeAccountContractPrefixKey(account uint64, contract uint64) []byte {
	key := make([]byte, TokenPrefixLen)

	key[0] = PrefixAccountContract
	binary.BigEndian.PutUint64(key[1:], account)
	binary.BigEndian.PutUint64(key[9:], contract)
	return key
}

func EncodeAccountContractKey(account uint64, contract uint64, shardNum byte, ordinalNumber uint64) []byte {
	key := make([]byte, TokenKeyLen)

	key[0] = PrefixAccountContract
	binary.BigEndian.PutUint64(key[1:], account)
	binary.BigEndian.PutUint64(key[9:], contract)

	// We want the rows to be sorted by shard ascending 0 -> n
	key[17] = shardNum
	binary.BigEndian.PutUint64(key[18:], ^ordinalNumber)

	return key
}

func DecodeAccountContractKeySeqNum(key []byte) (uint64, uint64, byte, uint64) {
	_ = key[TokenKeyLen-1] //bounds check
	account := binary.BigEndian.Uint64(key[1:])
	contract := binary.BigEndian.Uint64(key[9:])
	shardNum := key[17]
	ordinalNumber := binary.BigEndian.Uint64(key[18:])
	return account, contract, shardNum, ^ordinalNumber
}

func EncodeAccountPrefixKey(account uint64) []byte {
	key := make([]byte, AccountPrefixKeyLen)

	key[0] = PrefixAccount
	binary.BigEndian.PutUint64(key[1:], account)
	return key
}

func EncodeAccountKey(account uint64, shardNum byte, ordinalNumber uint64) []byte {
	key := make([]byte, AccountKeyLen)

	key[0] = PrefixAccount

	binary.BigEndian.PutUint64(key[1:], account)

	// We want the rows to be sorted by shard ascending 0 -> n
	key[9] = shardNum
	binary.BigEndian.PutUint64(key[10:], ^ordinalNumber)

	return key
}

func DecodeAccountKeySeqNum(key []byte) (uint64, byte, uint64) {
	_ = key[AccountKeyLen-1] //bounds check
	account := binary.BigEndian.Uint64(key[1:])
	shardNum := key[9]
	ordinalNumber := binary.BigEndian.Uint64(key[10:])
	return account, shardNum, ^ordinalNumber
}

func EncodeAccountCheckpointKey(shardNum byte) []byte {
	key := make([]byte, CheckpointLen)
	key[0] = PrefixAccountCheckpoint
	key[1] = shardNum
	return key
}

func EncodeAccountContractCheckpointKey(shardNum byte) []byte {
	key := make([]byte, CheckpointLen)
	key[0] = PrefixAccountContractCheckpoint
	key[1] = shardNum
	return key
}

func DecodeCheckpointKey(key []byte) byte {
	_ = key[CheckpointLen-1] //bounds check
	return key[1]
}
