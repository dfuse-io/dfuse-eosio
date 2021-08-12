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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/streamingfast/dhammer"
	eos "github.com/eoscanada/eos-go"

	"github.com/streamingfast/logging"
	"github.com/dfuse-io/validator"
	"github.com/streamingfast/derr"
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

	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("prepare read failed: %w", err))
		return
	}

	var serializationInfo *rowSerializationInfo
	if request.ToJSON {
		serializationInfo, err = srv.newRowSerializationInfo(ctx, request.Account, request.Table, actualBlockNum, speculativeWrites)
		if err != nil {
			writeError(ctx, w, fmt.Errorf("unable to obtain serialziation info: %w", err))
			return
		}
	}

	// Sort by scope so at least, a constant order is kept across calls
	sort.Slice(request.Scopes, func(leftIndex, rightIndex int) bool {
		return request.Scopes[leftIndex] < request.Scopes[rightIndex]
	})

	scopes := make([]interface{}, len(request.Scopes))
	for i, s := range request.Scopes {
		scopes[i] = string(s)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	nailer := dhammer.NewNailer(64, func(ctx context.Context, i interface{}) (interface{}, error) {
		scope := i.(string)

		tablet := statedb.NewContractStateTablet(request.Account, request.Table, scope)
		tabletRows, err := srv.db.ReadTabletAt(
			ctx,
			actualBlockNum,
			tablet,
			speculativeWrites,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to read tablet %s at %d: %w", tablet, request.BlockNum, err)
		}

		var abi *eos.ABI
		if serializationInfo != nil && request.WithABI {
			abi = serializationInfo.abi
		}

		resp := &getTableResponse{
			Account: request.Account,
			Scope:   scope,
			readTableResponse: &readTableResponse{
				ABI:  abi,
				Rows: make([]*tableRow, len(tabletRows)),
			},
		}

		for i, tabletRow := range tabletRows {
			response, err := toTableRow(tabletRow.(*statedb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
			if err != nil {
				return nil, fmt.Errorf("creating table row response failed: %w", err)
			}

			resp.Rows[i] = response
		}

		return resp, nil
	}, dhammer.NailerLogger(zlog))

	nailer.PushAll(ctx, scopes)

	response := &getMultiTableRowsResponse{
		commonStateResponse: newCommonGetResponse(upToBlock, lastWrittenBlock),
	}

	for {
		select {
		case <-ctx.Done():
			writeError(ctx, w, fmt.Errorf("request terminated prior to completed: %w", err))
			return
		case next, ok := <-nailer.Out:
			if !ok {
				zlog.Debug("streaming response", zap.Int("table_count", len(response.Tables)), zap.Reflect("common_response", response.commonStateResponse))
				streamResponse(ctx, w, response)
				return
			}

			response.Tables = append(response.Tables, next.(*getTableResponse))
		}
	}
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
