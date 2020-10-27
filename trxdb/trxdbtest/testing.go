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

package trxdbtest

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
)

type DriverCleanupFunc func()
type DriverFactory func() (trxdb.DB, DriverCleanupFunc)
type DriverTestFunc func(t *testing.T, driverFactory DriverFactory)

func TestAll(t *testing.T, driverName string, driverFactory DriverFactory) {
	all := map[string][]DriverTestFunc{
		"accounts_reader":    accountsReaderTest,
		"db_reader":          dbReaderTests,
		"db_writer":          dbWritterTests,
		"timeline_exporter":  timelineExplorerTests,
		"transaction_reader": transactionReaderTests,
	}

	for driverName, testFuncs := range all {
		for _, testFunc := range testFuncs {
			t.Run(driverName+"/"+getFunctionName(testFunc), func(t *testing.T) {
				testFunc(t, driverFactory)
			})
		}
	}
}

// func TestBlock(t *testing.T, blkId string, previousBlkId string, trxTraceJSONs ...string) *pbcodec.Block {
// 	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
// 	for i, trxTraceJSON := range trxTraceJSONs {
// 		trxTrace := new(pbcodec.TransactionTrace)
// 		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

// 		trxTraces[i] = trxTrace
// 	}

// 	pbblock := &pbcodec.Block{
// 		Id:                          blkId,
// 		Number:                      eos.BlockNum(blkId),
// 		UnfilteredTransactionTraces: trxTraces,
// 	}

// 	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
// 	require.NoError(t, err)

// 	blockTimestamp, err := ptypes.TimestampProto(blockTime)
// 	require.NoError(t, err)

// 	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
// 	pbblock.Header = &pbcodec.BlockHeader{
// 		Previous:  fmt.Sprintf("%08d%s", pbblock.Number-1, blkId[8:]),
// 		Producer:  "tester",
// 		Timestamp: blockTimestamp,
// 	}

// 	if os.Getenv("DEBUG") != "" {
// 		marshaler := &jsonpb.Marshaler{}
// 		out, err := marshaler.MarshalToString(pbblock)
// 		require.NoError(t, err)

// 		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
// 		normalizedOut := map[string]interface{}{}
// 		require.NoError(t, json.Unmarshal([]byte(out), &normalizedOut))

// 		zlog.Debug("created test block", zap.Any("block", normalizedOut))
// 	}

// 	return pbblock
// }

// getFunctionName reads the program counter adddress and return the function
// name representing this address.
//
// The `FuncForPC` format is in the form of `github.com/.../.../package.func`.
// As such, we use `filepath.Base` to obtain the `package.func` part and then
// split it at the `.` to extract the function name.
func getFunctionName(i interface{}) string {
	pcIdentifier := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	baseName := filepath.Base(pcIdentifier)
	parts := strings.SplitN(baseName, ".", 2)
	if len(parts) <= 1 {
		return parts[0]
	}

	return parts[1]
}

func toTimestamp(t time.Time) *tspb.Timestamp {
	el, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return el
}
