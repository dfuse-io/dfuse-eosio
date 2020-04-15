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

package search

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/dfuse-io/bstream"
	_ "github.com/dfuse-io/dfuse-eosio/codecs/deos"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/search"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	search.GetMatchCollector = Collect
}

func TestPreIndexerRunSingleIndexQuery(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	preIndexer := search.NewPreIndexer(NewEOSBlockMapper("dfuseiohooks:event", nil), tmpDir)

	block, err := ToBStreamBlock(newBlock("00000001a", "00000000a", trxID(1), "eosio.token"))
	require.NoError(t, err)
	matchCollector := Collect

	preprocessObj, err := preIndexer.Preprocess(block)
	index := preprocessObj.(*search.SingleIndex)
	ctx := context.Background()
	sortDesc := false
	lowBlockNum := uint64(0)
	highBlockNum := uint64(1)
	releaseFunc := func() {}
	metrics := search.NewQueryMetrics(zap.NewNop(), sortDesc, "", 1, 0, 0)
	bleveQuery, err := search.NewParsedQuery("account:eosio.token")

	matches, err := search.RunSingleIndexQuery(ctx, sortDesc, lowBlockNum, highBlockNum, matchCollector, bleveQuery, index.Index, releaseFunc, metrics)
	require.NoError(t, err)
	require.Len(t, matches, 1)
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
