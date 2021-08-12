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
	"strconv"

	"github.com/eoscanada/eos-go"

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/streamingfast/logging"
	"github.com/dfuse-io/validator"
	"go.uber.org/zap"
)

func (srv *EOSServer) listTableRowsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlog := logging.Logger(ctx, zlog)

	errors := validateGetTableRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractGetTableRequest(r)
	zlog.Debug("extracted request", zap.Reflect("request", request))

	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, request.IrreversibleOnly)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("prepare read failed: %w", err))
		return
	}

	tablet := statedb.NewContractStateTablet(request.Account, request.Table, request.Scope)
	rows, serializationInfo, err := srv.readContractStateTable(
		ctx,
		tablet,
		actualBlockNum,
		request.ToJSON,
		speculativeWrites,
	)

	if err != nil {
		writeError(ctx, w, fmt.Errorf("read rows failed: %w", err))
		return
	}

	var abi *eos.ABI
	if serializationInfo != nil && request.WithABI {
		abi = serializationInfo.abi
	}

	response := &getTableRowsResponse{
		commonStateResponse: newCommonGetResponse(upToBlock, lastWrittenBlock),
		readTableResponse: &readTableResponse{
			ABI:  abi,
			Rows: []*tableRow{},
		},
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	for _, row := range rows {
		tableRow, err := toTableRow(row.(*statedb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
		if err != nil {
			writeError(ctx, w, fmt.Errorf("creating table row failed: %w", err))
			return
		}
		response.Rows = append(response.Rows, tableRow)
	}

	zlog.Debug("streaming response", zap.Int("row_count", len(response.readTableResponse.Rows)), zap.Reflect("common_response", response.commonStateResponse))
	streamResponse(ctx, w, response)
}

type listTableRowsRequest struct {
	*readRequestCommon

	IrreversibleOnly bool   `json:"irreversible_only"`
	Account          string `json:"account"`
	Table            string `json:"table"`
	Scope            string `json:"scope"`
}

func validateGetTableRequest(r *http.Request) url.Values {
	errors := validator.ValidateQueryParams(r, withCommonValidationRules(validator.Rules{
		"account":           []string{"required", "fluxdb.eos.name"},
		"table":             []string{"required", "fluxdb.eos.name"},
		"scope":             []string{"fluxdb.eos.extendedName"},
		"irreversible_only": []string{"bool"},
	}))

	// Let's ensure the scope param is at least present (but can be the empty string)
	if _, ok := r.Form["scope"]; !ok {
		errors["scope"] = []string{"The scope field is required"}
	}

	return errors
}

func extractGetTableRequest(r *http.Request) *listTableRowsRequest {
	irreversibleOnly, _ := strconv.ParseBool(r.FormValue("irreversible_only"))
	return &listTableRowsRequest{
		readRequestCommon: extractReadRequestCommon(r),

		Table:            r.FormValue("table"),
		Account:          r.FormValue("account"),
		Scope:            r.FormValue("scope"),
		IrreversibleOnly: irreversibleOnly,
	}
}
