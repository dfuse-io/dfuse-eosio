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

package codec

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "net/http/pprof"

	"github.com/andreyvit/diff"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConsoleReaderPerformances(t *testing.T) {
	dmlogBenchmarkFile := os.Getenv("PERF_DMLOG_BENCHMARK_FILE")
	if dmlogBenchmarkFile == "" || !fileExists(dmlogBenchmarkFile) {
		t.Skipf("Environment variable 'PERF_DMLOG_BENCHMARK_FILE' not set or value %q is not an existing file", dmlogBenchmarkFile)
		return
	}

	go func() {
		fmt.Println("starting profiler")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			zlog.Info("listening localhost:6060", zap.Error(err))
		}
	}()

	fl, err := os.Open(dmlogBenchmarkFile)
	require.NoError(t, err)

	r, err := NewConsoleReader(fl)
	require.NoError(t, err)
	defer r.Close()

	count := 1999

	t0 := time.Now()

	for i := 0; i < count; i++ {
		blki, err := r.Read()
		require.NoError(t, err)

		blk := blki.(*pbcodec.Block)
		fmt.Fprintln(os.Stderr, "Processing block", blk.Num())
	}

	d1 := time.Since(t0)
	perSec := float64(count) / (float64(d1) / float64(time.Second))
	fmt.Printf("%d blocks in %s (%f blocks/sec)", count, d1, perSec)
}

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		deepMindFile string
	}{
		{"testdata/deep-mind.dmlog"},
		{"testdata/deep-mind-offchain-abi-decoding.dmlog"},
		{"testdata/dtrx-hard-fail.dmlog"},
		{"testdata/dtrx-soft-fail-onerror-not-present.dmlog"},
		{"testdata/dtrx-soft-fail-onerror-failed.dmlog"},
		{"testdata/dtrx-soft-fail-onerror-succeed.dmlog"},
	}

	for _, test := range tests {
		t.Run(test.deepMindFile, func(t *testing.T) {
			cr := testFileConsoleReader(t, test.deepMindFile)
			buf := &bytes.Buffer{}

			for {
				out, err := cr.Read()
				if out != nil && out.(*pbcodec.Block) != nil {
					blk := out.(*pbcodec.Block)

					if len(buf.Bytes()) != 0 {
						buf.Write([]byte("\n"))
					}

					buf.Write([]byte(protoJSONMarshalIndent(t, blk)))
				}

				if err == io.EOF {
					break
				}

				require.NoError(t, err)
			}

			goldenFile := test.deepMindFile + ".golden.json"
			if os.Getenv("GOLDEN_UPDATE") == "true" {
				ioutil.WriteFile(goldenFile, buf.Bytes(), os.ModePerm)
			}

			cnt, err := ioutil.ReadFile(goldenFile)
			require.NoError(t, err)

			if !assert.Equal(t, string(cnt), buf.String()) {
				t.Error("previous diff:\n" + unifiedDiff(t, cnt, buf.Bytes()))
			}
		})
	}
}

