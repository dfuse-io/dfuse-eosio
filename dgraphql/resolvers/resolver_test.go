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

package resolvers

import (
	"context"
	"fmt"
	"os"
	"testing"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/search/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/streamingfast/dtracing"
	"github.com/streamingfast/logging"
	pbsearch "github.com/streamingfast/pbgo/dfuse/search/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/streamingfast/dgraphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("TEST_LOG") != "" {
		zlog = logging.MustCreateLoggerWithLevel("test", zap.NewAtomicLevelAt(zap.DebugLevel))
		logging.Set(zlog)
	}
}

func newSearchMatchArchive(trxID string) *pbsearch.SearchMatch {
	cs, err := ptypes.MarshalAny(&pbsearcheos.Match{})
	if err != nil {
		panic(err)
	}
	return &pbsearch.SearchMatch{
		TrxIdPrefix:   trxID,
		BlockNum:      0,
		Index:         0,
		Cursor:        "",
		ChainSpecific: cs,
		Undo:          false,
		IrrBlockNum:   0,
	}
}

func newSearchMatchLive(trxID string, idx int) *pbsearch.SearchMatch {
	cs, err := ptypes.MarshalAny(&pbsearcheos.Match{
		Block: &pbsearcheos.BlockTrxPayload{
			Trace: &pbcodec.TransactionTrace{Index: uint64(idx)},
		},
	})
	if err != nil {
		panic(err)
	}

	return &pbsearch.SearchMatch{
		TrxIdPrefix:   trxID,
		ChainSpecific: cs,
	}
}

func newDgraphqlResponse(trxID string, idx int) *SearchTransactionForwardResponse {
	return &SearchTransactionForwardResponse{
		SearchTransactionBackwardResponse: SearchTransactionBackwardResponse{
			trxIDPrefix: trxID,
			trxTrace: &pbcodec.TransactionTrace{
				Index: uint64(idx),
			},
		},
	}
}
func TestSubscriptionSearchForward(t *testing.T) {
	ctx := dtracing.NewFixedTraceIDInContext(context.Background(), "00000000000000000000000000000000")

	tests := []struct {
		name        string
		fromRouter  []interface{}
		fromDB      map[string][]*pbcodec.TransactionEvent
		expect      []*SearchTransactionForwardResponse
		expectError error
	}{
		{
			name: "simple",
			fromRouter: []interface{}{
				newSearchMatchArchive("trx123"),
				fmt.Errorf("failed"),
			},
			fromDB: map[string][]*pbcodec.TransactionEvent{
				"trx123": {
					{Id: "trx12399999999999999999", Event: pbcodec.NewSimpleTestExecEvent(5)},
				},
			},
			expect: []*SearchTransactionForwardResponse{
				newDgraphqlResponse("trx123", 5),
				{
					err: dgraphql.Errorf(ctx, "hammer search result: failed"),
				},
			},

			expectError: nil,
		},
		{
			name: "hammered",
			fromRouter: []interface{}{
				newSearchMatchArchive("trx000"),
				newSearchMatchArchive("trx001"),
				newSearchMatchArchive("trx002"),
				newSearchMatchArchive("trx022"),
				newSearchMatchLive("trx003", 8),
				newSearchMatchLive("trx004", 9),
				newSearchMatchLive("trx005", 10),
			},
			fromDB: map[string][]*pbcodec.TransactionEvent{
				"trx000": {
					{Id: "trx000boo", Event: pbcodec.NewSimpleTestExecEvent(5)},
				},
				"trx001": {
					{Id: "trx001boo", Event: pbcodec.NewSimpleTestExecEvent(6)},
				},
				"trx002": {
					{Id: "trx002boo", Event: pbcodec.NewSimpleTestExecEvent(7)},
				},
				"trx022": {
					{Id: "trx022boo", Event: pbcodec.NewSimpleTestExecEvent(11)},
				},
			},
			expect: []*SearchTransactionForwardResponse{
				newDgraphqlResponse("trx000", 5),
				newDgraphqlResponse("trx001", 6),
				newDgraphqlResponse("trx002", 7),
				newDgraphqlResponse("trx022", 11),
				newDgraphqlResponse("trx003", 8),
				newDgraphqlResponse("trx004", 9),
				newDgraphqlResponse("trx005", 10),
			},

			expectError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			root := &Root{
				searchClient: pbsearch.NewTestRouterClient(test.fromRouter),
				trxsReader:   trxdb.NewTestTransactionsReader(test.fromDB),
			}

			res, err := root.streamSearchTracesBoth(true, ctx, StreamSearchArgs{})
			if test.expectError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				var expect []*SearchTransactionForwardResponse
				for el := range res {
					expect = append(expect, el)
				}

				assert.Equal(t, test.expect, expect)
			}
		})
	}
}
