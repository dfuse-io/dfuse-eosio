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

package resolvers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/search/v1"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/graph-gophers/graphql-go"
	rateLimiter "github.com/streamingfast/dauth/ratelimiter"
	"github.com/streamingfast/dgraphql"
	"github.com/streamingfast/dgraphql/analytics"
	commonTypes "github.com/streamingfast/dgraphql/types"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/opaque"
	"go.uber.org/zap"
)

type AccounthistClient struct {
	Account         pbaccounthist.AccountHistoryClient
	AccountContract pbaccounthist.AccountContractHistoryClient
}

// Root is the root resolver.
type Root struct {
	searchClient                  pbsearch.RouterClient
	trxsReader                    trxdb.TransactionsReader
	blocksReader                  trxdb.BlocksReader
	accountsReader                trxdb.AccountsReader
	blockmetaClient               *pbblockmeta.Client
	chainDiscriminatorClient      *pbblockmeta.ChainDiscriminatorClient
	abiCodecClient                pbabicodec.DecoderClient
	tokenmetaClient               pbtokenmeta.TokenMetaClient
	accounthistClients            *AccounthistClient
	requestRateLimiter            rateLimiter.RateLimiter
	requestRateLimiterLastLogTime time.Time
}

func NewRoot(
	searchClient pbsearch.RouterClient,
	dbReader trxdb.DBReader,
	blockMetaClient *pbblockmeta.Client,
	abiCodecClient pbabicodec.DecoderClient,
	requestRateLimiter rateLimiter.RateLimiter,
	tokenmetaClient pbtokenmeta.TokenMetaClient,
	accounthistClients *AccounthistClient,
) (interface{}, error) {
	return &Root{
		searchClient:       searchClient,
		trxsReader:         dbReader,
		blocksReader:       dbReader,
		accountsReader:     dbReader,
		tokenmetaClient:    tokenmetaClient,
		blockmetaClient:    blockMetaClient,
		abiCodecClient:     abiCodecClient,
		requestRateLimiter: requestRateLimiter,
		accounthistClients: accounthistClients,
	}, nil
}

// CAREFUL - this mirrored in the BigQuery schema - if you change this, make sure to be backwards compatible
type SearchArgs struct {
	Query            string
	SortDesc         bool
	LowBlockNum      *types.Int64
	HighBlockNum     *types.Int64
	Limit            types.Int64
	Cursor           *string
	IrreversibleOnly bool
}

func (r *Root) QuerySearchTransactionsForward(ctx context.Context, args SearchArgs) (*SearchTransactionsForwardResponse, error) {
	if err := r.RateLimit(ctx, "search"); err != nil {
		return nil, err
	}
	res, err := r.querySearchTransactionsBoth(ctx, true, args)
	if err != nil {
		return nil, err
	}

	var cursor string
	if len(res) != 0 {
		cursor = res[len(res)-1].cursor
	} else {
		// TODO: IF WE DIDN'T GET ANY RESULTS, THE CURSOR MUST USE THE
		// RESPONSE FROM streamCli to BUILD a new cursor, using the range
		// it reached
	}

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "QuerySearchTransactionsForward", "SearchArgs", args, "SearchResultsCount", len(res))
	/////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, Many Oubound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "SearchTransactionsForward",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(res)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return &SearchTransactionsForwardResponse{
		cursor:  cursor,
		Results: &res,
	}, nil
}

func (r *Root) QuerySearchTransactionsBackward(ctx context.Context, args SearchArgs) (*SearchTransactionsBackwardResponse, error) {
	if err := r.RateLimit(ctx, "search"); err != nil {
		return nil, err
	}

	res, err := r.querySearchTransactionsBoth(ctx, false, args)
	if err != nil {
		return nil, err
	}

	var backwardized []*SearchTransactionBackwardResponse
	if res != nil {
		for _, row := range res {
			backwardized = append(backwardized, &row.SearchTransactionBackwardResponse)
		}
	}

	var cursor string
	if len(backwardized) != 0 {
		cursor = backwardized[len(backwardized)-1].cursor
	} else {
		// TODO: IF WE DIDN'T GET ANY RESULTS, THE CURSOR MUST USE THE
		// RESPONSE FROM streamCli to BUILD a new cursor, using the range
		// it reached
	}

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "QuerySearchTransactionsBackward", "SearchArgs", args, "SearchResultsCount", len(res))
	/////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, Many Oubound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "SearchTransactionsBackward",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(backwardized)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return &SearchTransactionsBackwardResponse{
		cursor:  cursor,
		Results: &backwardized,
	}, nil
}

