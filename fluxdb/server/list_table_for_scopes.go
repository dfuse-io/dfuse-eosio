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

package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/abourget/llerrgroup"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	"go.uber.org/zap"
)

func (srv *EOSServer) listTablesRowsForScopesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlog := logging.Logger(ctx, zlog)

	errors := validateListTablesRowsForScopesRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractListTablesRowsForScopesRequest(r)
	zlog.Debug("extracted request", zap.Reflect("request", request))

	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("prepare read failed: %w", err))
		return
	}

	scopeCount := len(request.Scopes)
	tableResponses := make(chan *getTableResponse, scopeCount)
	keyConverter := getKeyConverterForType(request.KeyType)
	group := llerrgroup.New(parallelReadRequestCount)

	for _, scope := range request.Scopes {
		if group.Stop() {
			zlog.Debug("read table operations group completed")
			break
		}

		scope := scope
		group.Go(func() error {
			response, err := srv.readTable(
				ctx,
				actualBlockNum,
				request.Account,
				request.Table,
				scope,
				request.readRequestCommon,
				keyConverter,
				speculativeWrites,
			)

			if err != nil {
				return err
			}

			zlog.Debug("adding table read rows to response channel", zap.Int("row_count", len(response.Rows)))
			tableResponses <- &getTableResponse{
				Account:           request.Account,
				Scope:             scope,
				readTableResponse: response,
			}
			return nil
		})
	}

	zlog.Debug("waiting for all read requests to finish")
	if err := group.Wait(); err != nil {
		writeError(ctx, w, fmt.Errorf("waiting for read requests: %w", err))
		return
	}

	zlog.Debug("closing responses channel", zap.Int("response_count", len(tableResponses)))
	close(tableResponses)

	response := &getMultiTableRowsResponse{
		commonStateResponse: newCommonGetResponse(upToBlockID, lastWrittenBlockID),
	}

	zlog.Info("assembling table responses")
	for tableResponse := range tableResponses {
		response.Tables = append(response.Tables, tableResponse)
	}

	// Sort by scope so at least, a constant order is kept across calls
	sort.Slice(response.Tables, func(leftIndex, rightIndex int) bool {
		return response.Tables[leftIndex].Scope < response.Tables[rightIndex].Scope
	})

	zlog.Debug("streaming response", zap.Int("table_count", len(response.Tables)), zap.Reflect("common_response", response.commonStateResponse))
	streamResponse(ctx, w, response)
}

type listTablesRowsForScopesRequest struct {
	*readRequestCommon
	Account string   `json:"account"`
	Table   string   `json:"table"`
	Scopes  []string `json:"scopes"`
}

func validateListTablesRowsForScopesRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, withCommonValidationRules(validator.Rules{
		"account": []string{"required", "fluxdb.eos.name"},
		"table":   []string{"required", "fluxdb.eos.name"},
		"scopes":  []string{"required", "fluxdb.eos.scopesList"},
	}))
}

func extractListTablesRowsForScopesRequest(r *http.Request) *listTablesRowsForScopesRequest {
	scopes := validator.ExplodeNames(r.FormValue("scopes"), "|")

	return &listTablesRowsForScopesRequest{
		readRequestCommon: extractReadRequestCommon(r),

		Account: r.FormValue("account"),
		Table:   r.FormValue("table"),
		Scopes:  scopes,
	}
}
