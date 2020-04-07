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
	"net/http"
	"net/url"
	"strconv"

	"github.com/dfuse-io/derr"
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
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

	scopes, actualBlockNum, err := srv.listTableScopes(ctx, request.Account, request.Table, request.BlockNum)
	if err != nil {
		writeError(ctx, w, derr.Wrap(err, "list table scopes"))
		return
	}

	if scopes == nil {
		scopes = []eos.Name{}
	}

	writeResponse(ctx, w, &listTableScopesResponse{
		BlockNum: actualBlockNum,
		Scopes:   scopes,
	})
}

type listTableScopesRequest struct {
	Account  eos.AccountName `json:"account"`
	Table    eos.TableName   `json:"table"`
	BlockNum uint32          `json:"block_num"`
}

type listTableScopesResponse struct {
	BlockNum uint32     `json:"block_num"`
	Scopes   []eos.Name `json:"scopes"`
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
		BlockNum: uint32(blockNum64),
	}
}
