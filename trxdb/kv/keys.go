package kv

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dfuse-io/kvdb"
	"github.com/eoscanada/eos-go"
)

const (
	TblPrefixTrxs      = 0x00
	TblPrefixBlocks    = 0x01
	TblPrefixIrrBlks   = 0x02
	TblPrefixImplTrxs  = 0x03
	TblPrefixDtrxs     = 0x04
	TblPrefixTrxTraces = 0x05
	TblPrefixAccts     = 0x06
	TblTTL             = 0x10

	idxPrefixTimelineFwd = 0x80
	idxPrefixTimelineBck = 0x81

	dtrxSuffixCreated   = 0x90
	dtrxSuffixCancelled = 0x91
	dtrxSuffixFailed    = 0x92
)

var Keys Keyer

type Keyer struct{}

// Blocks virtual table

func (Keyer) PackBlocksKey(blockID string) []byte {
	// OPTIM: don't do moare hex serialization in `ReversedBlockID`,
	// deal directly with bytes.
	id, err := hex.DecodeString(kvdb.ReversedBlockID(blockID))
	if err != nil {
		panic(fmt.Sprintf("invalid block ID %q: %s", blockID, err))
	}
	return append([]byte{TblPrefixBlocks}, id...)
}

func (Keyer) UnpackBlocksKey(key []byte) (blockID string) {
	return kvdb.ReversedBlockID(hex.EncodeToString(key[1:]))
}

func (Keyer) PackBlockNumPrefix(blockNum uint32) []byte {
	hexBlockNum, err := hex.DecodeString(kvdb.HexRevBlockNum(blockNum))
	if err != nil {
		panic(fmt.Sprintf("invalid block num %d: %s", blockNum, err))
	}
	return append([]byte{TblPrefixBlocks}, hexBlockNum...)
}

func (Keyer) StartOfBlocksTable() []byte { return []byte{TblPrefixBlocks} }
func (Keyer) EndOfBlocksTable() []byte   { return []byte{TblPrefixBlocks + 1} }

// Irr Blocks virt table

func (Keyer) PackIrrBlocksKey(blockID string) []byte {
	id, err := hex.DecodeString(kvdb.ReversedBlockID(blockID))
	if err != nil {
		panic(fmt.Sprintf("invalid irr block ID %q: %s", blockID, err))
	}
	return append([]byte{TblPrefixIrrBlks}, id...)
}

func (Keyer) UnpackIrrBlocksKey(key []byte) (blockID string) {
	return kvdb.ReversedBlockID(hex.EncodeToString(key[1:]))
}

func (Keyer) PackIrrBlockNumPrefix(blockNum uint32) []byte {
	hexBlockNum, err := hex.DecodeString(kvdb.HexRevBlockNum(blockNum))
	if err != nil {
		panic(fmt.Sprintf("invalid block num %d: %s", blockNum, err))
	}
	return append([]byte{TblPrefixIrrBlks}, hexBlockNum...)
}

func (Keyer) StartOfIrrBlockTable() []byte { return []byte{TblPrefixIrrBlks} }
func (Keyer) EndOfIrrBlockTable() []byte   { return []byte{TblPrefixIrrBlks + 1} }

// Trx virt table

func (k Keyer) PackTrxsKey(trxID string, blockID string) []byte {
	return k.packTrxBlockIDKey(TblPrefixTrxs, trxID, blockID)
}

func (k Keyer) UnpackTrxsKey(key []byte) (trxID, blockID string) {
	return k.unpackTrxBlockIDKey(key)
}

func (k Keyer) PackTrxsPrefix(trxID string) []byte {
	return k.packTrxPrefix(TblPrefixTrxs, trxID)
}

func (Keyer) StartOfTrxsTable() []byte { return []byte{TblPrefixTrxs} }
func (Keyer) EndOfTrxsTable() []byte   { return []byte{TblPrefixTrxs + 1} }

// TrxTrace virt table
func (k Keyer) PackTrxTracesKey(trxID, blockID string) []byte {
	return k.packTrxBlockIDKey(TblPrefixTrxTraces, trxID, blockID)
}

func (k Keyer) UnpackTrxTracesKey(key []byte) (trxID, blockID string) {
	return k.unpackTrxBlockIDKey(key)
}

func (k Keyer) PackTrxTracesPrefix(trxID string) []byte {
	return k.packTrxPrefix(TblPrefixTrxTraces, trxID)
}

func (Keyer) StartOfTrxTracesTable() []byte { return []byte{TblPrefixTrxTraces} }
func (Keyer) EndOfTrxTracesTable() []byte   { return []byte{TblPrefixTrxTraces + 1} }

// Implicit trx virt table

func (k Keyer) PackImplicitTrxsKey(trxID, blockID string) []byte {
	return k.packTrxBlockIDKey(TblPrefixImplTrxs, trxID, blockID)
}

func (k Keyer) UnpackImplicitTrxsKey(key []byte) (trxID, blockID string) {
	return k.unpackTrxBlockIDKey(key)
}

func (k Keyer) PackImplicitTrxsPrefix(trxID string) []byte {
	return k.packTrxPrefix(TblPrefixImplTrxs, trxID)
}

func (Keyer) StartOfImplicitTrxsTable() []byte { return []byte{TblPrefixImplTrxs} }
func (Keyer) EndOfImplicitTrxsTable() []byte   { return []byte{TblPrefixImplTrxs + 1} }

