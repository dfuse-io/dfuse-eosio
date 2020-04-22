package codec

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
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

func TestABICache_Shared(t *testing.T) {
	cache := newABIDecoder()
	blocks := sharedBlocks(t)

	for _, block := range blocks {
		err := cache.postProcessBlock(block)
		require.NoError(t, err)
	}

	// Assertions for block@0

	assert.JSONEq(t, `{"from":"eosio","to":"b1","quantity":"1.0000 EOS","memo":""}`, blocks[1].TransactionTraces[0].ActionTraces[0].Action.JsonData)
}

func sharedBlocks(t *testing.T) []*pbcodec.Block {
	eosioTokenABI1 := readABI(t, "eosio.token.1.abi.json")
	eosioTestABI1 := readABI(t, "eosio.test.1.abi.json")
	eosioTestABI2 := readABI(t, "eosio.test.2.abi.json")
	eosioNekotABI1 := readABI(t, "eosio.nekot.1.abi.json")

	return []*pbcodec.Block{
		// Block #2 | Sets ABI on `eosio.token` (v1) and `eosio.test` (v1)
		testBlock(t, "00000002aa", "0000000000000000000000000000000000000000000000000000000000000000",
			trxTrace(t, actionSetABI(t, "eosio.token", 1, eosioTokenABI1)),
			trxTrace(t, actionSetABI(t, "eosio.test", 2, eosioTestABI1)),
		),

		// Block #3
		testBlock(t, "00000003aa", "00000002aa",
			trxTrace(t, action(t, "eosio.token:eosio.token:transfer", 3, eosioTokenABI1, `{"from":"eosio","to":"b1","quantity":"1.0000 EOS","memo":""}`)),
		),

		// Block #4
		testBlock(t, "00000004aa", "00000003aa",
			trxTrace(t,
				actionSetABI(t, "eosio.nekot", 4, eosioNekotABI1),
			),
		),

		// Block #5
		testBlock(t, "00000005aa", "00000004aa",
			// Set a new ABI on `eosio.test`
			trxTrace(t, actionSetABI(t, "eosio.test", 5, eosioTestABI2)),
		),
	}
}

func readABI(t *testing.T, abiFile string) (out *eos.ABI) {
	path := path.Join("testdata", "abi", abiFile)
	abiJSON, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	out = new(eos.ABI)
	err = json.Unmarshal(abiJSON, out)
	require.NoError(t, err)

	return
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
	trace := &pbcodec.TransactionTrace{}
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
			Receiver: "eosio",
		},
		Action: &pbcodec.Action{
			Account: "eosio",
			Name:    "setabi",
			RawData: rawData,
		},
	}
}
