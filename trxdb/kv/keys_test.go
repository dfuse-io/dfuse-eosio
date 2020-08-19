package kv

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKeyer_PackBlocksKey(t *testing.T) {
	key := "00000002aa"
	packed := Keys.PackBlocksKey(key)
	unpacked := Keys.UnpackBlocksKey(packed)
	require.Equal(t, key, unpacked)
}

func TestKeyer_PackAccountKey(t *testing.T) {
	key := ".eoscanadacp"
	packed := Keys.PackAccountKey(key)
	unpacked := Keys.UnpackAccountKey(packed)
	require.Equal(t, key, unpacked)
}

func TestKeyer_PackTimelineKey(t *testing.T) {
	expectedBlockID := "00000002aa"
	expectedBlockTime := time.Unix(0, 0).UTC()

	packed := Keys.PackTimelineKey(true, expectedBlockTime, expectedBlockID)
	blockTime, blockID := Keys.UnpackTimelineKey(true, packed)
	require.Equal(t, expectedBlockID, blockID)
	require.Equal(t, expectedBlockTime, blockTime)
}

func TestKeyer_PackDtrxsKeyCreated(t *testing.T) {
	expectedBlockID := "0000001aafcedbf5e651b27bee47c8a28de01635b5029ac2ce32896a1bcb1615"
	expectedTrxID := "f2c8602f6d2b8241894383b22614a82740338d3f5c34961c0c82b382ac9e11ae"

	packed := Keys.PackDtrxsKeyCreated(expectedTrxID, expectedBlockID)
	trxID, blockID := Keys.UnpackDtrxsKey(packed)
	require.Equal(t, uint8(dtrxSuffixCreated), packed[65])
	require.Equal(t, expectedBlockID, blockID)
	require.Equal(t, expectedTrxID, trxID)
}

func TestKeyer_PackDtrxsKeyCancelled(t *testing.T) {
	expectedBlockID := "0000001aafcedbf5e651b27bee47c8a28de01635b5029ac2ce32896a1bcb1615"
	expectedTrxID := "f2c8602f6d2b8241894383b22614a82740338d3f5c34961c0c82b382ac9e11ae"

	packed := Keys.PackDtrxsKeyCancelled(expectedTrxID, expectedBlockID)
	trxID, blockID := Keys.UnpackDtrxsKey(packed)
	require.Equal(t, uint8(dtrxSuffixCancelled), packed[65])
	require.Equal(t, expectedBlockID, blockID)
	require.Equal(t, expectedTrxID, trxID)

}

func TestKeyer_PackDtrxsKeyFailed(t *testing.T) {
	expectedBlockID := "0000001aafcedbf5e651b27bee47c8a28de01635b5029ac2ce32896a1bcb1615"
	expectedTrxID := "f2c8602f6d2b8241894383b22614a82740338d3f5c34961c0c82b382ac9e11ae"

	packed := Keys.PackDtrxsKeyFailed(expectedTrxID, expectedBlockID)
	trxID, blockID := Keys.UnpackDtrxsKey(packed)
	require.Equal(t, uint8(dtrxSuffixFailed), packed[65])
	require.Equal(t, expectedBlockID, blockID)
	require.Equal(t, expectedTrxID, trxID)

}
