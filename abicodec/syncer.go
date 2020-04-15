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

package abicodec

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/abicodec/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	searchclient "github.com/dfuse-io/dfuse-eosio/search-client"
	"github.com/dfuse-io/dgrpc"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type IsLive chan interface{}

type ABISyncer struct {
	*shutter.Shutter

	cache        Cache
	client       *searchclient.EOSClient
	isLive       bool
	onLive       func()
	syncCtx      context.Context
	cancelSyncer func()
}

func NewSyncer(cache Cache, dbReader eosdb.DBReader, searchAddr string, onLive func()) (*ABISyncer, error) {
	zlog.Info("initializing syncer", zap.String("search_addr", searchAddr))
	searchConn, err := dgrpc.NewInternalClient(searchAddr)
	if err != nil {
		return nil, fmt.Errorf("unable to init gRPC search connection: %w", err)
	}

	syncCtx, cancelSyncer := context.WithCancel(context.Background())

	syncer := &ABISyncer{
		Shutter:      shutter.New(),
		cache:        cache,
		client:       searchclient.NewEOSClient(searchConn, dbReader),
		onLive:       onLive,
		syncCtx:      syncCtx,
		cancelSyncer: cancelSyncer,
	}
	syncer.OnTerminating(syncer.cleanup)

	return syncer, nil
}

func (s *ABISyncer) cleanup(error) {
	zlog.Info("terminating syncer via shutter")
	s.cancelSyncer()
}

func (s *ABISyncer) Sync() {
	for {
		zlog.Info("starting ABI syncer")
		err := s.streamABIChanges()
		if err != nil && !errors.Is(err, context.Canceled) {
			zlog.Info("the search stream ended with error", zap.Error(err))
		}

		zlog.Info("waiting before startng ABI syncer")
		select {
		case <-s.syncCtx.Done():
			return
			// FIXME: Exponential backoff!
		case <-time.After(1 * time.Second):
		}
	}
}

func (s *ABISyncer) streamABIChanges() error {
	zlog.Debug("streaming abi changes", zap.String("cursor", s.cache.GetCursor()))

	ctx, cancelSearch := context.WithCancel(s.syncCtx)
	defer cancelSearch()

	stream, err := s.client.StreamMatches(ctx, &pbsearch.RouterRequest{
		Query:              "receiver:eosio action:setabi notif:false",
		LowBlockNum:        1,
		HighBlockUnbounded: true,
		LiveMarkerInterval: 1,
		WithReversible:     true,
		Cursor:             s.cache.GetCursor(),
		Mode:               pbsearch.RouterRequest_STREAMING,
	})
	if err != nil {
		return fmt.Errorf("unble to init search query for all ABIs: %w", err)
	}

	for {
		match, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				zlog.Error("received end of stream marker, but this should never happen")
				return nil
			}

			return fmt.Errorf("search stream terminated with error: %w", err)
		}

		if traceEnabled {
			zlog.Debug("received search ABI match from client")
		}

		blockRef := bstream.BlockRefFromID(match.BlockID)
		if match.TransactionTrace == nil {
			zlog.Debug("found a live marker")
			s.handleLiveMaker(blockRef, match.Cursor)
			continue
		}

		transactionID := match.TransactionTrace.Id
		for _, action := range match.MatchingActions {
			s.handleABIAction(blockRef, transactionID, action, match.Undo)
		}
	}
}

func (s *ABISyncer) handleABIAction(blockRef bstream.BlockRef, trxID string, actionTrace *pbeos.ActionTrace, undo bool) error {
	account := actionTrace.GetData("account").String()
	hexABI := actionTrace.GetData("abi")

	if !hexABI.Exists() {
		zlog.Warn("'setabi' action data payload not present", zap.String("account", account), zap.String("transaction_id", trxID))
		return nil
	}

	if undo {
		s.cache.RemoveABIAtBlockNum(account, uint32(blockRef.Num()))
		return nil
	}

	hexData := hexABI.String()
	if hexData == "" {
		zlog.Info("empty ABI in 'setabi' action", zap.String("account", account), zap.String("transaction_id", trxID))
		return nil
	}

	abiData, err := hex.DecodeString(hexData)
	if err != nil {
		zlog.Info("failed to hex decode abi string", zap.String("account", account), zap.String("transaction_id", trxID), zap.Error(err))
		return nil // do not return the error. Worker will retry otherwise
	}

	var abi *eos.ABI
	err = eos.UnmarshalBinary(abiData, &abi)
	if err != nil {
		abiHexCutAt := math.Min(50, float64(len(hexData)))

		zlog.Info("failed to unmarshal abi from binary",
			zap.String("account", account),
			zap.String("transaction_id", trxID),
			zap.String("abi_hex_prefix", hexData[0:int(abiHexCutAt)]),
			zap.Error(err),
		)

		return nil
	}

	zlog.Debug("setting new abi", zap.String("account", account), zap.Stringer("transaction_id", blockRef), zap.Stringer("block", blockRef))
	s.cache.SetABIAtBlockNum(account, uint32(blockRef.Num()), abi)

	return nil
}

func (s *ABISyncer) handleLiveMaker(blockRef bstream.BlockRef, cursor string) {
	s.cache.SetCursor(cursor)
	if !s.isLive {
		zlog.Info("received the first live maker, we are now receiving data from live block")
		s.isLive = true

		if s.onLive != nil {
			zlog.Info("notifying on live callback")
			s.onLive()
		}
	}

	metrics.HeadBlockNumer.SetUint64(blockRef.Num())
}
