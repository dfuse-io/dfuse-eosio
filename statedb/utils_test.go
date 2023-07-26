package statedb

import (
	"encoding/hex"
	"testing"

	"github.com/golang/protobuf/proto"
	pbfluxdb "github.com/streamingfast/pbgo/dfuse/fluxdb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoint_Unmarshal(t *testing.T) {
	dataHex := "088c011245088c01124030303030303038636665353638633661383835393366613864313732623533346236626162653866393435663161623763636566356662323535336236306261"
	data, err := hex.DecodeString(dataHex)
	require.NoError(t, err)

	checkpoint := pbfluxdb.Checkpoint{}
	err = proto.Unmarshal(data, &checkpoint)
	require.NoError(t, err)

	assert.Equal(t, "0000008cfe568c6a88593fa8d172b534b6babe8f945f1ab7ccef5fb2553b60ba", checkpoint.Block.Id)
	assert.Equal(t, uint64(140), checkpoint.Block.Num)
	assert.Equal(t, uint64(140), checkpoint.Height)

}
