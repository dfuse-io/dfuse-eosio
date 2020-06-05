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

	accountCount := len(request.Accounts)
	tableResponses := make(chan *getTableResponse, accountCount)
	keyConverter := getKeyConverterForType(request.KeyType)
	group := llerrgroup.New(parallelReadRequestCount)

	zlog.Debug("starting read table operations group", zap.Int("account_count", accountCount))
	for _, account := range request.Accounts {
		if group.Stop() {
			zlog.Debug("read table operations group completed")
			break
		}

		account := account
		group.Go(func() error {
			response, err := srv.readTable(
				ctx,
				actualBlockNum,
				account,
				request.Table,
				request.Scope,
				request.readRequestCommon,
				keyConverter,
				speculativeWrites,
			)

			if err != nil {
				return err
			}

			zlog.Debug("adding table read rows to response channel", zap.Int("row_count", len(response.Rows)))
			tableResponses <- &getTableResponse{
				Account:           account,
				Scope:             request.Scope,
				readTableResponse: response,
			}

			return nil
		})
	}

	zlog.Info("waiting for all read requests to finish")
	if err := group.Wait(); err != nil {
		writeError(ctx, w, fmt.Errorf("waiting for all read request to complete: %w", err))
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

	// Sort by code so at least, a constant order is kept across calls
	sort.Slice(response.Tables, func(leftIndex, rightIndex int) bool {
		return response.Tables[leftIndex].Account < response.Tables[rightIndex].Account
	})

	zlog.Debug("streaming response", zap.Int("table_count", len(response.Tables)), zap.Reflect("common_response", response.commonStateResponse))
	streamResponse(ctx, w, response)
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
