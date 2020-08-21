package codec

import (
	"github.com/dfuse-io/bstream"
)

func init() {
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(BlockDecoder)
}
