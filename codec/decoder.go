// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbbstream "github.com/streamingfast/pbgo/dfuse/bstream/v1"
	"github.com/golang/protobuf/proto"
)

// BlockDecoder transforms a `bstream.Block` payload into a proper `deth.Block` value
func BlockDecoder(blk *bstream.Block) (interface{}, error) {
	if blk.Kind() != pbbstream.Protocol_EOS {
		return nil, fmt.Errorf("expected kind %s, got %s", pbbstream.Protocol_EOS, blk.Kind())
	}

	if blk.Version() != 1 {
		return nil, fmt.Errorf("this decoder only knows about bstream.Block version 1, got %d", blk.Version())
	}

	block := new(pbcodec.Block)
	err := proto.Unmarshal(blk.Payload(), block)
	if err != nil {
		return nil, fmt.Errorf("unable to decode payload: %s", err)
	}

	// This whole BlockDecoder method is being called through the `bstream.Block.ToNative()`
	// method. Hence, it's a great place to add temporary data normalization calls to backport
	// some features that were not in all blocks yet (because we did not re-process all blocks
	// yet).
	//
	// Thoughts for the future: Ideally, we would leverage the version information here to take
	// a decision, like `do X if version <= 2.1` so we would not pay the performance hit
	// automatically instead of having to re-deploy a new version of bstream (which means
	// rebuild everything mostly)
	//
	// We reconstruct the transaction & action count values

	const MAX_SUPPORTED_PBCODEC_VERSION = 1
	if block.Version > MAX_SUPPORTED_PBCODEC_VERSION {
		return nil, fmt.Errorf("future block formats not supported, this code supports dfuse.eosio.codec.v1.Block version %d, received version %d", MAX_SUPPORTED_PBCODEC_VERSION, block.Version)
	}

	block.MigrateV0ToV1()
	//block.MigrateV1ToV2()

	return block, nil
}
