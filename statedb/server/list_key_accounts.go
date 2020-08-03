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

	blockNum := uint64(request.BlockNum)
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("unable to prepare read: %w", err))
		return
	}

	tablet := statedb.NewKeyAccountTablet(request.PublicKey)
	tabletRows, err := srv.db.ReadTabletAt(
		ctx,
		actualBlockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		writeError(ctx, w, fmt.Errorf("unable to read tablet at %d: %w", blockNum, err))
		return
	}

	zlogger.Debug("post-processing key accounts", zap.Int("key_account_count", len(tabletRows)))
	accountNames := sortedUniqueKeyAccounts(tabletRows)
	if len(accountNames) == 0 {
		zlogger.Debug("no account found for request, checking if we ever seen this public key")
		seen, err := srv.db.HasSeenAnyRowForTablet(ctx, tablet)
		if err != nil {
			writeError(ctx, w, fmt.Errorf("unable to know if public key was seen once in db: %w", err))
			return
		}

		if !seen {
			writeError(ctx, w, statedb.DataPublicKeyNotFoundError(ctx, request.PublicKey))
			return
		}
	}

	writeResponse(ctx, w, &listKeyAccountsResponse{
		BlockNum:     actualBlockNum,
		AccountNames: accountNames,
	})
}

type listKeyAccountsRequest struct {
	PublicKey string `json:"public_key"`
	BlockNum  uint64 `json:"block_num"`
}

type listKeyAccountsResponse struct {
	BlockNum     uint64   `json:"block_num"`
	AccountNames []string `json:"account_names"`
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
		BlockNum:  uint64(blockNum64),
	}
}

func sortedUniqueKeyAccounts(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	accountNameSet := map[string]bool{}
	for _, tabletRow := range tabletRows {
		account, _ := tabletRow.(*statedb.KeyAccountRow).Explode()
		accountNameSet[account] = true
	}

	i := 0
	out = make([]string, len(accountNameSet))
	for account := range accountNameSet {
		out[i] = account
		i++
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