func unifiedDiff(t *testing.T, cnt1, cnt2 []byte) string {
	file1 := "/tmp/gotests-linediff-1"
	file2 := "/tmp/gotests-linediff-2"
	err := ioutil.WriteFile(file1, cnt1, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(file2, cnt2, 0600)
	require.NoError(t, err)

	cmd := exec.Command("diff", "-u", file1, file2)
	out, _ := cmd.Output()

	return string(out)
}

func TestGeneratePBBlocks(t *testing.T) {
	cr := testFileConsoleReader(t, "testdata/deep-mind.dmlog")

	for {
		out, err := cr.Read()
		if out != nil && out.(*pbcodec.Block) != nil {
			block := out.(*pbcodec.Block)

			outputFile, err := os.Create(fmt.Sprintf("testdata/pbblocks/battlefield-block.%d.deos.pb", block.Number))
			require.NoError(t, err)

			bytes, err := proto.Marshal(block)
			require.NoError(t, err)

			_, err = outputFile.Write(bytes)
			require.NoError(t, err)

			outputFile.Close()
		}

		if err == io.EOF {
			break
		}

		require.NoError(t, err)
	}
}

func testFileConsoleReader(t *testing.T, filename string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(filename)
	require.NoError(t, err)

	return testReaderConsoleReader(t, fl, func() { fl.Close() })
}

func testReaderConsoleReader(t *testing.T, reader io.Reader, closer func()) *ConsoleReader {
	t.Helper()

	consoleReader, err := NewConsoleReader(reader)
	require.NoError(t, err)

	return consoleReader
}

func Test_BlockRlimitOp(t *testing.T) {
	tests := []struct {
		line        string
		expected    *pbcodec.RlimitOp
		expectedErr error
	}{
		{
			`RLIMIT_OP CONFIG INS {"cpu_limit_parameters":{"target":20000,"max":200000,"periods":120,"max_multiplier":1000,"contract_rate":{"numerator":99,"denominator":100},"expand_rate":{"numerator":1000,"denominator":999}},"net_limit_parameters":{"target":104857,"max":1048576,"periods":120,"max_multiplier":1000,"contract_rate":{"numerator":99,"denominator":100},"expand_rate":{"numerator":1000,"denominator":999}},"account_cpu_usage_average_window":172800,"account_net_usage_average_window":172800}`,
			&pbcodec.RlimitOp{
				Operation: pbcodec.RlimitOp_OPERATION_INSERT,
				Kind: &pbcodec.RlimitOp_Config{
					Config: &pbcodec.RlimitConfig{
						CpuLimitParameters: &pbcodec.ElasticLimitParameters{
							Target:        20000,
							Max:           200000,
							Periods:       120,
							MaxMultiplier: 1000,
							ContractRate: &pbcodec.Ratio{
								Numerator:   99,
								Denominator: 100,
							},
							ExpandRate: &pbcodec.Ratio{
								Numerator:   1000,
								Denominator: 999,
							},
						},
						NetLimitParameters: &pbcodec.ElasticLimitParameters{
							Target:        104857,
							Max:           1048576,
							Periods:       120,
							MaxMultiplier: 1000,
							ContractRate: &pbcodec.Ratio{
								Numerator:   99,
								Denominator: 100,
							},
							ExpandRate: &pbcodec.Ratio{
								Numerator:   1000,
								Denominator: 999,
							},
						},
						AccountCpuUsageAverageWindow: 172800,
						AccountNetUsageAverageWindow: 172800,
					},
				},
			},
			nil,
		},
		{
			`RLIMIT_OP STATE INS {"average_block_net_usage":{"last_ordinal":1,"value_ex":2,"consumed":3},"average_block_cpu_usage":{"last_ordinal":4,"value_ex":5,"consumed":6},"pending_net_usage":7,"pending_cpu_usage":8,"total_net_weight":9,"total_cpu_weight":10,"total_ram_bytes":11,"virtual_net_limit":1048576,"virtual_cpu_limit":200000}`,
			&pbcodec.RlimitOp{
				Operation: pbcodec.RlimitOp_OPERATION_INSERT,
				Kind: &pbcodec.RlimitOp_State{
					State: &pbcodec.RlimitState{
						AverageBlockNetUsage: &pbcodec.UsageAccumulator{
							LastOrdinal: 1,
							ValueEx:     2,
							Consumed:    3,
						},
						AverageBlockCpuUsage: &pbcodec.UsageAccumulator{
							LastOrdinal: 4,
							ValueEx:     5,
							Consumed:    6,
						},
						PendingNetUsage: 7,
						PendingCpuUsage: 8,
						TotalNetWeight:  9,
						TotalCpuWeight:  10,
						TotalRamBytes:   11,
						VirtualNetLimit: 1048576,
						VirtualCpuLimit: 200000,
					},
				},
			},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readRlimitOp(test.line)

			require.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				require.Len(t, ctx.block.RlimitOps, 1)

				expected := protoJSONMarshalIndent(t, test.expected)
				actual := protoJSONMarshalIndent(t, ctx.block.RlimitOps[0])

				assert.JSONEq(t, expected, actual, diff.LineDiff(expected, actual))
			}
		})
	}
}

