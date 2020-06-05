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

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (srv *EOSServer) listKeyAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)

	errors := validateListKeyAccountsRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractListKeyAccountsRequest(r)
	zlogger.Debug("extracted request", zap.Reflect("request", request))

	accountNames, actualBlockNum, err := srv.listKeyAccounts(ctx, request.PublicKey, request.BlockNum)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("list key accounts: %w", err))
		return
	}

	if accountNames == nil {
		accountNames = []eos.AccountName{}
	}

	writeResponse(ctx, w, &listKeyAccountsResponse{
		BlockNum:     actualBlockNum,
		AccountNames: accountNames,
	})
}

type listKeyAccountsRequest struct {
	PublicKey string `json:"public_key"`
	BlockNum  uint32 `json:"block_num"`
}

type listKeyAccountsResponse struct {
	BlockNum     uint32            `json:"block_num"`
	AccountNames []eos.AccountName `json:"account_names"`
}

func validateListKeyAccountsRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"public_key": []string{"required", "fluxdb.eos.publicKey"},
		"block_num":  []string{"fluxdb.eos.blockNum"},
	})
}

func extractListKeyAccountsRequest(r *http.Request) *listKeyAccountsRequest {
	blockNum64, _ := strconv.ParseInt(r.FormValue("block_num"), 10, 64)

	return &listKeyAccountsRequest{
		PublicKey: r.FormValue("public_key"),
		BlockNum:  uint32(blockNum64),
	}
}
