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
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/golang/protobuf/proto"
)

func BlockFromProto(b *pbcodec.Block) (*bstream.Block, error) {
	content, err := proto.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	blockTime, err := b.Time()
	if err != nil {
		return nil, err
	}

	return &bstream.Block{
		Id:             b.ID(),
		Number:         b.Num(),
		PreviousId:     b.PreviousID(),
		Timestamp:      blockTime,
		LibNum:         b.LIBNum(),
		PayloadKind:    pbbstream.Protocol_EOS,
		PayloadVersion: 1,
		PayloadBuffer:  content,
	}, nil
}
