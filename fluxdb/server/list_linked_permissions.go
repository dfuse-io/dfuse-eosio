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
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (srv *EOSServer) listLinkedPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	zlog := logging.Logger(ctx, zlog)

	errors := validateGetLinkedPermissionsRequest(r)
	if len(errors) > 0 {
		writeError(ctx, w, derr.RequestValidationError(ctx, errors))
		return
	}

	request := extractGetLinkedPermissionsRequest(r)
	zlog.Debug("extracted request", zap.Reflect("request", request))

	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := srv.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("prepare read failed: %w", err))
		return
	}

	linkedPermissions, err := srv.db.ReadLinkedPermissions(ctx, actualBlockNum, request.Account, speculativeWrites)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("reading linked permissions failed: %w", err))
		return
	}

	response := &listLinkedPermissionsResponse{
		commonStateResponse: newCommonGetResponse(upToBlockID, lastWrittenBlockID),
		LinkedPermissions:   linkedPermissions,
	}

	writeResponse(ctx, w, response)
}

type listLinkedPermissionsRequest struct {
	BlockNum uint32          `json:"block_num"`
	Account  eos.AccountName `json:"account"`
}

type listLinkedPermissionsResponse struct {
	*commonStateResponse

	LinkedPermissions []*fluxdb.LinkedPermission `json:"linked_permissions"`
}

func validateGetLinkedPermissionsRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"block_num": []string{"fluxdb.eos.blockNum"},
		"account":   []string{"required", "fluxdb.eos.name"},
	})
}

func extractGetLinkedPermissionsRequest(r *http.Request) *listLinkedPermissionsRequest {
	blockNum64, _ := strconv.ParseInt(r.FormValue("block_num"), 10, 64)

	return &listLinkedPermissionsRequest{
		BlockNum: uint32(blockNum64),
		Account:  eos.AccountName(r.FormValue("account")),
	}
}
