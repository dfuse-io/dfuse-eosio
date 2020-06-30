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

package live

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/dfuse-io/derr"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pb "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/search"
	searchLive "github.com/dfuse-io/search/live"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func Test_processSingleBlocks(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	mapper, _ := filtering.NewBlockMapper("dfuseiohooks:event", false, "", "", "")
	preIndexer := search.NewPreIndexer(mapper, tmpDir)

	cases := []struct {
		name                  string
		block                 *pbcodec.Block
		expectedMatchCount    int
		expectedLastBlockRead uint64
		cancelContext         bool
		expectedError         error
	}{
		{
			name:                  "sunny path",
			block:                 newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
			expectedLastBlockRead: uint64(6),
			expectedMatchCount:    1,
		},
		{
			name:               "canceled context",
			block:              newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
			cancelContext:      true,
			expectedMatchCount: 0,
			expectedError:      derr.Status(codes.Canceled, "context canceled"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			block, err := ToBStreamBlock(c.block)
			require.NoError(t, err)
			preprocessObj, err := preIndexer.Preprocess(block)

			idxBlk := &searchLive.IndexedBlock{
				Idx: preprocessObj.(*search.SingleIndex),
				Blk: block,
			}

			bleveQuery, err := search.NewParsedQuery("account:eosio.token")
			matchCollector := search.GetMatchCollector
			if matchCollector == nil {
				panic(fmt.Errorf("no match collector set, should not happen, you should define a collector"))
			}

			incomingMatches := make(chan *pb.SearchMatch)

			q := searchLive.LiveQuery{
				BleveQuery: bleveQuery,
				Request: &pb.BackendRequest{
					LowBlockNum: 5,
				},
			}

			matchesReceived := make(chan bool)
			var matches []*pb.SearchMatch
			if c.expectedMatchCount > 0 {
				go func() {
					for {
						select {
						case m := <-incomingMatches:
							matches = append(matches, m)
							if len(matches) == c.expectedMatchCount {
								close(matchesReceived)
							}
						}
					}
				}()
			} else {
				close(matchesReceived)
			}

			ctx := context.Background()
			if c.cancelContext {
				canceledContext, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceledContext
			}
			err = q.ProcessSingleBlocks(ctx, idxBlk, matchCollector, incomingMatches)
			if c.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, c.expectedError, err)
				return
			}

			require.NoError(t, err)
			<-matchesReceived
			assert.Equal(t, c.expectedLastBlockRead, q.LastBlockRead)
			assert.Len(t, matches, c.expectedMatchCount)
		})
	}

}
