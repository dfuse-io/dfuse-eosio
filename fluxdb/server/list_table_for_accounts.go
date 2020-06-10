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

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dhammer"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	"go.uber.org/zap"
)

func (srv *EOSServer) listTablesRowsForAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlog := logging.Logger(ctx, zlog)

	errors := validateListTablesRowsForAccountsRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractListTablesRowsForAccountsRequest(r)
	zlog.Debug("extracted request", zap.Reflect("request", request))

	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("prepare read failed: %w", err))
		return
	}

	// Sort by contract so at least, a constant order is kept across calls
	sort.Slice(request.Accounts, func(leftIndex, rightIndex int) bool {
		return request.Accounts[leftIndex] < request.Accounts[rightIndex]
	})

	accounts := make([]interface{}, len(request.Accounts))
	for i, s := range request.Accounts {
		accounts[i] = string(s)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	nailer := dhammer.NewNailer(64, func(ctx context.Context, i interface{}) (interface{}, error) {
		account := i.(string)

		tablet := fluxdb.NewContractStateTablet(account, request.Scope, request.Table)
		rows, serializationInfo, err := srv.readContractStateTable(
			ctx,
			tablet,
			actualBlockNum,
			request.ToJSON,
			speculativeWrites,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to read contract state tablet %q: %w", tablet, err)
		}

		resp := &getTableResponse{
			Account: account,
			Scope:   request.Scope,
			readTableResponse: &readTableResponse{
				ABI:  serializationInfo.abi,
				Rows: make([]*tableRow, len(rows)),
			},
		}

		for i, row := range rows {
			response, err := toTableRow(row.(*fluxdb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
			if err != nil {
				return nil, fmt.Errorf("creating table row response failed: %w", err)
			}

			resp.Rows[i] = response
		}

		return resp, nil
	})

	nailer.PushAll(ctx, accounts)

	response := &getMultiTableRowsResponse{
		commonStateResponse: newCommonGetResponse(upToBlockID, lastWrittenBlockID),
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
			}
			response.Tables = append(response.Tables, next.(*getTableResponse))
		}
	}
}

type listTablesRowsForAccountsRequest struct {
	*readRequestCommon

	Accounts []string `json:"accounts"`
	Table    string   `json:"table"`
	Scope    string   `json:"scope"`
}

func validateListTablesRowsForAccountsRequest(r *http.Request) url.Values {
	errors := validator.ValidateQueryParams(r, withCommonValidationRules(validator.Rules{
		"accounts": []string{"required", "fluxdb.eos.accountsList"},
		"table":    []string{"required", "fluxdb.eos.name"},
		"scope":    []string{"fluxdb.eos.extendedName"},
	}))

	// Let's ensure the scope param is at least present (but can be the empty string)
	if _, ok := r.Form["scope"]; !ok {
		errors["scope"] = []string{"The scope field is required"}
	}

	return errors
}

func extractListTablesRowsForAccountsRequest(r *http.Request) *listTablesRowsForAccountsRequest {
	accounts := validator.ExplodeNames(r.FormValue("accounts"), "|")

	return &listTablesRowsForAccountsRequest{
		readRequestCommon: extractReadRequestCommon(r),

		Table:    r.FormValue("table"),
		Accounts: accounts,
		Scope:    r.FormValue("scope"),
	}
}
