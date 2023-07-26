// Copyright 2020 dfuse Platform Inc.
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

package rest

import (
	"testing"
	"time"

	"context"

	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/hub"
	"github.com/streamingfast/shutter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type archiveFile struct {
	name    string
	content string
}

func Test_Handoffs(t *testing.T) {
	cases := []struct {
		name                string
		blocks              []*bstream.Block
		awaitHandoffs       int
		expectedLastBlockID string
		expectHandoff       bool
	}{
		{
			name: "fork with recovery-todo FIXME",
			blocks: []*bstream.Block{
				txpushTestBlock(t, "00000001a", "00000000a", "eoscanadacom", "a"),
				txpushTestBlock(t, "00000002a", "00000001a", "eoscanadacom", "expected.tx.id"),
				txpushTestBlock(t, "00000002b", "00000001a", "eosswedenorg", "x"),
				txpushTestBlock(t, "00000003b", "00000002b", "eosswedenorg", "y"),
				txpushTestBlock(t, "00000003a", "00000002a", "eosriobrazil", "z"),
				txpushTestBlock(t, "00000004a", "00000003a", "eosriobrazil", "z2"),
			},
			awaitHandoffs:       1,
			expectedLastBlockID: "00000004a",
			expectHandoff:       true,
		},
		{
			name: "fork without recovery",
			blocks: []*bstream.Block{
				txpushTestBlock(t, "00000001a", "00000000a", "eoscanadacom", "a"),
				txpushTestBlock(t, "00000002a", "00000001a", "eoscanadacom", "expected.tx.id"),
				txpushTestBlock(t, "00000002b", "00000001a", "eosswedenorg", "x"),
				txpushTestBlock(t, "00000003b", "00000002b", "eosswedenorg", "y"),
				txpushTestBlock(t, "00000004b", "00000003b", "eosriobrazil", "z"),
				txpushTestBlock(t, "00000005b", "00000004b", "eosriobrazil", "z2"),
			},
			awaitHandoffs:       1,
			expectedLastBlockID: "00000004b",
			expectHandoff:       false,
		},
		{
			name: "2 handoffs",
			blocks: []*bstream.Block{
				txpushTestBlock(t, "00000001a", "00000000a", "eoscanadacom", "a"),
				txpushTestBlock(t, "00000002a", "00000001a", "eoscanadacom", "expected.tx.id"),
				txpushTestBlock(t, "00000003a", "00000002a", "eosriobrazil", "z"),
				txpushTestBlock(t, "00000004a", "00000003a", "secondone", "z2"),
				txpushTestBlock(t, "00000005a", "00000004a", "secondone", "z2"),
			},
			awaitHandoffs:       2,
			expectedLastBlockID: "00000004a",
			expectHandoff:       true,
		},
		{
			name: "3 handoffs",
			blocks: []*bstream.Block{
				txpushTestBlock(t, "00000001a", "00000000a", "eoscanadacom", "a"),
				txpushTestBlock(t, "00000002a", "00000001a", "eoscanadacom", "expected.tx.id"),
				txpushTestBlock(t, "00000003a", "00000002a", "eosriobrazil", "z"),
				txpushTestBlock(t, "00000004a", "00000003a", "secondone", "z2"),
				txpushTestBlock(t, "00000005a", "00000004a", "thirddone", "z2"),
				txpushTestBlock(t, "00000006a", "00000005a", "thirddone", "z2"),
			},
			awaitHandoffs:       3,
			expectedLastBlockID: "00000005a",
			expectHandoff:       true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tooLate := make(chan interface{})

			blockSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
				tooLateHandler := NewTooLateHandler(h, c.expectedLastBlockID, tooLate)

				return bstream.NewMockSource(c.blocks, tooLateHandler)
			})

			liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
				return newDummySource()
			})

			buf := bstream.NewBuffer("mama", zlog)
			tailManager := bstream.NewSimpleTailManager(buf, 10)
			subhub, err := hub.NewSubscriptionHub(0, buf, tailManager.TailLock, blockSourceFactory, liveSourceFactory)
			require.NoError(t, err)

			trxFound, _ := awaitTransactionPassedHandoffs(context.Background(), "00000001a", "expected.tx.id", c.awaitHandoffs, subhub)

			handoffPassed := false
			select {
			case <-tooLate:
			case <-trxFound:
				handoffPassed = true
			case <-time.After(1 * time.Second):
				require.Failf(t, "Expected to got some kind of feedback after 1s, but received nothing", "failure")
			}

			require.Equal(t, c.expectHandoff, handoffPassed)
		})
	}
}

type TooLateHandler struct {
	tooLate             chan interface{}
	expectedLastBlockID string
	passed              bool
	next                bstream.Handler
}

func NewTooLateHandler(next bstream.Handler, expectedLastBlockID string, tooLate chan interface{}) *TooLateHandler {
	return &TooLateHandler{
		tooLate:             tooLate,
		next:                next,
		expectedLastBlockID: expectedLastBlockID,
	}
}

func (p *TooLateHandler) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	if p.passed {
		close(p.tooLate)
	}

	if blk.ID() == p.expectedLastBlockID {
		p.passed = true
	}

	return p.next.ProcessBlock(blk, obj)

}

func newDummySource() bstream.Source {
	s := &dummySource{}
	s.Shutter = shutter.New()
	return s
}

type dummySource struct {
	*shutter.Shutter
}

func (s *dummySource) Run() {}

func (s *dummySource) SetLogger(logger *zap.Logger) {}

func txpushTestBlock(t *testing.T, id, previousID, producer, trxID string) *bstream.Block {
	pbblock := &pbcodec.Block{
		Id:     id,
		Number: eos.BlockNum(id),
		Header: &pbcodec.BlockHeader{
			Previous:  previousID,
			Producer:  producer,
			Timestamp: &timestamp.Timestamp{},
		},
		UnfilteredTransactionTraces: []*pbcodec.TransactionTrace{
			{
				Id: trxID,
			},
		},
	}

	block, err := codec.BlockFromProto(pbblock)
	require.NoError(t, err)

	return block
}
