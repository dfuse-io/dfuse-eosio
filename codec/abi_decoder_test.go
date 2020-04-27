package codec

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestABIDecoder(t *testing.T) {
	type expectedTrace struct {
		path      string
		jsonValue string
	}

	in := func(blocks ...*pbcodec.Block) []*pbcodec.Block {
		return blocks
	}

	eosioTokenABI1 := readABI(t, "token.1.abi.json")
	eosioTokenABI2 := readABI(t, "token.2.abi.json")
	eosioTestABI1 := readABI(t, "test.1.abi.json")
	eosioTestABI2 := readABI(t, "test.2.abi.json")
	eosioTestABI3 := readABI(t, "test.3.abi.json")
	// eosioNekotABI1 := readABI(t, "nekot.1.abi.json")

	tests := []struct {
		name           string
		blocks         []*pbcodec.Block
		expectedTraces []expectedTrace
	}{
		{
			name: "setabi and usage, same trace",
			blocks: in(testBlock(t, "00000002aa", "00000001aa",
				trxTrace(t, actionSetABI(t, "test", 1, eosioTestABI1), action(t, "test:test:act1", 2, eosioTestABI1, `{"from":"test1"}`)),
			)),
			expectedTraces: []expectedTrace{
				{"block 0/trace 0/action 1", `{"from":"test1"}`},
			},
		},
		{
			name: "setabi and usage, same block, two traces",
			blocks: in(testBlock(t, "00000002aa", "00000001aa",
				trxTrace(t, actionSetABI(t, "test", 1, eosioTestABI1)),
				trxTrace(t, action(t, "test:test:act1", 2, eosioTestABI1, `{"from":"test1"}`)),
			)),
			expectedTraces: []expectedTrace{
				{"block 0/trace 1/action 0", `{"from":"test1"}`},
			},
		},
		{
			name: "setabi and usage, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionSetABI(t, "test", 1, eosioTestABI1)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, action(t, "test:test:act1", 2, eosioTestABI1, `{"from":"test1"}`)),
				),
			),
			expectedTraces: []expectedTrace{
				{"block 1/trace 0/action 0", `{"from":"test1"}`},
			},
		},

		{
			name: "set multiple times, within same transaction, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t,
						actionSetABI(t, "test", 1, eosioTestABI1),
						action(t, "test:test:act1", 2, eosioTestABI1, `{"from":"test1"}`),
						actionSetABI(t, "test", 3, eosioTestABI2),
						action(t, "test:test:act2", 4, eosioTestABI2, `{"to":20}`),
						actionSetABI(t, "test", 5, eosioTestABI3),
					),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, action(t, "test:test:act3", 6, eosioTestABI3, `{"quantity":"1.0 EOS"}`)),
				),
			),
			expectedTraces: []expectedTrace{
				{"block 0/trace 0/action 1", `{"from":"test1"}`},
				{"block 0/trace 0/action 3", `{"to":20}`},
				{"block 1/trace 0/action 0", `{"quantity":"1.0 EOS"}`},
			},
		},

		{
			name: "set multiple times, across transactions, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionSetABI(t, "test", 1, eosioTestABI1)),
					trxTrace(t, action(t, "test:test:act1", 2, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, actionSetABI(t, "test", 3, eosioTestABI2)),
					trxTrace(t, action(t, "test:test:act2", 4, eosioTestABI2, `{"to":20}`)),
					trxTrace(t, actionSetABI(t, "test", 5, eosioTestABI3)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, action(t, "test:test:act3", 6, eosioTestABI3, `{"quantity":"1.0 EOS"}`)),
				),
			),
			expectedTraces: []expectedTrace{
				{"block 0/trace 1/action 0", `{"from":"test1"}`},
				{"block 0/trace 3/action 0", `{"to":20}`},
				{"block 1/trace 0/action 0", `{"quantity":"1.0 EOS"}`},
			},
		},

		{
			name: "fork multiple block",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionSetABI(t, "test", 1, eosioTestABI1)),
					trxTrace(t, actionSetABI(t, "token", 2, eosioTokenABI1)),
				),
				testBlock(t, "00000002bb", "00000001aa",
					trxTrace(t, action(t, "test:test:act1", 3, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, actionSetABI(t, "test", 4, eosioTestABI2)),
					trxTrace(t, action(t, "test:test:act2", 5, eosioTestABI2, `{"to":20}`)),
				),
				testBlock(t, "00000003bb", "00000002bb",
					trxTrace(t, action(t, "test:test:act2", 6, eosioTestABI2, `{"to":20}`)),
					trxTrace(t, actionSetABI(t, "token", 7, eosioTokenABI2)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, action(t, "test:test:act1", 3, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, action(t, "token:token:transfer", 4, eosioTokenABI1, `{"to":"transfer3"}`)),
				),
			),
			expectedTraces: []expectedTrace{
				{"block 1/trace 0/action 0", `{"from":"test1"}`},
				{"block 1/trace 2/action 0", `{"to":20}`},
				{"block 2/trace 0/action 0", `{"to":20}`},
				{"block 3/trace 0/action 0", `{"from":"test1"}`},
				{"block 3/trace 1/action 0", `{"to":"transfer3"}`},
			},
		},

		// TODO: Add those tests
		//        - ensures "hard-coded" system methods like `setabi`, `setcode` always work?
		//        - transaction soft_fail, set abi works inside transaction, but not outside
		//        - transaction soft_fail follow by success onerror correctly records ABI for next
		//        - transaction soft_fail follow by failed onerror works inside transaction, but not outside
	}

	toString := func(in proto.Message) string {
		out, err := (&jsonpb.Marshaler{}).MarshalToString(in)
		require.NoError(t, err)

		return out
	}

	pathRegex := regexp.MustCompile("block ([0-9]+)/trace ([0-9]+)/action ([0-9]+)")
	toInt := func(in string) int {
		out, err := strconv.ParseInt(in, 10, 32)
		require.NoError(t, err)

		return int(out)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := newABIDecoder()

			for _, block := range test.blocks {
				err := decoder.startBlock(context.Background(), block.Num())
				require.NoError(t, err)

				for _, trxTrace := range block.TransactionTraces {
					err := decoder.processTransaction(trxTrace)
					require.NoError(t, err)
				}

				// This should wait for all decoding in the block to terminate
				err = decoder.endBlock(block.AsRef())
				require.NoError(t, err)
			}

			for _, expect := range test.expectedTraces {
				match := pathRegex.FindAllStringSubmatch(expect.path, -1)[0]
				block := test.blocks[toInt(match[1])]
				trace := block.TransactionTraces[toInt(match[2])]
				actionTrace := trace.ActionTraces[toInt(match[3])]

				require.NotEmpty(t, actionTrace.Action.JsonData, toString(actionTrace))
				assert.JSONEq(t, expect.jsonValue, actionTrace.Action.JsonData)
			}
		})
	}
}

