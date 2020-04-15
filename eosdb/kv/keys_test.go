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
