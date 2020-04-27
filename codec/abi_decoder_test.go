package codec

import (
	"context"
	"encoding/hex"
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
	type expectation struct {
		path string
		// If value is a hex string, it expects `rawData` to match it, otherwise, it expects `jsonData` to match it
		value string
	}

	type testData struct {
		name         string
		blocks       []*pbcodec.Block
		expectations []expectation
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

	softFailStatus := pbcodec.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL
	hardFailStatus := pbcodec.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL

	tests := []testData{
		{
			name: "setabi and usage, same trace",
			blocks: in(testBlock(t, "00000002aa", "00000001aa",
				trxTrace(t,
					actionTraceSetABI(t, "test", 0, 1, eosioTestABI1),
					actionTrace(t, "test:test:act1", 1, 2, eosioTestABI1, `{"from":"test1"}`),
				),
			)),
			expectations: []expectation{
				{"block 0/trace 0/action 1", `{"from":"test1"}`},
			},
		},
		{
			name: "setabi and usage, same block, two traces",
			blocks: in(testBlock(t, "00000002aa", "00000001aa",
				trxTrace(t, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
				trxTrace(t, actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test1"}`)),
			)),
			expectations: []expectation{
				{"block 0/trace 1/action 0", `{"from":"test1"}`},
			},
		},
		{
			name: "setabi and usage, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test1"}`)),
				),
			),
			expectations: []expectation{
				{"block 1/trace 0/action 0", `{"from":"test1"}`},
			},
		},
		{
			name: "set multiple times, within same transaction, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t,
						actionTraceSetABI(t, "test", 0, 1, eosioTestABI1),
						actionTrace(t, "test:test:act1", 1, 2, eosioTestABI1, `{"from":"test1"}`),
						actionTraceSetABI(t, "test", 2, 3, eosioTestABI2),
						actionTrace(t, "test:test:act2", 2, 4, eosioTestABI2, `{"to":20}`),
						actionTraceSetABI(t, "test", 4, 5, eosioTestABI3),
					),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act3", 0, 6, eosioTestABI3, `{"quantity":"1.0 EOS"}`)),
				),
			),
			expectations: []expectation{
				{"block 0/trace 0/action 1", `{"from":"test1"}`},
				{"block 0/trace 0/action 3", `{"to":20}`},
				{"block 1/trace 0/action 0", `{"quantity":"1.0 EOS"}`},
			},
		},
		{
			name: "set multiple times, across transactions, two different blocks",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
					trxTrace(t, actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, actionTraceSetABI(t, "test", 0, 3, eosioTestABI2)),
					trxTrace(t, actionTrace(t, "test:test:act2", 0, 4, eosioTestABI2, `{"to":20}`)),
					trxTrace(t, actionTraceSetABI(t, "test", 0, 5, eosioTestABI3)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act3", 0, 6, eosioTestABI3, `{"quantity":"1.0 EOS"}`)),
				),
			),
			expectations: []expectation{
				{"block 0/trace 1/action 0", `{"from":"test1"}`},
				{"block 0/trace 3/action 0", `{"to":20}`},
				{"block 1/trace 0/action 0", `{"quantity":"1.0 EOS"}`},
			},
		},
		{
			name: "fork multiple block",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
					trxTrace(t, actionTraceSetABI(t, "token", 0, 2, eosioTokenABI1)),
				),
				testBlock(t, "00000002bb", "00000001aa",
					trxTrace(t, actionTrace(t, "test:test:act1", 0, 3, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, actionTraceSetABI(t, "test", 0, 4, eosioTestABI2)),
					trxTrace(t, actionTrace(t, "test:test:act2", 0, 5, eosioTestABI2, `{"to":20}`)),
				),
				testBlock(t, "00000003bb", "00000002bb",
					trxTrace(t, actionTrace(t, "test:test:act2", 0, 6, eosioTestABI2, `{"to":20}`)),
					trxTrace(t, actionTraceSetABI(t, "token", 0, 7, eosioTokenABI2)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act1", 0, 3, eosioTestABI1, `{"from":"test1"}`)),
					trxTrace(t, actionTrace(t, "token:token:transfer", 0, 4, eosioTokenABI1, `{"to":"transfer3"}`)),
				),
			),
			expectations: []expectation{
				{"block 1/trace 0/action 0", `{"from":"test1"}`},
				{"block 1/trace 2/action 0", `{"to":20}`},
				{"block 2/trace 0/action 0", `{"to":20}`},
				{"block 3/trace 0/action 0", `{"from":"test1"}`},
				{"block 3/trace 1/action 0", `{"to":"transfer3"}`},
			},
		},
		{
			name: "fail transaction, does not save ABI",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, hardFailStatus, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test1"}`)),
				),
			),
			expectations: []expectation{
				{"block 1/trace 0/action 0", `000000008090b1ca`},
			},
		},
		{
			name: "fail transaction, still works from failed transaction but does not record ABI",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, actionTraceSetABI(t, "test", 0, 1, eosioTestABI1)),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, hardFailStatus,
						actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test1"}`),
						actionTraceSetABI(t, "test", 1, 3, eosioTestABI2),
						actionTrace(t, "test:test:act2", 2, 4, eosioTestABI2, `{"to":1}`),
						actionTrace(t, "test:test:act2", 3, 5, eosioTestABI2, `{"to":2}`),
						actionTraceSetABI(t, "test", 4, 6, eosioTestABI3),
						actionTraceFail(t, "test:test:act3", 5, eosioTestABI3, `{"quantity":"1.0000 EOS"}`),
					),
				),
				testBlock(t, "00000004aa", "00000003aa",
					trxTrace(t,
						actionTrace(t, "test:test:act1", 0, 2, eosioTestABI1, `{"from":"test3"}`),
						// Let's assume there is a bunch of transaction in-between, so we test that no recording actually occurred!
						actionTrace(t, "test:test:act1", 1, 7, eosioTestABI1, `{"from":"test4"}`),
					),
				),
			),
			expectations: []expectation{
				{"block 1/trace 0/action 0", `{"from":"test1"}`},
				{"block 1/trace 0/action 2", `{"to":1}`},
				{"block 1/trace 0/action 3", `{"to":2}`},
				{"block 1/trace 0/action 5", `{"quantity":"1.0000 EOS"}`},
				{"block 2/trace 0/action 0", `{"from":"test3"}`},
				{"block 2/trace 0/action 1", `{"from":"test4"}`},
			},
		},

		{
			name: "soft_fail onerror, still records ABI",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, softFailStatus,
						actionTrace(t, "eosio:eosio:onerror", 0, 1, nil, ""),
						actionTraceSetABI(t, "test", 1, 2, eosioTestABI2),
						actionTrace(t, "test:test:act2", 2, 3, eosioTestABI2, `{"to":1}`),
						actionTraceSetABI(t, "test", 3, 4, eosioTestABI3),
					),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act3", 0, 5, eosioTestABI3, `{"quantity":"1.0000 EOS"}`)),
				),
			),
			expectations: []expectation{
				{"block 0/trace 0/action 2", `{"to":1}`},
				{"block 1/trace 0/action 0", `{"quantity":"1.0000 EOS"}`},
			},
		},
		{
			name: "hard_fail onerror, still works from failed transaction but does not record ABI",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t, hardFailStatus,
						actionTrace(t, "eosio:eosio:onerror", 0, 1, nil, ""),
						actionTraceSetABI(t, "test", 1, 2, eosioTestABI2),
						actionTrace(t, "test:test:act2", 2, 3, eosioTestABI2, `{"to":1}`),
						actionTraceSetABI(t, "test", 3, 4, eosioTestABI3),
						actionTraceFail(t, "any:any:any", 4, nil, ""),
					),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t, actionTrace(t, "test:test:act3", 0, 1, eosioTestABI3, `{"quantity":"1.0000 EOS"}`)),
					// Let's assume there is a bunch of transaction in-between, so we test that no recording actually occurred!
					trxTrace(t, actionTrace(t, "test:test:act3", 0, 8, eosioTestABI3, `{"quantity":"2.0000 EOS"}`)),
				),
			),
			expectations: []expectation{
				{"block 0/trace 0/action 2", `{"to":1}`},
				{"block 1/trace 0/action 0", `102700000000000004454f5300000000`},
				{"block 1/trace 1/action 0", `204e00000000000004454f5300000000`},
			},
		},

		{
			name: "dtrx ops are correctly decoded",
			blocks: in(
				testBlock(t, "00000002aa", "00000001aa",
					trxTrace(t,
						actionTraceSetABI(t, "test", 0, 1, eosioTestABI1),
						actionTraceSetABI(t, "token", 1, 2, eosioTokenABI1),
						actionTrace(t, "test:test:act1", 2, 3, eosioTestABI1, `{"from":"block1"}`),

						// A dtrx op created by action index 2
						dtrxOp(t, 2, signedTrx(t,
							cfaAction(t, "token:transfer", eosioTokenABI1, `{"to":"someone"}`),
							action(t, "test:act1", eosioTestABI1, `{"from":"inner1"}`),
						)),
					),
				),
				testBlock(t, "00000003aa", "00000002aa",
					trxTrace(t,
						actionTrace(t, "test:test:act1", 0, 4, eosioTestABI1, `{"from":"block2"}`),

						// A dtrx op created by action index 0
						dtrxOp(t, 0, signedTrx(t,
							cfaAction(t, "token:transfer", eosioTokenABI1, `{"to":"somelse"}`),
							action(t, "test:act1", eosioTestABI1, `{"from":"inner2"}`),
						)),
					),
				),
			),
			expectations: []expectation{
				{"block 0/trace 0/action 2", `{"from":"block1"}`},
				{"block 0/trace 0/dtrxOp 0/action 0", `{"from":"inner1"}`},
				{"block 0/trace 0/dtrxOp 0/cfaAction 0", `{"to":"someone"}`},

				{"block 1/trace 0/dtrxOp 0/action 0", `{"from":"inner2"}`},
				{"block 1/trace 0/dtrxOp 0/cfaAction 0", `{"to":"somelse"}`},
			},
		},

		// TODO: Add those tests
		//        - ensures "hard-coded" system methods like `setabi`, `setcode` always work?
	}

	toString := func(in proto.Message) string {
		out, err := (&jsonpb.Marshaler{}).MarshalToString(in)
		require.NoError(t, err)

		return out
	}

	hexRegex := regexp.MustCompile("^[0-9a-fA-F]+$")
	actionTraceRegex := regexp.MustCompile("^block (\\d+)/trace (\\d+)/action (\\d+)$")
	dtrxOpRegex := regexp.MustCompile("^block (\\d+)/trace (\\d+)/dtrxOp (\\d+)/(action|cfaAction) (\\d+)$")

	toInt := func(in string) int {
		out, err := strconv.ParseInt(in, 10, 32)
		require.NoError(t, err)

		return int(out)
	}

	extractTrace := func(testData *testData, regexMatch []string) (block *pbcodec.Block, trace *pbcodec.TransactionTrace) {
		block = testData.blocks[toInt(regexMatch[1])]
		trace = block.TransactionTraces[toInt(regexMatch[2])]
		return
	}

	assertMatchAction := func(expected string, action *pbcodec.Action) {
		if hexRegex.MatchString(expected) {
			require.Equal(t, expected, hex.EncodeToString(action.RawData), toString(action))
			require.Empty(t, action.JsonData, toString(action))
		} else {
			require.NotEmpty(t, action.JsonData, toString(action))
			assert.JSONEq(t, expected, action.JsonData)
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := newABIDecoder()

			for _, block := range test.blocks {
				maybePrintBlock(t, block)

				err := decoder.startBlock(context.Background(), block.Num())
				require.NoError(t, err)

				for _, trxTrace := range block.TransactionTraces {
					err := decoder.processTransaction(trxTrace)
					require.NoError(t, err)
				}

				// This should wait for all decoding in the block to terminate
				err = decoder.endBlock(block)
				require.NoError(t, err)
			}

			for _, expect := range test.expectations {
				var match []string

				if match = fullMatchRegex(actionTraceRegex, expect.path); match != nil {
					_, trace := extractTrace(&test, match)
					assertMatchAction(expect.value, trace.ActionTraces[toInt(match[3])].Action)
					continue
				}

				if match = fullMatchRegex(dtrxOpRegex, expect.path); match != nil {
					_, trace := extractTrace(&test, match)
					dtrxOp := trace.DtrxOps[toInt(match[3])]

					if match[4] == "cfaAction" {
						assertMatchAction(expect.value, dtrxOp.Transaction.Transaction.ContextFreeActions[toInt(match[5])])
					} else if match[4] == "action" {
						assertMatchAction(expect.value, dtrxOp.Transaction.Transaction.Actions[toInt(match[5])])
					}
					continue
				}

				assert.Fail(t, "Unable to assert unknown expectation", "Expecation path %q not matching any assertion regex", expect.path)
			}
		})
	}
}