func (r *Root) querySearchTransactionsBoth(ctx context.Context, forward bool, args SearchArgs) ([]*SearchTransactionForwardResponse, error) {

	limit := args.Limit.Native()
	if limit > 1000 {
		return nil, dgraphql.Errorf(ctx, "Invalid limit for this query: max 1000")
	}

	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("executing search query",
		zap.String("query", args.Query),
		zap.Int64("low_block_num", args.LowBlockNum.Native()),
		zap.Int64("high_block_num", args.HighBlockNum.Native()),
		zap.Bool("low_block_unbounded", (args.LowBlockNum == nil)),
		zap.Bool("high_block_unbounded", (args.HighBlockNum == nil)),
		zap.String("cursor", decodeCursor(args.Cursor)),
		zap.Bool("descending", !forward))

	stream, err := r.searchClient.StreamMatches(ctx, &pbsearch.RouterRequest{
		LowBlockNum:        args.LowBlockNum.Native(),
		HighBlockNum:       args.HighBlockNum.Native(),
		LowBlockUnbounded:  args.LowBlockNum == nil,
		HighBlockUnbounded: args.HighBlockNum == nil,
		Descending:         !forward,
		Query:              args.Query,
		Cursor:             decodeCursor(args.Cursor),
		Limit:              limit,
		WithReversible:     !args.IrreversibleOnly,
		Mode:               pbsearch.RouterRequest_PAGINATED,
	})
	if err != nil {
		zlogger.Error("unable to start search transaction trace stream", zap.Error(err))
		// TODO: extract `status` from the grpc call, and transform into meaningful
		// message for the user.
		// FIXME: in `subscriptions.go`, even though we return an `error.QueryError` over here,
		// it gets repacked (err.Error()), so we lose any depth we can add here.
		return nil, dgraphql.Errorf(ctx, "backend error")
	}

	var res []*SearchTransactionForwardResponse
	for {
		if ctx.Err() != nil {
			break
		}

		match, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			zlogger.Info("error receiving message from search stream client", zap.Error(err))
			return nil, dgraphql.UnwrapError(ctx, err)
		}

		eosMatch, err := searchSpecificMatchToEOSMatch(match)
		if err != nil {
			return nil, err
		}

		out := &SearchTransactionForwardResponse{
			SearchTransactionBackwardResponse: SearchTransactionBackwardResponse{
				abiCodecClient:        r.abiCodecClient,
				cursor:                match.GetCursor(),
				trxIDPrefix:           match.TrxIdPrefix,
				irreversibleBlockNum:  uint32(match.IrrBlockNum),
				matchingActionIndexes: eosMatch.ActionIndexes,
			},
			Undo: match.Undo,
		}

		if eosMatch.Block != nil {
			out.blockID = eosMatch.Block.BlockID
			out.blockHeader = eosMatch.Block.BlockHeader
			out.trxTrace = eosMatch.Block.Trace
		} else {
			// FIXME: this should rather call a function like:
			//    dbReader.GetIrreversibleTransactionTraces(ctx, idPrefix)
			events, err := r.trxsReader.GetTransactionTraces(ctx, match.TrxIdPrefix)
			if err != nil {
				if err != context.Canceled {
					zlogger.Error("error retrieving raw transaction traces", zap.Error(err), zap.String("trx_id_prefix", match.TrxIdPrefix))
					return nil, dgraphql.Errorf(ctx, "data backend failure")
				}
				return nil, err
			}

			// This ensures that we have Irreversible Traces events, even if `kvdb` didn't load
			// it fast enough.  If we can't validate it with the `inCanonicalChain` call,
			// then we hard-fail.
			lifecycle := pbcodec.MergeTransactionEvents(events, func(id string) bool {
				// Query blockmetaClient.Ge
				resp, err := r.blockmetaClient.ChainDiscriminatorClient().InLongestChain(ctx, &pbblockmeta.InLongestChainRequest{
					BlockID: id,
				})
				if err != nil {
					return false
				}

				return resp.Irreversible
			})

			if lifecycle == nil || lifecycle.ExecutionTrace == nil {
				// INCREASE THE STATEOS saying there's inconsistencies between kvdb
				// and search, return an error to users, "Internal server error"
				// or whatever.
				return nil, dgraphql.Errorf(ctx, "cannot find requested transaction: database may not be in sync. try again later")
			}

			out.blockHeader = lifecycle.ExecutionBlockHeader
			out.blockID = lifecycle.ExecutionTrace.ProducerBlockId
			out.trxTrace = lifecycle.ExecutionTrace
		}

		zlogger.Debug("sending message", zap.String("trx_id", match.TrxIdPrefix))
		res = append(res, out)
	}

	return res, nil
}

// CAREFUL - this mirrored in the BigQuery schema - if you change this, make sure to be backwards compatible
type StreamSearchArgs struct {
	Query              string
	LowBlockNum        *types.Int64
	HighBlockNum       *types.Int64
	Cursor             *string
	Limit              types.Int64
	IrreversibleOnly   bool
	LiveMarkerInterval commonTypes.Uint32
}

func (r *Root) SubscriptionSearchTransactionsForward(ctx context.Context, args StreamSearchArgs) (<-chan *SearchTransactionForwardResponse, error) {
	if err := r.RateLimit(ctx, "search"); err != nil {
		return nil, err
	}
	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "SubscriptionSearchTransactionsForward", "StreamSearchArgs", args)
	/////////////////////////////////////////////////////////////////////////

	return r.streamSearchTracesBoth(true, ctx, args)
}

func (r *Root) SubscriptionSearchTransactionsBackward(ctx context.Context, args StreamSearchArgs) (<-chan *SearchTransactionForwardResponse, error) {
	if err := r.RateLimit(ctx, "search"); err != nil {
		return nil, err
	}
	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "SubscriptionSearchTransactionsBackward", "StreamSearchArgs", args)
	/////////////////////////////////////////////////////////////////////////

	return r.streamSearchTracesBoth(false, ctx, args)
}

