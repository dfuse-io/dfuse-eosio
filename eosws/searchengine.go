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

package eosws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/search/v1"
	"github.com/dfuse-io/dtracing"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	"github.com/dfuse-io/logging"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/opaque"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type SearchEngine struct {
	trxdb        DB
	searchClient pbsearch.RouterClient
}

func NewSearchEngine(db DB, searchClient pbsearch.RouterClient) *SearchEngine {
	return &SearchEngine{
		trxdb:        db,
		searchClient: searchClient,
	}
}

type SearchQuery struct {
	Query          string `json:"query"`
	StartBlock     uint32 `json:"start_block"`
	BlockCount     uint32 `json:"block_count"`
	SortDescending bool   `json:"sort_desc"`
	Limit          uint64 `json:"limit"`
	Cursor         string `json:"cursor"`
	WithReversible bool   `json:"with_reversible"`
	Format         string `json:"format"`
}

type searchClientResponse struct {
	Cursor       string                     `json:"cursor"`
	Transactions []*searchClientTransaction `json:"transactions"`
}

type searchFormattedClientResponse struct {
	Cursor       string            `json:"cursor"`
	Transactions []json.RawMessage `json:"transactions"`
}

type searchClientTransaction struct {
	Lifecycle *v1.TransactionLifecycle `json:"lifecycle"`
	Actions   []uint32                 `json:"action_idx"`
}

func (s *SearchEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	errors := validateSearchTransactionsRequest(r)
	if len(errors) > 0 {
		WriteError(w, r, derr.RequestValidationError(ctx, errors))
		return
	}

	zlogger := logging.Logger(ctx, zlog)

	searchQuery := extractSearchQueryFromRequest(r)
	zlogger.Debug("extracted search query", zap.Any("query", searchQuery))

	matches, rangeCompleted, err := s.DoRequest(ctx, toSearchNativeRequest(ctx, searchQuery))
	if err != nil {
		WriteError(w, r, derr.Wrap(err, "server error: unable to initiate to search"))
		return
	}

	clientResponse, err := s.fillSearchClientResponse(ctx, zlogger, matches, rangeCompleted)
	if err != nil {
		WriteError(w, r, derr.Wrap(err, "unable to fill search client response"))
		return
	}

	writeResponse(ctx, w, r, false, clientResponse)

	//////////////////////////////////////////////////////////////////////
	// Billable event on REST API endpoint
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "eosws",
		Kind:           "REST API",
		Method:         "/v0/search/transactions",
		RequestsCount:  1,
		ResponsesCount: int64(len(matches)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////
}

func (s *SearchEngine) DoRequest(ctx context.Context, q *pbsearch.RouterRequest) (matches []*pbsearch.SearchMatch, rangeCompleted bool, err error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("running search native request", zap.Any("native_search_request", q))

	stream, err := s.searchClient.StreamMatches(ctx, q)
	if err != nil {
		return nil, false, err
	}

	for {
		searchMatch, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, false, err
		}

		matches = append(matches, searchMatch)
	}

	trail := stream.Trailer()
	rangeCompleted, _ = strconv.ParseBool(trailerGetOne(trail, "range-completed")) // ignoring error -> will be false

	return
}

func trailerGetOne(in metadata.MD, key string) string {
	vals := in.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func toSearchNativeRequest(ctx context.Context, from *SearchQuery) (out *pbsearch.RouterRequest) {
	return &pbsearch.RouterRequest{
		UseLegacyBoundaries: true,
		// FIXME: Does this have an impact? I think yes because within this code path, I assume `-100` is possible ...!
		StartBlock:     uint64(from.StartBlock),
		BlockCount:     uint64(from.BlockCount),
		Query:          from.Query,
		Limit:          int64(from.Limit),
		Cursor:         from.Cursor,
		WithReversible: from.WithReversible,
		Descending:     from.SortDescending,
	}
}

func writeResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, formatted bool, response interface{}) {
	ctx, span := dtracing.StartSpan(ctx, "search write response", "formatted", formatted)
	defer span.End()

	WriteJSON(w, r, response)
}

func (s *SearchEngine) fillSearchClientResponse(
	ctx context.Context,
	zlogger *zap.Logger,
	matches []*pbsearch.SearchMatch,
	rangeCompleted bool,
) (*searchClientResponse, error) {

	actions := map[string][]uint32{}
	trxIDS := make([]string, len(matches))

	var lastCursor string
	for i, match := range matches {
		var eosMatchAny ptypes.DynamicAny
		err := ptypes.UnmarshalAny(match.GetChainSpecific(), &eosMatchAny)
		if err != nil {
			return nil, err
		}

		eosMatch := eosMatchAny.Message.(*pbsearcheos.Match)

		actions[match.TrxIdPrefix] = eosMatch.ActionIndexes
		trxIDS[i] = match.TrxIdPrefix
		lastCursor = match.Cursor
	}

	zlogger.Debug("fetching transactions from bigtable", zap.Int("count", len(trxIDS)))
	lifecycles, err := s.trxdb.GetTransactions(ctx, trxIDS)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get transactions from database")
	}

	zlogger.Debug("got transaction lifecycles", zap.Int("count", len(lifecycles)))
	out := &searchClientResponse{}
	for _, lifecycle := range lifecycles {
		truncatedTrxID := lifecycle.Id[:32] // CONSTANT number of hex chars, pourri..

		lc, err := mdl.ToV1TransactionLifecycle(lifecycle)
		if err != nil {
			return nil, fmt.Errorf("fill search client response: %w", err)
		}

		out.Transactions = append(out.Transactions, &searchClientTransaction{
			Lifecycle: lc,
			Actions:   actions[truncatedTrxID],
		})
	}
	if !rangeCompleted && lastCursor != "" { // no cursor if search completed over the range
		out.Cursor, _ = opaque.ToOpaque(lastCursor)
	}
	return out, nil
}

func extractSearchQueryFromRequest(r *http.Request) (q *SearchQuery) {
	q = &SearchQuery{}
	q.Query = r.FormValue("q")
	q.StartBlock = uint32(int64Input(r.FormValue("start_block")))
	if blockCount := r.FormValue("block_count"); blockCount != "" {
		q.BlockCount = uint32(int64Input(blockCount))
	} else {
		q.BlockCount = math.MaxUint32
	}

	if limit := r.FormValue("limit"); limit != "" {
		q.Limit = uint64(int64Input(limit))
	} else {
		q.Limit = 100
	}

	if cursor := r.FormValue("cursor"); cursor != "" {
		q.Cursor, _ = opaque.FromOpaque(cursor)
	}

	q.SortDescending = strings.ToLower(r.FormValue("sort")) == "desc"
	q.WithReversible = strings.ToLower(r.FormValue("with_reversible")) == "true"
	q.Format = r.FormValue("format")

	return q
}
