package codec

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/golang/protobuf/proto"
)

func BlockDecoder(blk *bstream.Block) (interface{}, error) {
	block := new(pbcodec.Block)
	err := proto.Unmarshal(blk.Payload(), block)
	if err != nil {
		return nil, fmt.Errorf("unable to decode payload: %s", err)
	}

	return block, nil
}
