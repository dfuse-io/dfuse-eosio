package ct

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	"github.com/dfuse-io/logging"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"github.com/golang/protobuf/ptypes"
	pbts "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mitchellh/go-testing-interface"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type Hash string
type BlockTime string
type BlockTimestamp time.Time

var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/codec/testing", &zlog)
}

func (h Hash) Bytes(t testing.T) []byte {
	bytes, err := hex.DecodeString(string(h))
	require.NoErrorf(t, err, "hash %q is to valid hex: %w", h)

	return bytes
}

type FilteredBlock struct {
	Include         string
	Exclude         string
	UnfilteredStats Counts
	FilteredStats   Counts
}

type Counts struct {
	TrxTraceCount      int
	ActTraceInputCount int
	ActTraceTotalCount int
}

func Block(t testing.T, blkID string, components ...interface{}) *pbcodec.Block {
	ref := bstream.NewBlockRefFromID(blkID)

	pbblock := &pbcodec.Block{
		Id:     blkID,
		Number: uint32(ref.Num()),
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbcodec.BlockHeader{
		Previous:  fmt.Sprintf("%08x%s", pbblock.Number-1, blkID[8:]),
		Producer:  "tester",
		Timestamp: blockTimestamp,
	}

	for _, component := range components {
		switch v := component.(type) {
		case BlockTime:
			blockTime, err := time.Parse(time.RFC3339, string(v))
			require.NoError(t, err)

			pbblock.Header.Timestamp, err = ptypes.TimestampProto(blockTime)
			require.NoError(t, err)
		case BlockTimestamp:
			pbblock.Header.Timestamp, err = ptypes.TimestampProto(time.Time(v))
			require.NoError(t, err)

		case *pbcodec.TransactionTrace:
			v.BlockNum = pbblock.Num()
			v.BlockTime, err = ptypes.TimestampProto(pbblock.MustTime())
			require.NoError(t, err)

			pbblock.UnfilteredTransactionTraces = append(pbblock.UnfilteredTransactionTraces, v)
		case *pbcodec.TrxOp:
			pbblock.UnfilteredImplicitTransactionOps = append(pbblock.UnfilteredImplicitTransactionOps, v)
		case FilteredBlock:
			// Performed at the very end
		default:
			failInvalidComponent(t, "block", component)
		}
	}

	pbblock.MigrateV0ToV1()

	// Need to go at the end to ensure we catch all transaction traces
	if component := findComponent(components, func(component interface{}) bool { _, ok := component.(FilteredBlock); return ok }); component != nil {
		filtered := component.(FilteredBlock)

		pbblock.FilteringApplied = true
		pbblock.FilteringIncludeFilterExpr = filtered.Include
		pbblock.FilteringExcludeFilterExpr = filtered.Exclude
		pbblock.FilteredTransactionTraces = pbblock.UnfilteredTransactionTraces
		pbblock.UnfilteredTransactionTraces = nil
		pbblock.MigrateV0ToV1()

		pbblock.UnfilteredTransactionTraceCount = uint32(filtered.UnfilteredStats.TrxTraceCount)
		pbblock.UnfilteredExecutedInputActionCount = uint32(filtered.UnfilteredStats.ActTraceInputCount)
		pbblock.UnfilteredExecutedTotalActionCount = uint32(filtered.UnfilteredStats.ActTraceTotalCount)

		pbblock.FilteredTransactionTraceCount = uint32(filtered.FilteredStats.TrxTraceCount)
		pbblock.FilteredExecutedInputActionCount = uint32(filtered.FilteredStats.ActTraceInputCount)
		pbblock.FilteredExecutedTotalActionCount = uint32(filtered.FilteredStats.ActTraceTotalCount)
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

func ToBstreamBlock(t testing.T, block *pbcodec.Block) *bstream.Block {
	blk, err := codec.BlockFromProto(block)
	require.NoError(t, err)

	return blk
}

func ToPbbstreamBlock(t testing.T, block *pbcodec.Block) *pbbstream.Block {
	blk, err := ToBstreamBlock(t, block).ToProto()
	require.NoError(t, err)

	return blk
}

type TrxID string

func TrxTrace(t testing.T, components ...interface{}) *pbcodec.TransactionTrace {
	trace := &pbcodec.TransactionTrace{
		Receipt: &pbcodec.TransactionReceiptHeader{
			Status: pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED,
		},
	}

	for _, element := range components {
		switch v := element.(type) {
		case TrxID:
			trace.Id = string(v)
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
		default:
			failInvalidComponent(t, "transaction trace", element)
		}
	}

	return trace
}

func SignedTrx(t testing.T, elements ...interface{}) *pbcodec.SignedTransaction {
	signedTrx := &pbcodec.SignedTransaction{}
	signedTrx.Transaction = Trx(t, elements...)

	return signedTrx
}

type ContextFreeAction *pbcodec.Action

func Trx(t testing.T, elements ...interface{}) *pbcodec.Transaction {
	trx := &pbcodec.Transaction{}

	for _, element := range elements {
		switch v := element.(type) {
		case *pbcodec.Action:
			trx.Actions = append(trx.Actions, v)
		case ContextFreeAction:
			trx.ContextFreeActions = append(trx.ContextFreeActions, (*pbcodec.Action)(v))
		default:
			failInvalidComponent(t, "transaction", element)
		}
	}

	return trx
}

type ActionData string
type actionMatched bool

var ActionMatched = actionMatched(true)

type ExecutionIndex uint32
type GlobalSequence uint64

func ActionTrace(t testing.T, tripletName string, components ...interface{}) *pbcodec.ActionTrace {
	parts := strings.Split(tripletName, ":")
	receiver := parts[0]

	var account, actionName string
	if len(parts) == 2 {
		account = receiver
		actionName = parts[1]
	} else {
		// We assume 3 for now
		account = parts[1]
		actionName = parts[2]
	}

	actTrace := &pbcodec.ActionTrace{
		Receiver: receiver,
		Receipt: &pbcodec.ActionReceipt{
			Receiver: receiver,
		},
		Action: Action(t, account+":"+actionName, components...),
	}

	return transformActionTrace(t, actTrace, components)
}

func ActionTraceFail(t testing.T, tripletName string, components ...interface{}) *pbcodec.ActionTrace {
	components = append(components, GlobalSequence(0))
	out := ActionTrace(t, tripletName, components...)
	out.Receipt = nil

	return out
}

func ActionTraceSetABI(t testing.T, account string, abi *eos.ABI, components ...interface{}) *pbcodec.ActionTrace {
	abiData, err := eos.MarshalBinary(abi)
	require.NoError(t, err)

	setABI := &system.SetABI{Account: eos.AccountName(account), ABI: eos.HexBytes(abiData)}
	rawData, err := eos.MarshalBinary(setABI)
	require.NoError(t, err)

	actTrace := &pbcodec.ActionTrace{
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

	return transformActionTrace(t, actTrace, components)
}

func transformActionTrace(t testing.T, actTrace *pbcodec.ActionTrace, components []interface{}) *pbcodec.ActionTrace {
	ignoreIfActionComponent := ignoreComponent(func(component interface{}) bool {
		switch component.(type) {
		case ActionData:
		default:
			return false
		}

		// Ignore all
		return true
	})

	for _, component := range components {
		switch v := component.(type) {
		case ExecutionIndex:
			actTrace.ExecutionIndex = uint32(v)
		case GlobalSequence:
			actTrace.Receipt.GlobalSequence = uint64(v)
		case actionMatched:
			actTrace.FilteringMatched = bool(v)
		default:
			failInvalidComponent(t, "action trace", component, ignoreIfActionComponent)
		}
	}

	return actTrace
}

func CFAAction(t testing.T, pairName string, abi *eos.ABI, data string) ContextFreeAction {
	return ContextFreeAction(Action(t, pairName, abi, data))
}

func Action(t testing.T, pairName string, components ...interface{}) *pbcodec.Action {
	parts := strings.Split(pairName, ":")
	account := parts[0]
	actionName := parts[1]

	abi := findABIComponent(components)
	data := findActionData(components)

	var rawData []byte
	if abi != nil && data != "" {
		var err error
		rawData, err = abi.EncodeAction(eos.ActionName(actionName), []byte(data))
		require.NoError(t, err)
	}

	return &pbcodec.Action{
		Account:  account,
		Name:     actionName,
		RawData:  rawData,
		JsonData: data,
	}
}

func findABIComponent(components []interface{}) *eos.ABI {
	if component := findComponent(components, func(component interface{}) bool { _, ok := component.(*eos.ABI); return ok }); component != nil {
		return component.(*eos.ABI)
	}

	return nil
}

func findActionData(components []interface{}) string {
	if component := findComponent(components, func(component interface{}) bool { _, ok := component.(ActionData); return ok }); component != nil {
		return string(component.(ActionData))
	}

	return ""
}

func findComponent(components []interface{}, doesMatch func(component interface{}) bool) interface{} {
	for _, component := range components {
		if doesMatch(component) {
			return component
		}
	}

	return nil
}

func hasComponent(components []interface{}, doesMatch func(component interface{}) bool) bool {
	return findComponent(components, doesMatch) != nil
}

func TrxOp(t testing.T, signedTrx *pbcodec.SignedTransaction) *pbcodec.TrxOp {
	op := &pbcodec.TrxOp{
		Transaction: signedTrx,
	}

	return op
}

type DtrxOpActionIndex uint32
type DtrxOpPayer string

func DtrxOp(t testing.T, operation string, trxID string, components ...interface{}) *pbcodec.DTrxOp {
	opName := pbcodec.DTrxOp_Operation_value["OPERATION_"+strings.ToUpper(operation)]

	op := &pbcodec.DTrxOp{
		Operation:     pbcodec.DTrxOp_Operation(opName),
		TransactionId: trxID,
	}

	for _, component := range components {
		switch v := component.(type) {
		case DtrxOpActionIndex:
			op.ActionIndex = uint32(v)
		case DtrxOpPayer:
			op.Payer = string(v)
		case *pbcodec.SignedTransaction:
			op.Transaction = v
		default:
			failInvalidComponent(t, "dtrx op", component)
		}
	}

	return op
}

func ToTimestamp(t time.Time) *pbts.Timestamp {
	el, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}

	return el
}

type ignoreComponent func(v interface{}) bool

func failInvalidComponent(t testing.T, tag string, component interface{}, options ...interface{}) {
	shouldIgnore := ignoreComponent(func(v interface{}) bool { return false })
	for _, option := range options {
		switch v := option.(type) {
		case ignoreComponent:
			shouldIgnore = v
		}
	}

	if shouldIgnore(component) {
		return
	}

	require.FailNowf(t, "invalid component", "Invalid %s component of type %T", tag, component)
}

func logInvalidComponent(tag string, component interface{}) {
	zlog.Info(fmt.Sprintf("invalid %s component of type %T", tag, component))
}