func Test_TraceRlimitOp(t *testing.T) {
	tests := []struct {
		line        string
		expected    *pbcodec.RlimitOp
		expectedErr error
	}{
		{
			`RLIMIT_OP ACCOUNT_LIMITS INS {"owner":"eosio.ram","net_weight":-1,"cpu_weight":-1,"ram_bytes":-1}`,
			&pbcodec.RlimitOp{
				Operation: pbcodec.RlimitOp_OPERATION_INSERT,
				Kind: &pbcodec.RlimitOp_AccountLimits{
					AccountLimits: &pbcodec.RlimitAccountLimits{
						Owner:     "eosio.ram",
						NetWeight: -1,
						CpuWeight: -1,
						RamBytes:  -1,
					},
				},
			},
			nil,
		},
		{
			`RLIMIT_OP ACCOUNT_USAGE UPD {"owner":"eosio","net_usage":{"last_ordinal":0,"value_ex":868696,"consumed":1},"cpu_usage":{"last_ordinal":0,"value_ex":572949,"consumed":101},"ram_usage":1181072}`,
			&pbcodec.RlimitOp{
				Operation: pbcodec.RlimitOp_OPERATION_UPDATE,
				Kind: &pbcodec.RlimitOp_AccountUsage{
					AccountUsage: &pbcodec.RlimitAccountUsage{
						Owner:    "eosio",
						NetUsage: &pbcodec.UsageAccumulator{LastOrdinal: 0, ValueEx: 868696, Consumed: 1},
						CpuUsage: &pbcodec.UsageAccumulator{LastOrdinal: 0, ValueEx: 572949, Consumed: 101},
						RamUsage: 1181072,
					},
				},
			},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readRlimitOp(test.line)

			require.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				require.Len(t, ctx.trx.RlimitOps, 1)

				expected := protoJSONMarshalIndent(t, test.expected)
				actual := protoJSONMarshalIndent(t, ctx.trx.RlimitOps[0])

				assert.JSONEq(t, expected, actual, diff.LineDiff(expected, actual))
			}
		})
	}
}

func Test_readPermOp(t *testing.T) {
	auth := &pbcodec.Authority{
		Threshold: 1,
		Accounts: []*pbcodec.PermissionLevelWeight{
			{
				Permission: &pbcodec.PermissionLevel{Actor: "eosio", Permission: "active"},
				Weight:     1,
			},
		},
	}

	tests := []struct {
		line        string
		expected    *pbcodec.PermOp
		expectedErr error
	}{
		{
			`PERM_OP INS 0 {"owner":"eosio.ins","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_INSERT,
				ActionIndex: 0,
				OldPerm:     nil,
				NewPerm: &pbcodec.PermissionObject{
					Owner:       "eosio.ins",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP UPD 0 {"old":{"owner":"eosio.old","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}},"new":{"owner":"eosio.new","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_UPDATE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					Owner:       "eosio.old",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
				NewPerm: &pbcodec.PermissionObject{
					Owner:       "eosio.new",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP REM 0 {"owner":"eosio.rem","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_REMOVE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					Owner:       "eosio.rem",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
				NewPerm: nil,
			},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readPermOp(test.line)

			require.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				require.Len(t, ctx.trx.PermOps, 1)

				expected := protoJSONMarshalIndent(t, test.expected)
				actual := protoJSONMarshalIndent(t, ctx.trx.PermOps[0])

				assert.JSONEq(t, expected, actual, diff.LineDiff(expected, actual))
			}
		})
	}
}

func mustTimeParse(input string) time.Time {
	value, err := time.Parse("2006-01-02T15:04:05", input)
	if err != nil {
		panic(err)
	}

	return value
}

func reader(in string) io.Reader {
	return bytes.NewReader([]byte(in))
}

var jsonpbMarshaler = &jsonpb.Marshaler{
	Indent: "  ",
}

func protoJSONMarshalIndent(t *testing.T, message proto.Message) string {
	value, err := jsonpbMarshaler.MarshalToString(message)
	require.NoError(t, err)

	return value
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	return !info.IsDir()
}