type matchOrError struct {
	match *pbsearch.SearchMatch
	err   error
}

func processMatchOrError(ctx context.Context, m *matchOrError, rows [][]*pbcodec.TransactionEvent, rowMap map[string]int, abiCodecClient pbabicodec.DecoderClient) (*SearchTransactionForwardResponse, error) {
	zl := logging.Logger(ctx, zlog)
	if m.err != nil {
		return &SearchTransactionForwardResponse{
			err: dgraphql.UnwrapError(ctx, m.err),
		}, nil
	}

	match := m.match
	eosMatch, err := searchSpecificMatchToEOSMatch(match)
	if err != nil {
		return nil, err
	}

	out := &SearchTransactionForwardResponse{
		SearchTransactionBackwardResponse: SearchTransactionBackwardResponse{
			abiCodecClient:        abiCodecClient,
			cursor:                match.GetCursor(),
			trxIDPrefix:           match.TrxIdPrefix,
			irreversibleBlockNum:  uint32(match.IrrBlockNum),
			matchingActionIndexes: eosMatch.ActionIndexes,
		},
		Undo: match.Undo,
	}

	// from Payload
	if eosMatch.Block != nil {
		out.blockID = eosMatch.Block.BlockID
		out.blockHeader = eosMatch.Block.BlockHeader
		out.trxTrace = eosMatch.Block.Trace
		return out, nil
	}

	// LiveMarker
	if match.TrxIdPrefix == "" {
		return out, nil
	}

	// From Archive (kvdb lookup)
	idx, ok := rowMap[match.TrxIdPrefix]
	if !ok { // careful, kvdb can return {"prefix": nil}
		zl.Error("cannot get transaction data from match", zap.String("trx_id_prefix", match.TrxIdPrefix))
		// TODO: implement some graphs, increasing internal server errors, we want to avoid
		// this, but we don't see any data about it..
		return &SearchTransactionForwardResponse{
			err: dgraphql.Errorf(ctx, "Internal server error"),
		}, nil
	}

	events := rows[idx]
	if events == nil {
		zl.Error("cannot get transaction data from match", zap.String("trx_id_prefix", match.TrxIdPrefix))
		// TODO: implement some graphs, increasing internal server errors, we want to avoid
		// this, but we don't see any data about it..
		return &SearchTransactionForwardResponse{
			err: dgraphql.Errorf(ctx, "Internal server error"),
		}, nil
	}

	// There's only ONE case where we're fetching data from KVDB, it's
	// when we receive a match that is IRREVERSIBLE, and therefore we
	// don't have the Block and Trace data within the `StreamMatches`
	// results.  All other cases are cases of inconsistencies.  We
	// need to FAIL, and increase our error rates. We cannot send a
	// potentially forked transaction in the place of what is expected
	// to be irreversible, because kvdb is not fully in sync.

	// FIXME: here `row` will become `events`.  Want to filter those
	// events to use the `Irreversible` Trace If there's only one
	// choice, return that Only ONE thing guarantees it: it's that
	// Trace corresponds to a block in the canonical chain.  Query a
	// block meta, to know if the block_id is in canonical chain.
	lifecycle := pbcodec.MergeTransactionEvents(events, func(id string) bool { return true })

	out.blockHeader = lifecycle.ExecutionBlockHeader
	out.blockID = lifecycle.ExecutionTrace.ProducerBlockId
	out.trxTrace = lifecycle.ExecutionTrace
	return out, nil
}

