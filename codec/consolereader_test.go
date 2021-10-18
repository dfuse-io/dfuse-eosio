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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "net/http/pprof"

	"github.com/andreyvit/diff"
	eosio_v2_0 "github.com/dfuse-io/dfuse-eosio/codec/eosio/v2.0"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/jsonpb"
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
		name          string
		deepMindFile  string
		includeBlock  func(block *pbcodec.Block) bool
		readerOptions []ConsoleReaderOption
	}{
		{"full", "testdata/deep-mind.dmlog", nil, nil},
		{"full-2.1.x", "testdata/deep-mind-2.1.x.dmlog", nil, nil},
		{"max-console-log", "testdata/deep-mind.dmlog", blockWithConsole, []ConsoleReaderOption{LimitConsoleLength(10)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := testFileConsoleReader(t, test.deepMindFile, test.readerOptions...)
			buf := &bytes.Buffer{}

			for {
				out, err := cr.Read()
				if out != nil && out.(*pbcodec.Block) != nil {
					blk := out.(*pbcodec.Block)
					if test.includeBlock != nil && !test.includeBlock(blk) {
						continue
					}

					if len(buf.Bytes()) != 0 {
						buf.Write([]byte("\n"))
					}

					buf.Write([]byte(protoJSONMarshalIndent(t, blk)))
				}

				if err == io.EOF {
					break
				}

				if err != nil {
					// It appears that since our error can be quite large, the `require.NoError`
					// seems to not print it in full when an error occurred. In fact, it only
					// prints an unexpected error occurred. To ensure the error is debuggable,
					// we print it first when an error is present.
					fmt.Println(err)
				}
				require.NoError(t, err)
			}

			goldenFile := filepath.Join("testdata", test.name+".golden.json")
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

	// Create the folder, it might not exists in some contexts
	err := os.MkdirAll("testdata/pbblocks", os.ModePerm)
	require.NoError(t, err)

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

func testFileConsoleReader(t *testing.T, filename string, options ...ConsoleReaderOption) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(filename)
	require.NoError(t, err)

	return testReaderConsoleReader(t, fl, func() { fl.Close() }, options...)
}

func testReaderConsoleReader(t *testing.T, reader io.Reader, closer func(), options ...ConsoleReaderOption) *ConsoleReader {
	t.Helper()

	consoleReader, err := NewConsoleReader(reader, options...)
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

func Test_readKvOp(t *testing.T) {
	toBytes := func(in string) []byte {
		out, err := hex.DecodeString(in)
		require.NoError(t, err)

		return out
	}

	tests := []struct {
		name        string
		line        string
		expected    *pbcodec.KVOp
		expectedErr error
	}{
		{
			"insert standard",
			`KV_OP INS 0 battlefield john b6876876616c7565 78c159f95d672d640539`,
			&pbcodec.KVOp{
				Operation:   pbcodec.KVOp_OPERATION_INSERT,
				ActionIndex: 0,
				Code:        "battlefield",
				OldPayer:    "",
				NewPayer:    "john",
				Key:         toBytes("b6876876616c7565"),
				OldData:     nil,
				NewData:     toBytes("78c159f95d672d640539"),
			},
			nil,
		},
		{
			"update standard",
			`KV_OP UPD 1 battlefield jane b6876876616c7565 78c159f95d672d640539:78c159f95d672d640561`,
			&pbcodec.KVOp{
				Operation:   pbcodec.KVOp_OPERATION_UPDATE,
				ActionIndex: 1,
				Code:        "battlefield",
				// Unavailable in data ...
				OldPayer: "",
				NewPayer: "jane",
				Key:      toBytes("b6876876616c7565"),
				OldData:  toBytes("78c159f95d672d640539"),
				NewData:  toBytes("78c159f95d672d640561"),
			},
			nil,
		},
		{
			"remove standard",
			`KV_OP REM 2 battlefield jane b6876876616c7565 78c159f95d672d640561`,
			&pbcodec.KVOp{
				Operation:   pbcodec.KVOp_OPERATION_REMOVE,
				ActionIndex: 2,
				Code:        "battlefield",
				OldPayer:    "jane",
				NewPayer:    "",
				Key:         toBytes("b6876876616c7565"),
				OldData:     toBytes("78c159f95d672d640561"),
				NewData:     nil,
			},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readKVOp(test.line)

			require.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				require.Len(t, ctx.trx.KvOps, 1)

				expected := protoJSONMarshalIndent(t, test.expected)
				actual := protoJSONMarshalIndent(t, ctx.trx.KvOps[0])

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
			`PERM_OP INS 0 {"parent":1,"owner":"eosio.ins","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_INSERT,
				ActionIndex: 0,
				OldPerm:     nil,
				NewPerm: &pbcodec.PermissionObject{
					ParentId:    1,
					Owner:       "eosio.ins",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP UPD 0 {"old":{"parent":2,"owner":"eosio.old","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}},"new":{"parent":3,"owner":"eosio.new","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_UPDATE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					ParentId:    2,
					Owner:       "eosio.old",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
				NewPerm: &pbcodec.PermissionObject{
					ParentId:    3,
					Owner:       "eosio.new",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP REM 0 {"parent":4,"owner":"eosio.rem","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_REMOVE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					ParentId:    4,
					Owner:       "eosio.rem",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
				NewPerm: nil,
			},
			nil,
		},

		// New format
		{
			`PERM_OP INS 0 2 {"parent":1,"owner":"eosio.ins","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_INSERT,
				ActionIndex: 0,
				OldPerm:     nil,
				NewPerm: &pbcodec.PermissionObject{
					Id:          2,
					ParentId:    1,
					Owner:       "eosio.ins",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP UPD 0 4 {"old":{"parent":2,"owner":"eosio.old","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}},"new":{"parent":3,"owner":"eosio.new","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_UPDATE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					Id:          4,
					ParentId:    2,
					Owner:       "eosio.old",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
				NewPerm: &pbcodec.PermissionObject{
					Id:          4,
					ParentId:    3,
					Owner:       "eosio.new",
					Name:        "prod.major",
					LastUpdated: mustProtoTimestamp(mustTimeParse("2018-06-08T08:08:08.888")),
					Authority:   auth,
				},
			},
			nil,
		},
		{
			`PERM_OP REM 0 3 {"parent":4,"owner":"eosio.rem","name":"prod.major","last_updated":"2018-06-08T08:08:08.888","auth":{"threshold":1,"keys":[],"accounts":[{"permission":{"actor":"eosio","permission":"active"},"weight":1}],"waits":[]}}`,
			&pbcodec.PermOp{
				Operation:   pbcodec.PermOp_OPERATION_REMOVE,
				ActionIndex: 0,
				OldPerm: &pbcodec.PermissionObject{
					Id:          3,
					ParentId:    4,
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

func Test_readABIDump_Start(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		expectedErr error
	}{
		{
			"version 12",
			`ABIDUMP START`,
			nil,
		},
		{
			"version 13",
			`ABIDUMP START 44 500`,
			nil,
		},
		{
			"version 13, invalid block num",
			`ABIDUMP START s44 500`,
			errors.New(`block_num is not a valid number, got: "s44"`),
		},
		{
			"version 13, invalid global sequence num",
			`ABIDUMP START 44 s500`,
			errors.New(`global_sequence_num is not a valid number, got: "s500"`),
		},
		{
			"invalid number of field",
			`ABIDUMP START 44`,
			errors.New(`expected to have either 2 or 4 fields, got 3`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readABIStart(test.line)

			require.Equal(t, test.expectedErr, err)
		})
	}
}

func Test_readDeepMindVersion(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		expectedErr error
	}{
		{
			"version 12",
			`DEEP_MIND_VERSION 12`,
			nil,
		},
		{
			"version 13",
			`DEEP_MIND_VERSION 13 0`,
			nil,
		},
		{
			"version 13, unsupported",
			`DEEP_MIND_VERSION 14 0`,
			errors.New("deep mind reported version 14, but this reader supports only 12, 13"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newParseCtx()
			_, err := ctx.readDeepmindVersion(test.line)

			require.Equal(t, test.expectedErr, err)
		})
	}
}

func Test_readABIDump_ABI(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		expectedErr error
	}{
		{
			"version 12",
			`ABIDUMP ABI 44 eosio AAAAAAAAAAAA`,
			nil,
		},
		{
			"version 13",
			`ABIDUMP ABI eosio AAAAAAAAAAAA`,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newParseCtx()
			err := ctx.readABIDump(test.line)

			require.Equal(t, test.expectedErr, err)

			if test.expectedErr == nil {
				contractABI := ctx.abiDecoder.cache.findABI("eosio", 0)
				assert.NotNil(t, contractABI)
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

func blockWithConsole(block *pbcodec.Block) bool {
	for _, trxTrace := range block.TransactionTraces() {
		for _, actTrace := range trxTrace.ActionTraces {
			if len(actTrace.Console) > 0 {
				return true
			}
		}
	}

	return false
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		hydrator:   eosio_v2_0.NewHydrator(zlog),
		abiDecoder: newABIDecoder(),
		block:      &pbcodec.Block{},
		trx:        &pbcodec.TransactionTrace{},
	}
}
