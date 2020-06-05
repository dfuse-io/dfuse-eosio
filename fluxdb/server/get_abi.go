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

func (srv *EOSServer) getABIHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)

	errors := validateGetABIRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractGetABIRequest(r)
	zlogger.Debug("extracted request", zap.Reflect("request", request))

	abiRow, abi, err := srv.fetchABI(ctx, string(request.Account), request.BlockNum, request.ToJSON)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("fetch ABI: %w", err))
		return
	}

	response := &getABIResponse{
		BlockNum: abiRow.BlockNum,
		Account:  request.Account,
	}

	if request.ToJSON {
		response.ABI = abi
	} else {
		response.ABI = eos.HexBytes(abiRow.PackedABI)
	}

	writeResponse(ctx, w, response)
}

type getABIRequest struct {
	Account  eos.AccountName `json:"account"`
	BlockNum uint32          `json:"block_num"`
	ToJSON   bool            `json:"json"`
}

type getABIResponse struct {
	BlockNum uint32          `json:"block_num"`
	Account  eos.AccountName `json:"account"`
	ABI      interface{}     `json:"abi"`
}

func validateGetABIRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"account":   []string{"required", "fluxdb.eos.name"},
		"block_num": []string{"fluxdb.eos.blockNum"},
		"json":      []string{"bool"},
	})
}

func extractGetABIRequest(r *http.Request) *getABIRequest {
	blockNum64, _ := strconv.ParseInt(r.FormValue("block_num"), 10, 64)

	return &getABIRequest{
		BlockNum: uint32(blockNum64),
		Account:  eos.AccountName(r.FormValue("account")),
		ToJSON:   boolInput(r.FormValue("json")),
	}
}
