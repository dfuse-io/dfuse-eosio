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

package eosdbtest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/jsonpb"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type DriverCleanupFunc func()
type DriverFactory func() (eosdb.Driver, DriverCleanupFunc)

func TestAll(t *testing.T, driverName string, driverFactory DriverFactory) {
	TestAllAccountsReader(t, driverName, driverFactory)
	TestAllDbWriter(t, driverName, driverFactory)
	TestAllDbReader(t, driverName, driverFactory)
	TestAllTimelineExplorer(t, driverName, driverFactory)
	TestAllTransactionsReader(t, driverName, driverFactory)
}

func TestBlock(t *testing.T, blkId string, previousBlkId string, trxTraceJSONs ...string) *pbeos.Block {
	trxTraces := make([]*pbeos.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbeos.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbeos.Block{
		Id:                blkId,
		Number:            eos.BlockNum(blkId),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbeos.BlockHeader{
		Previous:  fmt.Sprintf("%08d%s", pbblock.Number-1, blkId[8:]),
		Producer:  "tester",
		Timestamp: blockTimestamp,
	}

	if os.Getenv("DEBUG") != "" {
		marshaler := &jsonpb.Marshaler{}
		out, err := marshaler.MarshalToString(pbblock)
		require.NoError(t, err)

		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
		normalizedOut := map[string]interface{}{}
		require.NoError(t, json.Unmarshal([]byte(out), &normalizedOut))

		zlog.Debug("created test block", zap.Any("block", normalizedOut))
	}

	return pbblock
}

func toTimestamp(t time.Time) *tspb.Timestamp {
	el, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return el
}