func testBlock(t *testing.T, blkID string, previousBlkID string, trxTraceJSONs ...string) *pbcodec.Block {
	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbcodec.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace), "actual string:\n"+trxTraceJSON)

		trxTraces[i] = trxTrace
	}

	pbblock := &pbcodec.Block{
		Id:                blkID,
		Number:            eos.BlockNum(blkID),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbcodec.BlockHeader{
		Previous:  previousBlkID,
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

func trxTrace(t *testing.T, elements ...proto.Message) string {
	trace := &pbcodec.TransactionTrace{
		Receipt: &pbcodec.TransactionReceiptHeader{
			Status: pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
		},
	}

	for _, element := range elements {
		switch v := element.(type) {
		case *pbcodec.ActionTrace:
			trace.ActionTraces = append(trace.ActionTraces, v)
		case *pbcodec.DBOp:
			trace.DbOps = append(trace.DbOps, v)
		case *pbcodec.TableOp:
			trace.TableOps = append(trace.TableOps, v)
		}
	}

	out, err := jsonpb.MarshalIndentToString(trace, "")
	require.NoError(t, err)

	return out
}

func action(t *testing.T, tripletName string, globalSequence uint64, abi *eos.ABI, data string) *pbcodec.ActionTrace {
	parts := strings.Split(tripletName, ":")
	receiver := parts[0]
	account := parts[1]
	actionName := parts[2]

	rawData, err := abi.EncodeAction(eos.ActionName(actionName), []byte(data))
	require.NoError(t, err)

	return &pbcodec.ActionTrace{
		Receiver: receiver,
		Receipt: &pbcodec.ActionReceipt{
			Receiver:       receiver,
			GlobalSequence: globalSequence,
		},
		Action: &pbcodec.Action{
			Account: account,
			Name:    actionName,
			RawData: rawData,
		},
	}
}

func actionSetABI(t *testing.T, account string, globalSequence uint64, abi *eos.ABI) *pbcodec.ActionTrace {
	abiData, err := eos.MarshalBinary(abi)
	require.NoError(t, err)

	setABI := &system.SetABI{Account: eos.AccountName(account), ABI: eos.HexBytes(abiData)}
	rawData, err := eos.MarshalBinary(setABI)
	require.NoError(t, err)

	return &pbcodec.ActionTrace{
		Receiver: "eosio",
		Receipt: &pbcodec.ActionReceipt{
			Receiver:       "eosio",
			GlobalSequence: globalSequence,
		},
		Action: &pbcodec.Action{
			Account: "eosio",
			Name:    "setabi",
			RawData: rawData,
		},
	}
}
