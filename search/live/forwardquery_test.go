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
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	_ "github.com/dfuse-io/dfuse-eosio/codec"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	eosSearch "github.com/dfuse-io/dfuse-eosio/search"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	pb "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/search"
	searchLive "github.com/dfuse-io/search/live"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_forwardProcessBlock(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	mapper, _ := eosSearch.NewEOSBlockMapper("dfuseiohooks:event", "")
	preIndexer := search.NewPreIndexer(mapper, tmpDir)

	cases := []struct {
		name                  string
		block                 *pbeos.Block
		expectedMatchCount    int
		expectedLastBlockRead uint64
		cancelContext         bool
		expectedError         string
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
			expectedError:      "rpc error: code = Canceled desc = context canceled",
		},
		{
			name:               "block to young context",
			block:              newBlock("00000009a", "00000001a", trxID(2), "eosio.token"),
			expectedMatchCount: 0,
			expectedError:      "end of block range",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			block, err := ToBStreamBlock(c.block)
			require.NoError(t, err)
			preprocessObj, err := preIndexer.Preprocess(block)

			fObj := &forkable.ForkableObject{
				Obj: preprocessObj.(*search.SingleIndex),
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
					LowBlockNum:  0,
					HighBlockNum: uint64(8),
				},
			}

			matchesReceived := make(chan bool)
			var matches []*pb.SearchMatch
			if c.expectedMatchCount > 0 {
				go func() {
					select {
					case m := <-incomingMatches:
						matches = append(matches, m)
						if len(matches) == c.expectedMatchCount {
							close(matchesReceived)
						}
					case <-time.After(100 * time.Millisecond):
						close(matchesReceived)
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
			q.Ctx = ctx
			q.MatchCollector = matchCollector
			q.IncomingMatches = incomingMatches
			err = q.ForwardProcessBlock(block, fObj)
			if c.expectedError != "" {
				require.Error(t, err)
				require.Equal(t, c.expectedError, err.Error())
				return
			}

			require.NoError(t, err)
			<-matchesReceived
			assert.Equal(t, c.expectedLastBlockRead, q.LastBlockRead)
			assert.Len(t, matches, c.expectedMatchCount)
		})
	}

}

func Test_processMatches(t *testing.T) {
	cases := []struct {
		name               string
		block              *pbeos.Block
		liveQuery          *searchLive.LiveQuery
		matches            []search.SearchMatch
		expectedMatchCount int
	}{
		{
			name:               "With Match no marker",
			liveQuery:          &searchLive.LiveQuery{},
			block:              newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
			expectedMatchCount: 1,
			matches: []search.SearchMatch{
				&eosSearch.EOSSearchMatch{},
			},
		},
		{
			name: "With Match and marker",
			liveQuery: &searchLive.LiveQuery{
				LiveMarkerReached:          true,
				LiveMarkerLastSentBlockNum: 1,
				Request: &pb.BackendRequest{
					LiveMarkerInterval: 2,
				},
			},
			matches: []search.SearchMatch{
				&eosSearch.EOSSearchMatch{},
			},
			block:              newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
			expectedMatchCount: 2,
		},
		{
			name: "No match and marker",
			liveQuery: &searchLive.LiveQuery{
				LiveMarkerReached:          true,
				LiveMarkerLastSentBlockNum: 1,
				Request: &pb.BackendRequest{
					LiveMarkerInterval: 2,
				},
			},
			block:              newBlock("00000006a", "00000005a", trxID(2), "eosio.token"),
			expectedMatchCount: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			block, err := ToBStreamBlock(c.block)
			require.NoError(t, err)

			ctx := context.Background()

			c.liveQuery.Ctx = ctx

			incomingMatches := make(chan *pb.SearchMatch)
			doneReceiving := make(chan bool)

			var matchesReceived []*pb.SearchMatch
			if c.expectedMatchCount > 0 {
				go func() {
					for {
						select {
						case m := <-incomingMatches:
							matchesReceived = append(matchesReceived, m)
							if len(matchesReceived) == c.expectedMatchCount {
								close(doneReceiving)
							}
						}
					}
				}()
			} else {
				close(doneReceiving)
			}

			c.liveQuery.IncomingMatches = incomingMatches

			err = c.liveQuery.ProcessMatches(c.matches, block, uint64(5), forkable.StepNew)
			require.NoError(t, err)
			<-doneReceiving

			assert.Len(t, matchesReceived, c.expectedMatchCount)

		})
	}

}

func newBlock(id, previous, trxID string, account string) *pbeos.Block {

	return &pbeos.Block{
		Id:     id,
		Number: eos.BlockNum(id),
		Header: &pbeos.BlockHeader{
			Previous:  previous,
			Timestamp: &timestamp.Timestamp{Nanos: 0, Seconds: 0},
		},
		TransactionTraces: []*pbeos.TransactionTrace{
			{
				Id: trxID,
				Receipt: &pbeos.TransactionReceiptHeader{
					Status: pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
				},
				ActionTraces: []*pbeos.ActionTrace{
					{
						Receipt: &pbeos.ActionReceipt{
							Receiver: "receiver.1",
						},
						Action: &pbeos.Action{
							Account: account,
							Name:    "transfer",
						},
					},
				},
			},
		},
	}
}

func ToBStreamBlock(block *pbeos.Block) (*bstream.Block, error) {
	time, _ := ptypes.Timestamp(block.Header.Timestamp)
	payload, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	return &bstream.Block{
		Id:             block.Id,
		Number:         uint64(block.Number),
		PreviousId:     block.PreviousID(),
		Timestamp:      time,
		LibNum:         block.LIBNum(),
		PayloadKind:    pbbstream.Protocol_EOS,
		PayloadVersion: 1,
		PayloadBuffer:  payload,
	}, nil
}

func trxID(num int) string {
	out := fmt.Sprintf("%d", num)
	for {
		out = fmt.Sprintf("%s.%d", out, num)
		if len(out) >= 32 {
			return out[:32]
		}
	}
}
