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

package forkresolver

import (
	"context"
	"testing"

	_ "github.com/dfuse-io/dfuse-eosio/codecs/deos"
	"github.com/dfuse-io/dmesh"
	"github.com/dfuse-io/dstore"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	pb "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FetchMatchingBlocks(t *testing.T) {
	t.Skip() // requires real blockstore connection

	blocksStoreURL := "gs://dfuseio-global-blocks-us/eos-mainnet/v3"

	store, err := dstore.NewDBinStore(blocksStoreURL)
	require.NoError(t, err)
	peer := &dmesh.SearchPeer{}
	protocol := pbbstream.Protocol_EOS

	fr := NewForkResolver(store, nil, peer, protocol, "", ":9000", ":8080", nil, "/tmp")
	blocks, lib, err := fr.getBlocksDescending(context.Background(),
		[]*pb.BlockRef{
			{
				BlockID:  "060dc735c405743846419363aeee78e2d342e9b7e819f6c5d338673bad91c6a0",
				BlockNum: 101566261,
			},
			{
				BlockID:  "060dc734bf7e9db8c97c8e4edebfe98767337c0884a98aeecb9fc51a56714b04",
				BlockNum: 101566260,
			},
			{
				BlockID:  "060dc7363fd78e729b5298af5348bb90c09232309745425f306e0fa20d27f1b8",
				BlockNum: 101566262,
			},
		},
	)

	require.NoError(t, err)

	assert.Len(t, blocks, 3)
	assert.Equal(t, uint64(101566262), blocks[0].Number)
	assert.Equal(t, "060dc7363fd78e729b5298af5348bb90c09232309745425f306e0fa20d27f1b8", blocks[0].ID())
	assert.Equal(t, uint64(101566261), blocks[1].Number)
	assert.Equal(t, "060dc735c405743846419363aeee78e2d342e9b7e819f6c5d338673bad91c6a0", blocks[1].ID())
	assert.Equal(t, uint64(101566260), blocks[2].Number)
	assert.Equal(t, "060dc734bf7e9db8c97c8e4edebfe98767337c0884a98aeecb9fc51a56714b04", blocks[2].ID())
	assert.Equal(t, uint64(101566259), lib)

	//	cases := []struct {
	//		name                  string
	//		block                 *pbdeos.Block
	//		expectedMatchCount    int
	//		expectedLastBlockRead uint64
	//		cancelContext         bool
	//		expectedError         string
	//	}{
	//		{
	//			name:                  "sunny path",
	//			block:                 newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
	//			expectedLastBlockRead: uint64(6),
	//			expectedMatchCount:    1,
	//		},
	//		{
	//			name:               "canceled context",
	//			block:              newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
	//			cancelContext:      true,
	//			expectedMatchCount: 0,
	//			expectedError:      "rpc error: code = Canceled desc = context canceled",
	//		},
	//		{
	//			name:               "block to young context",
	//			block:              newBlock("00000009a", "00000001a", trxID(2), "eosio.token"),
	//			expectedMatchCount: 0,
	//			expectedError:      "end of block range",
	//		},
	//	}
	//
	//	for _, c := range cases {
	//		t.Run(c.name, func(t *testing.T) {

	//})

}
