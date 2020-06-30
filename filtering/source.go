package filtering

import (
	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

type FilteringPreprocessor struct {
	Filter *BlockFilter
}

func (f *FilteringPreprocessor) PreprocessBlock(blk *bstream.Block) (interface{}, error) {
	block := blk.ToNative().(*pbcodec.Block)

	// We modify the block in place, which means future `ToNative` will correctly return the
	// filtered block.
	f.Filter.TransformInPlace(block)

	return nil, nil
}
