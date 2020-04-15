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

package mdl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/jsonpb"
	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Benchmark_ToV1TransactionTrace(b *testing.B) {
	transactionTrace := &pbeos.TransactionTrace{}
	unmarshalFromFixture(filepath.Join("testdata", "01-trx-block-v2.json"), transactionTrace)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ToV1TransactionTrace(transactionTrace)
	}
}

func TestToV1ActionTraceEmptyNoPanic(t *testing.T) {
	ToV1ActionTrace(&pbeos.ActionTrace{}) // should not panic
}

// func TestConvertEOSToDEOS(t *testing.T) {
// 	transactionTrace := &eos.TransactionTrace{}
// 	unmarshalFromFixture(filepath.Join("testdata", "02-action-wraps-v2.json"), transactionTrace)

// 	pbTrace := deos.TransactionTraceToDEOS(transactionTrace)
// 	out := protoJSONMarshalIndent(t, pbTrace)
// 	ioutil.WriteFile("/tmp/throughpb02.json", []byte(out), 0644)
// }

func TestToTransactionV1Lifecycle(t *testing.T) {
	tests := []struct {
		fixture string
	}{
		{"01-trx-block-v2.pb.json"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			transactionTrace := &pbeos.TransactionTrace{}
			protoJSONUnmarshal(t, fromFixture(filepath.Join("testdata", test.fixture)), transactionTrace)

			convertedTrace, err := ToV1TransactionTrace(transactionTrace)
			require.NoError(t, err)

			actual, err := json.MarshalIndent(convertedTrace, "", "  ")
			require.NoError(t, err)

			golden := filepath.Join("testdata", test.fixture+".golden")
			if os.Getenv("GOLDEN_UPDATE") != "" {
				ioutil.WriteFile(golden, actual, 0644)
			}

			expected, err := ioutil.ReadFile(golden)
			require.NoError(t, err, "unable to read golden file %s, use go test ./mdl --update to update golden files: %s", golden, err)

			assert.JSONEq(t, string(expected), string(actual), "actual:\n%s\nexpected:\n%s", actual, expected)
		})
	}
}

func TestToActionTraceRaw(t *testing.T) {
	tests := []struct {
		fixture     string
		actionIndex int
	}{
		{"01-action-wraps-v2.pb.json", 0},
		{"02-action-wraps-v2.pb.json", 1},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			transactionTrace := &pbeos.TransactionTrace{}
			protoJSONUnmarshal(t, fromFixture(filepath.Join("testdata", test.fixture)), transactionTrace)

			actionTraces := transactionTrace.ActionTraces

			actual, err := ToV1ActionTraceRaw(actionTraces[test.actionIndex], actionTraces, true)
			require.NoError(t, err)
			golden := filepath.Join("testdata", test.fixture+".golden")
			if os.Getenv("GOLDEN_UPDATE") != "" {
				ioutil.WriteFile(golden, actual, 0644)
			}

			expected, err := ioutil.ReadFile(golden)
			require.NoError(t, err, "unable to read golden file %s, use go test ./mdl --update to update golden files: %s", golden, err)

			if !assert.JSONEq(t, string(expected), string(actual), "actual:\n%s\nexpected:\n%s", actual, expected) {
				ioutil.WriteFile("/tmp/expected.json", expected, 0644)
				ioutil.WriteFile("/tmp/actual.json", actual, 0644)
			}
		})
	}
}

var jsonpbMarshaler = &jsonpb.Marshaler{
	Indent: "  ",
}

func protoJSONMarshalIndent(t *testing.T, message proto.Message) string {
	value, err := jsonpbMarshaler.MarshalToString(message)
	require.NoError(t, err)

	return value
}

func protoJSONUnmarshal(t *testing.T, data []byte, into proto.Message) {
	require.NoError(t, jsonpb.UnmarshalString(string(data), into))
}