func fullMatchRegex(regex *regexp.Regexp, content string) []string {
	match := regex.FindAllStringSubmatch(content, -1)
	if match == nil {
		return nil
	}

	return match[0]
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

	return pbblock
}

func trxTrace(t *testing.T, elements ...interface{}) string {
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
		case *pbcodec.DTrxOp:
			trace.DtrxOps = append(trace.DtrxOps, v)
		case *pbcodec.TableOp:
			trace.TableOps = append(trace.TableOps, v)
		case pbcodec.TransactionStatus:
			trace.Receipt.Status = v
		}
	}

	out, err := jsonpb.MarshalIndentToString(trace, "")
	require.NoError(t, err)

	return out
}

func signedTrx(t *testing.T, elements ...interface{}) *pbcodec.SignedTransaction {
	signedTrx := &pbcodec.SignedTransaction{}
	signedTrx.Transaction = trx(t, elements...)

	return signedTrx
}

type ContextFreeAction *pbcodec.Action

func trx(t *testing.T, elements ...interface{}) *pbcodec.Transaction {
	trx := &pbcodec.Transaction{}

	for _, element := range elements {
		switch v := element.(type) {
		case *pbcodec.Action:
			trx.Actions = append(trx.Actions, v)
		case ContextFreeAction:
			trx.ContextFreeActions = append(trx.ContextFreeActions, (*pbcodec.Action)(v))
		}
	}

	return trx
}

