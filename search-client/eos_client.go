package searchclient

import (
	"context"
	"fmt"
	"io"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/search/eos/v1"
	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	searchclient "github.com/dfuse-io/search-client"
	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type EOSClient struct {
	*searchclient.CommonClient

	dbReader eosdb.DBReader
}

type EOSStreamMatchesClient interface {
	Recv() (*EOSSearchMatch, error)
}

type EOSSearchMatch struct {
	*pbsearch.SearchMatch

	BlockID          string
	BlockHeader      *pbeos.BlockHeader
	TransactionTrace *pbeos.TransactionTrace
	MatchingActions  []*pbeos.ActionTrace
}

func NewEOSClient(cc *grpc.ClientConn, dbReader eosdb.DBReader) *EOSClient {
	return &EOSClient{searchclient.NewCommonClient(cc), dbReader}
}

func (e *EOSClient) StreamMatches(callerCtx context.Context, req *pbsearch.RouterRequest) (EOSStreamMatchesClient, error) {
	hammer := dhammer.NewHammer(30, 20, e.hammerBatchProcessor)
	hammer.Start(callerCtx)

	go e.StreamSearchToHammer(callerCtx, hammer, req)

	esm := &eosStreamMatches{
		ctx:     callerCtx,
		errors:  make(chan error),
		matches: make(chan *EOSSearchMatch),
	}

	go e.HammerToConsumer(callerCtx, hammer, esm.onItem, esm.onError)

	return esm, nil
}

func (e *EOSClient) hammerBatchProcessor(ctx context.Context, items []interface{}) (out []interface{}, err error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("processing hammer batch", zap.Int("item_count", len(items)))

	prefixes, prefixToIndex := searchclient.GatherTransactionPrefixesToFetch(items, isIrreversibleEOSMatch)

	var rows [][]*pbeos.TransactionEvent
	if len(prefixes) > 0 {
		zlogger.Debug("performing retrieval of transaction traces", zap.Int("prefix_count", len(prefixes)))
		rows, err = e.dbReader.GetTransactionTracesBatch(ctx, prefixes)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch transaction traces batch: %w", err)
		}
	}

	for _, v := range items {
		m := v.(*matchOrError)
		resp, err := processEOSHammerItem(ctx, m, rows, prefixToIndex)
		if err != nil {
			return out, err
		}

		out = append(out, resp)
	}

	return out, nil
}

func processEOSHammerItem(ctx context.Context, m *matchOrError, rows [][]*pbeos.TransactionEvent, rowMap map[string]int) (*EOSSearchMatch, error) {
	if m.err != nil {
		return nil, m.err
	}

	trxIDPrefix := m.match.TrxIdPrefix

	var eosMatchAny ptypes.DynamicAny
	err := ptypes.UnmarshalAny(m.match.GetChainSpecific(), &eosMatchAny)
	if err != nil {
		return nil, err
	}

	eosMatch := eosMatchAny.Message.(*pbsearcheos.Match)

	var blockID string
	var blockHeader *pbeos.BlockHeader
	var trace *pbeos.TransactionTrace

	if eosMatch.Block != nil {
		blockID = eosMatch.Block.BlockID
		blockHeader = eosMatch.Block.BlockHeader
		trace = eosMatch.Block.Trace
	} else {
		idx, ok := rowMap[trxIDPrefix]
		if !ok {
			return nil, fmt.Errorf("no transaction events row pointer for trx prefix %q", trxIDPrefix)
		}

		events := rows[idx]
		if events == nil {
			return nil, fmt.Errorf("transaction events for trx prefix %q are missing", trxIDPrefix)
		}

		// If we are here, it must be because the result was irreversible (otherwise,
		// the `eosMatch.Block != nil` would have been `true`). Hence, it's ok to not have
		// a chain discriminator here.
		lifecycle := pbeos.MergeTransactionEvents(events, func(id string) bool { return true })
		if lifecycle.ExecutionTrace == nil {
			return nil, fmt.Errorf("unable to merge transaction events correctly")
		}

		blockID = lifecycle.ExecutionTrace.ProducerBlockId
		blockHeader = lifecycle.ExecutionBlockHeader
		trace = lifecycle.ExecutionTrace
	}

	var matchingActions []*pbeos.ActionTrace
	if trace != nil {
		matchingActions = make([]*pbeos.ActionTrace, len(eosMatch.ActionIndexes))
		for i, callIndex := range eosMatch.ActionIndexes {
			matchingActions[i] = trace.ActionTraces[callIndex]
		}
	}

	return &EOSSearchMatch{
		SearchMatch:      m.match,
		BlockID:          blockID,
		BlockHeader:      blockHeader,
		TransactionTrace: trace,
		MatchingActions:  matchingActions,
	}, nil
}

func isIrreversibleEOSMatch(match *pbsearch.SearchMatch) bool {
	// This sucks really hard. This was before a simple check if a variable was nil, now, it requires
	// a full decoding of the any message to the correct type. This is probably a performance hit here
	// to do that.
	//
	// Instead, the standard search engine should let us know if this match comes from a reversible or
	// an irreversible segment. That would make sense and would remove the need to perform some extra
	// decoding just to check if the block payload is present.
	var eosMatchAny ptypes.DynamicAny
	err := ptypes.UnmarshalAny(match.GetChainSpecific(), &eosMatchAny)
	if err != nil {
		panic("this should be an EOS match object, it should already been validated at this point, this should not happen")
	}

	return eosMatchAny.Message.(*pbsearcheos.Match).Block == nil
}

type eosStreamMatches struct {
	ctx     context.Context
	errors  chan error
	matches chan *EOSSearchMatch
}

func (e *eosStreamMatches) Recv() (*EOSSearchMatch, error) {
	select {
	case <-e.ctx.Done():
		if err := e.ctx.Err(); err != nil {
			return nil, err
		}

		return nil, io.EOF
	case err := <-e.errors:
		return nil, err
	case match := <-e.matches:
		return match, nil
	}
}

func (e *eosStreamMatches) onError(err error) {
	select {
	case <-e.ctx.Done():
		return
	case e.errors <- err:
	}
}

func (e *eosStreamMatches) onItem(v interface{}) {
	select {
	case <-e.ctx.Done():
		return
	case e.matches <- v.(*EOSSearchMatch):
	}
}