// Dtrx virt table
func (k Keyer) PackDtrxsKeyCreated(trxID, blockID string) []byte {
	return k.packDtrxsKey(trxID, blockID, dtrxSuffixCreated)
}

func (k Keyer) PackDtrxsKeyFailed(trxID, blockID string) []byte {
	return k.packDtrxsKey(trxID, blockID, dtrxSuffixFailed)
}

func (k Keyer) PackDtrxsKeyCancelled(trxID, blockID string) []byte {
	return k.packDtrxsKey(trxID, blockID, dtrxSuffixCancelled)
}

func (k Keyer) packDtrxsKey(trxID, blockID string, dtrxSuffix byte) []byte {
	id, err := hex.DecodeString(trxID + blockID)
	if err != nil {
		panic(fmt.Sprintf("invalid trx ID %q or block ID %q: %s", trxID, blockID, err))
	}
	key := append([]byte{TblPrefixDtrxs}, id...)
	return append(key, []byte{dtrxSuffix}...)
}

func (k Keyer) UnpackDtrxsKey(key []byte) (trxID, blockID string) {
	if len(key) != 66 {
		panic("invalid key length")
	}
	return hex.EncodeToString(key[1:33]), hex.EncodeToString(key[33:65])
}

func (k Keyer) PackDtrxsPrefix(trxID string) []byte {
	return k.packTrxPrefix(TblPrefixDtrxs, trxID)
}

func (Keyer) StartOfDtrxsTable() []byte { return []byte{TblPrefixDtrxs} }
func (Keyer) EndOfDtrxsTable() []byte   { return []byte{TblPrefixDtrxs + 1} }

// Account virt table

func (Keyer) PackAccountKey(accountName string) []byte {
	name, err := eos.StringToName(accountName)
	if err != nil {
		panic(fmt.Sprintf("invalid account name %q: %s", accountName, err))
	}
	b := make([]byte, 9)
	b[0] = TblPrefixAccts
	binary.LittleEndian.PutUint64(b[1:], name)
	return b
}

func (Keyer) UnpackAccountKey(key []byte) string {
	i := binary.LittleEndian.Uint64(key[1:])
	return eos.NameToString(i)
}

func (Keyer) StartOfAccountTable() []byte { return []byte{TblPrefixAccts} }
func (Keyer) EndOfAccountTable() []byte   { return []byte{TblPrefixAccts + 1} }

// Timeline indexes

func (Keyer) PackTimelineKey(fwd bool, blockTime time.Time, blockID string) []byte {
	bKey, err := hex.DecodeString(blockID)
	if err != nil {
		panic(fmt.Sprintf("failed to decode block ID %q: %s", blockID, err))
	}

	tKey := make([]byte, 9)
	if fwd {
		tKey[0] = idxPrefixTimelineFwd
	} else {
		tKey[0] = idxPrefixTimelineBck
	}
	nano := uint64(blockTime.UnixNano() / 100000000)
	if !fwd {
		nano = maxUnixTimestampDeciSeconds - nano
	}
	binary.BigEndian.PutUint64(tKey[1:], nano)
	return append(tKey, bKey...)
}

var maxUnixTimestampDeciSeconds = uint64(99999999999)

func (Keyer) UnpackTimelineKey(fwd bool, key []byte) (blockTime time.Time, blockID string) {
	t := binary.BigEndian.Uint64(key[1:9]) // skip table prefix
	if !fwd {
		t = maxUnixTimestampDeciSeconds - t
	}
	ns := (int64(t) % 10) * 100000000
	blockTime = time.Unix(int64(t)/10, ns).UTC()
	blockID = hex.EncodeToString(key[9:])
	return
}

func (k Keyer) PackTimelinePrefix(fwd bool, blockTime time.Time) []byte {
	return k.PackTimelineKey(fwd, blockTime, "")
}

func (Keyer) StartOfTimelineIndex(fwd bool) []byte {
	if fwd {
		return []byte{idxPrefixTimelineFwd}
	}
	return []byte{idxPrefixTimelineBck}
}

func (Keyer) EndOfTimelineIndex(fwd bool) []byte {
	if fwd {
		return []byte{idxPrefixTimelineFwd + 1}
	}
	return []byte{idxPrefixTimelineBck + 1}
}

func (Keyer) packTrxBlockIDKey(prefix byte, trxID, blockID string) []byte {
	id, err := hex.DecodeString(trxID + blockID)
	if err != nil {
		panic(fmt.Sprintf("invalid trx ID %q or block ID %q: %s", trxID, blockID, err))
	}
	return append([]byte{prefix}, id...)
}

func (Keyer) unpackTrxBlockIDKey(key []byte) (trxID, blockID string) {
	if len(key) != 65 {
		panic("invalid key length")
	}
	return hex.EncodeToString(key[1:33]), hex.EncodeToString(key[33:65])
}

func (Keyer) packTrxPrefix(prefix byte, trxIDPrefix string) []byte {
	id, err := hex.DecodeString(trxIDPrefix) // trxIDPrefix needs to be an even number'd chars, sanitize before calling
	if err != nil {
		panic(fmt.Sprintf("invalid trx ID hex prefix %q: %s", trxIDPrefix, err))
	}
	return append([]byte{prefix}, id...)
}