func (r *Root) streamSearchTracesBoth(forward bool, ctx context.Context, args StreamSearchArgs) (<-chan *SearchTransactionForwardResponse, error) {
	zl := logging.Logger(ctx, zlog)
	c := make(chan *SearchTransactionForwardResponse) // FIXME: should be buffered at least a bit

	// TODO: if HighBlockNum is not there.. we pass HighBlockUnbounded = true
	// EVENTUALLY

	lowBlockNum := args.LowBlockNum.Native()
	if lowBlockNum < 1 {
		lowBlockNum--
	}

	highBlockNum := args.HighBlockNum.Native()
	if args.HighBlockNum != nil && highBlockNum < 1 {
		highBlockNum--
	}

	args.LowBlockNum.Native()
	streamCli, err := r.searchClient.StreamMatches(ctx, &pbsearch.RouterRequest{
		LowBlockNum:        lowBlockNum,
		HighBlockNum:       highBlockNum,
		HighBlockUnbounded: args.HighBlockNum == nil,
		Descending:         !forward,
		Query:              args.Query,
		Limit:              args.Limit.Native(),
		Cursor:             decodeCursor(args.Cursor),
		WithReversible:     !args.IrreversibleOnly,
		LiveMarkerInterval: uint64(args.LiveMarkerInterval.Native()),
	})
	if err != nil {
		zl.Error("failed StreamTransactionTraceRefs request", zap.Error(err))
		return nil, dgraphql.Errorf(ctx, "internal server error: connection to live search failed")
	}

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Subscriptions
	// WARNING : Here we only track inbound subscription init
	//////////////////////////////////////////////////////////////////////
	var billingEventDirectionSuffix string
	if forward {
		billingEventDirectionSuffix = "Forward"
	} else {
		billingEventDirectionSuffix = "Backward"
	}
	dmetering.EmitWithContext(dmetering.Event{
		Source:        "dgraphql",
		Kind:          "GraphQL Subscription",
		Method:        "SearchTransactions" + billingEventDirectionSuffix,
		RequestsCount: 1,
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	// this function converts search matchOrError into SearchTransactionForwardResponse
	// by batching the lookup to kvdb
	hammer := dhammer.NewHammer(30, 20, func(ctx context.Context, batch []interface{}) ([]interface{}, error) {
		zl.Debug("inside hammer func", zap.Int("len_batch", len(batch)))

		var prefixesToLookupInKvdb []string
		rowToIndex := map[string]int{}
		for _, v := range batch {
			m := v.(*matchOrError)
			if m.err != nil {
				zl.Info("error receiving message from search stream client", zap.Error(m.err))
				continue
			}

			// This sucks really hard. This was before a simple check if a variable was nil, now, it requires
			// a full decoding of the any message to the correct type. This is probably a performance hit here
			// to do that.
			//
			// Instead, the standard search engine should let us know if this match comes from a reversible or
			// an irreversible segment. That would make sense and would remove the need to perform some extra
			// decoding just to check if the block payload is present.
			eosMatch, err := searchSpecificMatchToEOSMatch(m.match)
			if err != nil {
				return nil, fmt.Errorf("hammer func: %w", err)
			}

			if eosMatch.Block == nil {
				prefixesToLookupInKvdb = append(prefixesToLookupInKvdb, m.match.TrxIdPrefix)
				rowToIndex[m.match.TrxIdPrefix] = len(prefixesToLookupInKvdb) - 1
			}
		}

		// FIXME: here we should decide and filter the returned `events` (`rows` for now)
		// pick one in the list (they should be only Execution traces, otherwise error out)
		// use it if its irreversible, or if its the only one we have, otherwise, you better
		// have a chain discriminator nearby to know if its in the longest chain.

		var rows [][]*pbcodec.TransactionEvent
		if len(prefixesToLookupInKvdb) != 0 {
			rows, err = r.trxsReader.GetTransactionTracesBatch(ctx, prefixesToLookupInKvdb)
			if err != nil {
				return nil, err
			}
		}

		var out []interface{}
		for _, v := range batch {
			m := v.(*matchOrError)
			resp, err := processMatchOrError(ctx, m, rows, rowToIndex, r.abiCodecClient)
			if err != nil {
				return out, err
			}
			out = append(out, resp)
		}
		return out, nil
	})

	hammer.Start(ctx)

	// search results -> Hammer
	go func() {
		defer hammer.Close()
		for {
			match, err := streamCli.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				err = fmt.Errorf("hammer search result: %w", err)
			}

			out := &matchOrError{
				match: match,
				err:   err, // we send non-EOF errors back to the client
			}

			select {
			case <-ctx.Done():
				return
			case hammer.In <- out:
			}
			if err != nil {
				return
			}
		}
	}()

	// Hammer -> GraphQL user
	var documentCount int64
	go func() {
		defer func() {
			close(c)
			if documentCount == 0 {
				//////////////////////////////////////////////////////////////////////
				// Billable event on GraphQL Subscriptions
				// WARNING : Here we only track outbound documents
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "dgraphql",
					Kind:           "GraphQL Subscription",
					Method:         "SearchTransactions" + billingEventDirectionSuffix,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-hammer.Out:
				if !ok {
					zl.Debug("done sending hammer responses, channel closed")
					if hammer.Err() != nil && hammer.Err() != context.Canceled {
						zl.Error("hammer error", zap.Error(hammer.Err()))
					}
					return
				}
				resp := v.(*SearchTransactionForwardResponse)
				select {
				case <-ctx.Done():
					return
				case c <- resp:
					if resp.trxTrace != nil { // nil means progress notification
						//////////////////////////////////////////////////////////////////////
						// Billable event on GraphQL Subscriptions
						// WARNING : Here we only track outbound documents
						//////////////////////////////////////////////////////////////////////
						documentCount++
						dmetering.EmitWithContext(dmetering.Event{
							Source:         "dgraphql",
							Kind:           "GraphQL Subscription",
							Method:         "SearchTransactions" + billingEventDirectionSuffix,
							ResponsesCount: 1,
						}, ctx)
						//////////////////////////////////////////////////////////////////////
					}
				}
			}
		}
	}()

	return c, nil
}

// Paged responses

type SearchTransactionsForwardResponse struct {
	cursor  string
	Results *[]*SearchTransactionForwardResponse
}

func (t *SearchTransactionsForwardResponse) Cursor() *string {
	if t.cursor == "" {
		return nil
	}
	c, _ := opaque.ToOpaque(t.cursor)
	return &c
}

type SearchTransactionsBackwardResponse struct {
	cursor  string
	Results *[]*SearchTransactionBackwardResponse
}

func (t *SearchTransactionsBackwardResponse) Cursor() string {
	c, _ := opaque.ToOpaque(t.cursor)
	return c
}

// Single response
type SearchTransactionForwardResponse struct {
	SearchTransactionBackwardResponse

	Undo bool
	err  error
}

func (r *SearchTransactionForwardResponse) SubscriptionError() error {
	return r.err
}

type SearchTransactionBackwardResponse struct {
	abiCodecClient pbabicodec.DecoderClient
	cursor         string
	trxIDPrefix    string
	blockHeader    *pbcodec.BlockHeader
	trxTrace       *pbcodec.TransactionTrace

	irreversibleBlockNum  uint32
	matchingActionIndexes []uint32

	//FIXME: shouldn't this be shared betweeen the two Single search responses?
	ResolverError error
	blockID       string

	// when https://github.com/graph-gophers/graphql-go/issues/314 is implemented and merged (!)
}

// TODO: extract from the traces, they're in there, just
// because.. this probably belongs outside the `traces` anyway.

func (t *SearchTransactionBackwardResponse) Cursor() string {
	c, _ := opaque.ToOpaque(t.cursor)
	return c
}

func (t *SearchTransactionBackwardResponse) IrreversibleBlockNum() commonTypes.Uint32 {
	return commonTypes.Uint32(t.irreversibleBlockNum)
}
func (t *SearchTransactionBackwardResponse) IsIrreversible() bool {
	return t.irreversibleBlockNum == eos.BlockNum(t.blockID)
}

func (t *SearchTransactionBackwardResponse) Block() *BlockHeader {
	// TODO: fetch the block header here, instead of before.. if you don't use
	// the block, we save on that bandwidth.
	// TODO: check if `t.blockHeader == nil`, then fetch. It's possible it was
	// attached already, because it was included in live search results.
	return &BlockHeader{
		blockID:  t.blockID,
		blockNum: commonTypes.Uint32(eos.BlockNum(t.blockID)),
		h:        t.blockHeader,
	}
}

func (t *SearchTransactionBackwardResponse) Trace() *TransactionTrace {
	if t.trxIDPrefix == "" {
		return nil
	}

	return newTransactionTrace(
		t.trxTrace,
		t.blockHeader,
		t.matchingActionIndexes,
		t.abiCodecClient,
	)
}

type TransactionTrace struct {
	t                     *pbcodec.TransactionTrace
	blockHeader           *pbcodec.BlockHeader
	matchingActionIndexes []uint32

	abiCodecClient pbabicodec.DecoderClient

	memoizedFlattenActionTraces *[]*ActionTrace
}

func newTransactionTrace(trace *pbcodec.TransactionTrace, blockHeader *pbcodec.BlockHeader, matchingActionIndexes []uint32, abiCodecClient pbabicodec.DecoderClient) *TransactionTrace {
	tr := &TransactionTrace{
		abiCodecClient:        abiCodecClient,
		t:                     trace,
		blockHeader:           blockHeader,
		matchingActionIndexes: matchingActionIndexes,
	}

	return tr
}

func (t *TransactionTrace) ID() string                   { return t.t.Id }
func (t *TransactionTrace) blockNum() commonTypes.Uint32 { return commonTypes.Uint32(t.t.BlockNum) }
func (t *TransactionTrace) producerBlockID() string      { return t.t.ProducerBlockId }
func (t *TransactionTrace) Block() *BlockHeader {
	return &BlockHeader{
		blockID:  t.producerBlockID(),
		blockNum: t.blockNum(),
		h:        t.blockHeader,
	}
}

func (t *TransactionTrace) Status() string {
	return t.Receipt().Status()
}

func (t *TransactionTrace) Receipt() *TransactionReceiptHeader {
	return &TransactionReceiptHeader{h: t.t.Receipt}
}
func (t *TransactionTrace) Elapsed() types.Int64   { return types.Int64(t.t.Elapsed) }
func (t *TransactionTrace) NetUsage() types.Uint64 { return types.Uint64(t.t.NetUsage) }
func (t *TransactionTrace) Scheduled() bool        { return t.t.Scheduled }

func (t *TransactionTrace) flattenActions() (out []*ActionTrace) {
	if t.memoizedFlattenActionTraces != nil {
		return *t.memoizedFlattenActionTraces
	}

	for _, actionTrace := range t.t.ActionTraces {
		out = append(out, newActionTrace(actionTrace, t, t.abiCodecClient))
	}

	for _, match := range t.matchingActionIndexes {
		if int(match) < len(out) {
			out[match].matched = true
		}
	}
	t.memoizedFlattenActionTraces = &out
	return
}

func (t *TransactionTrace) ExecutedActions() (out []*ActionTrace) {
	return t.flattenActions()
}

func (t *TransactionTrace) TopLevelActions() (out []*ActionTrace) {
	for _, act := range t.flattenActions() {
		if act.actionTrace.CreatorActionOrdinal == 0 {
			out = append(out, act)
		}
	}
	return
}
func (t *TransactionTrace) MatchingActions() (out []*ActionTrace) {
	for _, act := range t.flattenActions() {
		if act.matched {
			out = append(out, act)
		}
	}
	return
}

func (t *TransactionTrace) ExceptJSON() (*commonTypes.JSON, error) {
	if t.t.Exception != nil {
		data, err := json.Marshal(t.t.Exception)
		if err != nil {
			return nil, err
		}
		j := commonTypes.JSON(data)
		return &j, nil
	}
	return nil, nil
}

type BlockRootMerkle struct {
	m *pbcodec.BlockRootMerkle
}

func newBlockRootMerkle(merkleRoot *pbcodec.BlockRootMerkle) BlockRootMerkle {
	return BlockRootMerkle{
		m: merkleRoot,
	}
}

func (b BlockRootMerkle) NodeCount() commonTypes.Uint32 { return commonTypes.Uint32(b.m.NodeCount) }
func (b BlockRootMerkle) ActiveNodes() (out []string) {
	out = make([]string, len(b.m.ActiveNodes))
	for i, n := range b.m.ActiveNodes {
		out[i] = hex.EncodeToString(n)
	}
	return
}

type BlockHeader struct {
	blockID  string
	blockNum commonTypes.Uint32
	h        *pbcodec.BlockHeader
}

func newBlockHeader(blockID string, blockNum commonTypes.Uint32, blockHeader *pbcodec.BlockHeader) *BlockHeader {
	return &BlockHeader{
		blockID:  blockID,
		blockNum: blockNum,
		h:        blockHeader,
	}
}

func (t BlockHeader) ID() string                    { return t.blockID }
func (t BlockHeader) Num() commonTypes.Uint32       { return t.blockNum }
func (t BlockHeader) Timestamp() graphql.Time       { return toTime(t.h.Timestamp) }
func (t BlockHeader) Producer() string              { return t.h.Producer }
func (t BlockHeader) Previous() string              { return t.h.Previous }
func (t BlockHeader) TransactionMRoot() string      { return hex.EncodeToString(t.h.TransactionMroot) }
func (t BlockHeader) ActionMRoot() string           { return hex.EncodeToString(t.h.ActionMroot) }
func (t BlockHeader) Confirmed() commonTypes.Uint32 { return commonTypes.Uint32(t.h.Confirmed) }
func (t BlockHeader) ScheduleVersion() commonTypes.Uint32 {
	return commonTypes.Uint32(t.h.ScheduleVersion)
}
func (t BlockHeader) NewProducers() (out *ProducerSchedule, err error) {
	// On EOSIO 2.x, when this field is `nil`, it means the actual producer schedule change is in the block header extensions
	if t.h.NewProducersV1 == nil {
		extension, err := t.findProducerScheduleChangeExtension()
		if err != nil {
			return nil, err
		}

		if extension != nil {
			return &ProducerSchedule{s: codec.ProducerAuthorityScheduleToDEOS(&extension.ProducerAuthoritySchedule)}, nil
		}

		// We really don't have any schedule change for this
		return nil, nil
	}

	return &ProducerSchedule{s: upgradeToProducerAuthoritySchedule(t.h.NewProducersV1)}, nil
}

func upgradeToProducerAuthoritySchedule(old *pbcodec.ProducerSchedule) *pbcodec.ProducerAuthoritySchedule {
	producers := make([]*pbcodec.ProducerAuthority, len(old.Producers))
	for i, oldProducer := range old.Producers {
		producers[i] = &pbcodec.ProducerAuthority{
			AccountName: oldProducer.AccountName,
			BlockSigningAuthority: &pbcodec.BlockSigningAuthority{
				Variant: &pbcodec.BlockSigningAuthority_V0{
					V0: &pbcodec.BlockSigningAuthorityV0{
						Threshold: 1,
						Keys: []*pbcodec.KeyWeight{
							{Weight: 1, PublicKey: oldProducer.BlockSigningKey},
						},
					},
				},
			},
		}
	}

	return &pbcodec.ProducerAuthoritySchedule{
		Version:   old.Version,
		Producers: producers,
	}
}

func (t BlockHeader) findProducerScheduleChangeExtension() (*eos.ProducerScheduleChangeExtension, error) {
	for _, e := range t.h.HeaderExtensions {
		if e.Type == uint32(eos.EOS_ProducerScheduleChangeExtension) {
			extension := &eos.ProducerScheduleChangeExtension{}
			err := eos.UnmarshalBinary(e.Data, extension)
			if err != nil {
				return nil, fmt.Errorf("unable to decode binary extension correctly: %s", err)
			}

			return extension, nil
		}
	}

	return nil, nil
}

type ProducerSchedule struct {
	s *pbcodec.ProducerAuthoritySchedule
}

func (s *ProducerSchedule) Version() commonTypes.Uint32 { return commonTypes.Uint32(s.s.Version) }
func (s *ProducerSchedule) Producers() (out []*ProducerKey) {
	for _, k := range s.s.Producers {
		out = append(out, &ProducerKey{k: k})
	}
	return
}

type ProducerKey struct {
	k *pbcodec.ProducerAuthority
}

func (k *ProducerKey) ProducerName() string { return k.k.AccountName }
func (k *ProducerKey) BlockSigningKey() string {
	return extractFirstPublicKeyFromAuthority(k.k.BlockSigningAuthority)
}

func extractFirstPublicKeyFromAuthority(in *pbcodec.BlockSigningAuthority) string {
	if in.GetV0() == nil {
		panic(fmt.Errorf("only knowns how to deal with BlockSigningAuthority_V0 type, got %t", in.Variant))
	}

	keys := in.GetV0().GetKeys()
	if len(keys) <= 0 {
		return ""
	}

	return keys[0].PublicKey
}

type TransactionReceiptHeader struct {
	h *pbcodec.TransactionReceiptHeader
}

func (h *TransactionReceiptHeader) Status() string {
	return strings.ToUpper(codec.TransactionStatusToEOS(h.h.Status).String())
}
func (h *TransactionReceiptHeader) CPUUsageMicroSeconds() commonTypes.Uint32 {
	return commonTypes.Uint32(h.h.CpuUsageMicroSeconds)
}
func (h *TransactionReceiptHeader) NetUsageWords() commonTypes.Uint32 {
	return commonTypes.Uint32(h.h.NetUsageWords)
}

type ActionTrace struct {
	actionTrace *pbcodec.ActionTrace
	trxTrace    *TransactionTrace
	matched     bool

	abiCodecClient pbabicodec.DecoderClient
}

func newActionTrace(actionTrace *pbcodec.ActionTrace, trxTrace *TransactionTrace, abiCodecClient pbabicodec.DecoderClient) (out *ActionTrace) {
	return &ActionTrace{
		abiCodecClient: abiCodecClient,
		actionTrace:    actionTrace,
		trxTrace:       trxTrace,
	}
}

func (t *ActionTrace) Seq() types.Uint64 {
	if t.actionTrace.Receipt == nil {
		return types.Uint64(0)
	}

	return types.Uint64(t.actionTrace.Receipt.GlobalSequence)
}

func (t *ActionTrace) ExecutionIndex() commonTypes.Uint32 {
	return commonTypes.Uint32(t.actionTrace.ExecutionIndex)
}
func (t *ActionTrace) Receipt() *ActionReceipt {
	if t.actionTrace.Receipt == nil {
		return nil
	}

	return &ActionReceipt{r: t.actionTrace.Receipt}
}
func (t *ActionTrace) Receiver() string                        { return t.actionTrace.Receiver }
func (t *ActionTrace) Action() *Action                         { return &Action{a: t.actionTrace.Action} }
func (t *ActionTrace) Account() string                         { return t.actionTrace.Account() }
func (t *ActionTrace) Name() string                            { return t.actionTrace.Name() }
func (t *ActionTrace) Authorization() (out []*PermissionLevel) { return t.Action().Authorization() }
func (t *ActionTrace) Data() *commonTypes.JSON                 { return t.Action().Data() }
func (t *ActionTrace) JSON() *commonTypes.JSON                 { return t.Action().JSON() }
func (t *ActionTrace) HexData() string                         { return t.Action().HexData() }
func (t *ActionTrace) TrxID() string                           { return t.actionTrace.TransactionId }
func (t *ActionTrace) BlockNum() types.Uint64                  { return types.Uint64(t.actionTrace.BlockNum) }
func (t *ActionTrace) BlockID() string                         { return t.actionTrace.ProducerBlockId }
func (t *ActionTrace) BlockTime() graphql.Time                 { return toTime(t.actionTrace.BlockTime) }

func (t *ActionTrace) Console() string       { return t.actionTrace.Console }
func (t *ActionTrace) ContextFree() bool     { return t.actionTrace.ContextFree }
func (t *ActionTrace) IsMatchingQuery() bool { return t.matched }
func (t *ActionTrace) Elapsed() types.Int64  { return types.Int64(t.actionTrace.Elapsed) }
func (t *ActionTrace) ExceptJSON() (*commonTypes.JSON, error) {
	if t.actionTrace.Exception != nil {
		data, err := json.Marshal(t.actionTrace.Exception)
		if err != nil {
			return nil, err
		}
		j := commonTypes.JSON(data)
		return &j, nil
	}

	return nil, nil
}

func (t *ActionTrace) IsNotify() bool {
	receiver := t.actionTrace.Receiver

	return receiver != "" && receiver != t.actionTrace.Account()
}

func (t *ActionTrace) CreatedActions(args struct{ Sort string }) (out []*ActionTrace) {
	for _, act := range t.trxTrace.flattenActions() {
		if act.actionTrace.CreatorActionOrdinal == t.actionTrace.ActionOrdinal {
			out = append(out, act)
		}
	}

	if args.Sort == "CREATION" {
		sort.Slice(out, func(i, j int) bool {
			return out[i].actionTrace.CreatorActionOrdinal < out[j].actionTrace.CreatorActionOrdinal
		})
	} /* otherwise, sorted by EXECUTION already */

	return
}

func (t *ActionTrace) CreatorAction() *ActionTrace {
	for _, act := range t.trxTrace.flattenActions() {
		if act.actionTrace.ActionOrdinal == t.actionTrace.CreatorActionOrdinal {
			return act
		}
	}
	return nil
}

// Not exposed right now. Please don't expose it.. because this parenthood is confusing and risks
// creating
func (t *ActionTrace) InlineTraces() (out []*ActionTrace) {
	for _, act := range t.trxTrace.flattenActions() {
		// Link to the closest unnotified ancestor
		if act.actionTrace.ClosestUnnotifiedAncestorActionOrdinal == t.actionTrace.ActionOrdinal {
			out = append(out, act)
		}
	}
	return
}

func (t *ActionTrace) ClosestUnnotifiedAncestorAction() (out *ActionTrace) {
	for _, act := range t.trxTrace.flattenActions() {
		if t.actionTrace.ClosestUnnotifiedAncestorActionOrdinal == act.actionTrace.ActionOrdinal {
			return act
		}
	}
	return nil
}

func (t *ActionTrace) RAMOps() (out []*RAMOp) {
	ramOps := t.trxTrace.t.RAMOpsForAction(t.actionTrace.ExecutionIndex)
	out = make([]*RAMOp, len(ramOps))

	for i, ramOp := range ramOps {
		out[i] = &RAMOp{op: ramOp}
	}

	return
}

type DBOpsArgs struct {
	Table *string
	Code  *string
}

func (t *ActionTrace) DBOps(args DBOpsArgs) (out []*DBOp) {
	for _, dbOp := range t.trxTrace.t.DBOpsForAction(t.actionTrace.ExecutionIndex) {
		if args.Table != nil && *args.Table != "" && dbOp.TableName != *args.Table {
			continue
		}

		if args.Code != nil && *args.Code != "" && *args.Code != dbOp.Code {
			continue
		}

		out = append(out, newDBOp(dbOp, t.actionTrace.BlockNum, t.abiCodecClient))
	}

	return
}
func (t *ActionTrace) DTrxOps() (out []*DTrxOp) {
	dtrxOps := t.trxTrace.t.DtrxOpsForAction(t.actionTrace.ExecutionIndex)
	out = make([]*DTrxOp, len(dtrxOps))

	for i, dtrxOp := range dtrxOps {
		out[i] = &DTrxOp{op: dtrxOp}
	}

	return
}
func (t *ActionTrace) TableOps() (out []*TableOp) {
	tableOps := t.trxTrace.t.TableOpsForAction(t.actionTrace.ExecutionIndex)
	out = make([]*TableOp, len(tableOps))

	for i, dtrxOp := range tableOps {
		out[i] = &TableOp{op: dtrxOp}
	}

	return
}

type ActionReceipt struct {
	r *pbcodec.ActionReceipt
}

func (r *ActionReceipt) Receiver() string             { return r.r.Receiver }
func (r *ActionReceipt) Digest() string               { return r.r.Digest }
func (r *ActionReceipt) GlobalSequence() types.Uint64 { return types.Uint64(r.r.GlobalSequence) }
func (r *ActionReceipt) RecvSequence() types.Uint64   { return types.Uint64(r.r.RecvSequence) }
func (r *ActionReceipt) CodeSequence() types.Uint64   { return types.Uint64(r.r.CodeSequence) }
func (r *ActionReceipt) ABISequence() types.Uint64    { return types.Uint64(r.r.AbiSequence) }

// Not exported until asked for.. we can do the full unpacking
// then.. and not ship a half-baked struct.
func (r *ActionReceipt) authSequence() commonTypes.JSON { return nil }

type Transaction struct {
	t *pbcodec.SignedTransaction
}

func (t *Transaction) Expiration() graphql.Time { return toTime(t.t.Transaction.Header.Expiration) }
func (t *Transaction) RefBlockNum() commonTypes.Uint32 {
	return commonTypes.Uint32(t.t.Transaction.Header.RefBlockNum)
}
func (t *Transaction) RefBlockPrefix() commonTypes.Uint32 {
	return commonTypes.Uint32(t.t.Transaction.Header.RefBlockPrefix)
}
func (t *Transaction) MaxNetUsageWords() commonTypes.Uint32 {
	return commonTypes.Uint32(t.t.Transaction.Header.MaxNetUsageWords)
}
func (t *Transaction) MaxCPUUsageMS() commonTypes.Uint32 {
	return commonTypes.Uint32(t.t.Transaction.Header.MaxCpuUsageMs)
}
func (t *Transaction) DelaySec() commonTypes.Uint32 {
	return commonTypes.Uint32(t.t.Transaction.Header.DelaySec)
}
func (t *Transaction) ContextFreeActions() (out []*Action) {
	for _, cfa := range t.t.Transaction.ContextFreeActions {
		out = append(out, &Action{a: cfa})
	}
	return
}
func (t *Transaction) Actions() (out []*Action) {
	for _, a := range t.t.Transaction.Actions {
		out = append(out, &Action{a: a})
	}
	return
}

type Action struct {
	a *pbcodec.Action
}

func (a *Action) Account() string { return a.a.Account }
func (a *Action) Name() string    { return a.a.Name }
func (a *Action) Authorization() (out []*PermissionLevel) {
	out = make([]*PermissionLevel, len(a.a.Authorization))
	for i, a := range a.a.Authorization {
		out[i] = &PermissionLevel{pl: a}
	}
	return
}

func fixUtf(r rune) rune {
	if r == utf8.RuneError {
		return '�'
	}
	return r
}

func (a *Action) Data() *commonTypes.JSON {
	return a.JSON()
}

func (a *Action) JSON() *commonTypes.JSON {
	if a.a.JsonData == "" {
		return nil
	}

	json := commonTypes.JSON(strings.Map(fixUtf, a.a.JsonData))
	return &json
}
func (a *Action) HexData() string { return hex.EncodeToString(a.a.RawData) }

type PermissionLevel struct {
	pl *pbcodec.PermissionLevel
}

func (t *PermissionLevel) Actor() string      { return t.pl.Actor }
func (t *PermissionLevel) Permission() string { return t.pl.Permission }

func searchSpecificMatchToEOSMatch(match *pbsearch.SearchMatch) (*pbsearcheos.Match, error) {
	var eosMatchAny ptypes.DynamicAny
	err := ptypes.UnmarshalAny(match.GetChainSpecific(), &eosMatchAny)
	if err != nil {
		return nil, err
	}

	return eosMatchAny.Message.(*pbsearcheos.Match), nil
}
