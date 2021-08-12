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
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/validator"
	eos "github.com/eoscanada/eos-go"
	"github.com/francoispqt/gojay"
	"go.uber.org/zap"
)

func (srv *EOSServer) decodeABIHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlog := logging.Logger(ctx, zlog)

	request := &decodeABIRequest{}
	err := extractDecodeABIRequest(r, request)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("extracting request: %w", err))
		return
	}

	zlog.Debug("extracted request", zap.Reflect("request", request))

	account := request.Account
	tableName := request.Table
	abiEntry, err := srv.fetchABI(ctx, string(account), request.BlockNum)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("fetch ABI: %w", err))
		return
	}

	if abiEntry == nil {
		writeError(ctx, w, statedb.DataABINotFoundError(ctx, string(account), request.BlockNum))
		return
	}

	abi, _, err := abiEntry.ABI()
	if err != nil {
		writeError(ctx, w, fmt.Errorf("decode ABI: %w", err))
		return
	}

	tableDef := abi.TableForName(request.Table)
	if tableDef == nil {
		writeError(ctx, w, statedb.DataTableNotFoundError(ctx, account, tableName))
		return
	}

	response := &decodeABIResponse{
		BlockNum: abiEntry.Height(),
		Account:  request.Account,
		Table:    request.Table,
	}

	for _, hexRow := range request.HexRows {
		hexData, _ := hex.DecodeString(hexRow)
		decodedRow, err := abi.DecodeTableRowTyped(tableDef.Type, hexData)
		if err != nil {
			writeError(ctx, w, statedb.DataDecodingRowError(ctx, hexRow))
			return
		}

		response.DecodedRows = append(response.DecodedRows, string(decodedRow))
	}

	zlog.Debug("streaming response", zap.Int("row_count", len(response.DecodedRows)), zap.Uint64("block_nun", response.BlockNum))
	streamResponse(ctx, w, response)
}

type decodeABIRequest struct {
	Account  eos.AccountName `json:"account"`
	Table    eos.TableName   `json:"table"`
	HexRows  []string        `json:"hex_rows"`
	BlockNum uint64          `json:"block_num"`
}

type decodeABIResponse struct {
	BlockNum    uint64          `json:"block_num"`
	Account     eos.AccountName `json:"account"`
	Table       eos.TableName   `json:"table"`
	DecodedRows []string        `json:"rows"` // The slice elements are valid JSON string
}

func validateDecodeABIRequest(r *http.Request, request *decodeABIRequest) url.Values {
	return validator.ValidateJSONBody(r, request, validator.Rules{
		"account":  []string{"required", "fluxdb.eos.name"},
		"hex_rows": []string{"required", "fluxdb.eos.hexRows"},
		"table":    []string{"required", "fluxdb.eos.name"},
	})
}

func extractDecodeABIRequest(r *http.Request, request *decodeABIRequest) error {
	ctx := r.Context()
	if r.Body == nil {
		return derr.MissingBodyError(ctx)
	}

	requestErrors := validateDecodeABIRequest(r, request)
	if len(requestErrors) > 0 {
		if _, ok := requestErrors["_error"]; ok {
			return derr.InvalidJSONError(ctx, errors.New(requestErrors["_error"][0]))
		}

		return derr.RequestValidationError(ctx, requestErrors)
	}

	return nil
}

func (r *decodeABIResponse) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddIntKey("block_num", int(r.BlockNum))
	enc.AddStringKey("account", string(r.Account))
	enc.AddStringKey("table", string(r.Table))

	enc.AddArrayKey("rows", gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
		lastIdx := len(r.DecodedRows) - 1
		for idx, decodedRow := range r.DecodedRows {
			enc.AppendBytes([]byte(decodedRow))
			if idx != lastIdx {
				enc.AppendByte(',')
			}
		}
	}))
}

func (r *decodeABIResponse) IsNil() bool { return r == nil }
