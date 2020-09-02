package accounthist

import pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"

const CursorMagicValue = 4374

func actionKeyToCursor(account uint64, key []byte) *pbaccounthist.Cursor {
	shardNo, seqNum := decodeActionKeySeqNum(key)
	return &pbaccounthist.Cursor{
		Version:        0,
		Magic:          CursorMagicValue,
		Account:        account,
		ShardNum:       uint32(shardNo),
		SequenceNumber: seqNum,
	}
}

func cursorToActionKey(cursor *pbaccounthist.Cursor) []byte {
	return encodeActionKey(cursor.Account, byte(cursor.ShardNum), cursor.SequenceNumber)
}