func actionTrace(t *testing.T, tripletName string, executionIndex uint32, globalSequence uint64, abi *eos.ABI, data string) *pbcodec.ActionTrace {
	parts := strings.Split(tripletName, ":")
	receiver := parts[0]
	account := parts[1]
	actionName := parts[2]

	return &pbcodec.ActionTrace{
		ExecutionIndex: executionIndex,
		Receiver:       receiver,
		Receipt: &pbcodec.ActionReceipt{
			Receiver:       receiver,
			GlobalSequence: globalSequence,
		},
		Action: action(t, account+":"+actionName, abi, data),
	}
}

func actionTraceFail(t *testing.T, tripletName string, executionIndex uint32, abi *eos.ABI, data string) *pbcodec.ActionTrace {
	out := actionTrace(t, tripletName, executionIndex, 0, abi, data)
	out.Receipt = nil

	return out
}

func actionTraceSetABI(t *testing.T, account string, executionIndex uint32, globalSequence uint64, abi *eos.ABI) *pbcodec.ActionTrace {
	abiData, err := eos.MarshalBinary(abi)
	require.NoError(t, err)

	setABI := &system.SetABI{Account: eos.AccountName(account), ABI: eos.HexBytes(abiData)}
	rawData, err := eos.MarshalBinary(setABI)
	require.NoError(t, err)

	return &pbcodec.ActionTrace{
		ExecutionIndex: executionIndex,
		Receiver:       "eosio",
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

func cfaAction(t *testing.T, pairName string, abi *eos.ABI, data string) ContextFreeAction {
	return ContextFreeAction(action(t, pairName, abi, data))
}

func action(t *testing.T, pairName string, abi *eos.ABI, data string) *pbcodec.Action {
	parts := strings.Split(pairName, ":")
	account := parts[0]
	actionName := parts[1]

	var rawData []byte
	if abi != nil && data != "" {
		var err error
		rawData, err = abi.EncodeAction(eos.ActionName(actionName), []byte(data))
		require.NoError(t, err)
	}

	return &pbcodec.Action{
		Account: account,
		Name:    actionName,
		RawData: rawData,
	}
}

func dtrxOp(t *testing.T, actionIndex uint32, signedTrx *pbcodec.SignedTransaction) *pbcodec.DTrxOp {
	op := &pbcodec.DTrxOp{
		ActionIndex: actionIndex,
		Transaction: signedTrx,
	}

	return op
}

func maybePrintBlock(t *testing.T, block *pbcodec.Block) {
	if os.Getenv("DEBUG") == "" && os.Getenv("TRACE") != "true" {
		return
	}

	marshaler := &jsonpb.Marshaler{}
	out, err := marshaler.MarshalToString(block)
	require.NoError(t, err)

	// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
	normalizedOut := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(out), &normalizedOut))

	zlog.Debug("processing test block", zap.Any("block", normalizedOut))
}
