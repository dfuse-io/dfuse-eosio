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
	"strconv"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/fluxdb"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (srv *EOSServer) listTableScopesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)

	errors := validateListTableScopesRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractListTableScopesRequest(r)
	zlogger.Debug("extracted request", zap.Reflect("request", request))

	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("unable to prepare read: %w", err))
		return
	}

	tablet := statedb.NewContractTableScopeTablet(string(request.Account), string(request.Table))
	tabletRows, err := srv.db.ReadTabletAt(
		ctx,
		actualBlockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("unable to read tablet at %d: %w", actualBlockNum, err))
		return
	}

	zlogger.Debug("post-processing table scopes", zap.Int("table_scope_count", len(tabletRows)))
	scopes := sortedScopes(tabletRows)
	if len(scopes) == 0 {
		zlogger.Debug("no scopes found for request, checking if we ever seen this table")
		seen, err := srv.db.HasSeenAnyRowForTablet(ctx, tablet)
		if err != nil {
			writeError(ctx, w, fmt.Errorf("unable to know if table scope was seen once in db: %w", err))
			return
		}

		if !seen {
			writeError(ctx, w, statedb.DataTableNotFoundError(ctx, request.Account, request.Table))
			return
		}
	}

	if len(scopes) <= 0 {
		scopes = []string{}
	}

	writeResponse(ctx, w, &listTableScopesResponse{
		BlockNum: actualBlockNum,
		Scopes:   scopes,
	})
}

type listTableScopesRequest struct {
	Account  eos.AccountName `json:"account"`
	Table    eos.TableName   `json:"table"`
	BlockNum uint64          `json:"block_num"`
}

type listTableScopesResponse struct {
	BlockNum uint64   `json:"block_num"`
	Scopes   []string `json:"scopes"`
}

func validateListTableScopesRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"account":   []string{"required", "fluxdb.eos.name"},
		"table":     []string{"required", "fluxdb.eos.name"},
		"block_num": []string{"fluxdb.eos.blockNum"},
	})
}

func extractListTableScopesRequest(r *http.Request) *listTableScopesRequest {
	blockNum64, _ := strconv.ParseInt(r.FormValue("block_num"), 10, 64)

	return &listTableScopesRequest{
		Account:  eos.AccountName(r.FormValue("account")),
		Table:    eos.TableName(r.FormValue("table")),
		BlockNum: uint64(blockNum64),
	}
}

func sortedScopes(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	out = make([]string, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i] = tabletRow.(*statedb.ContractTableScopeRow).Scope()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
