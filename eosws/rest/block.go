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

package rest

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/validator"
	"github.com/gorilla/mux"
	"github.com/streamingfast/dmetering"
)

func GetBlocksHandler(db eosws.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		errors := eosws.ValidateBlocksRequest(r)
		if len(errors) > 0 {
			eosws.WriteError(w, r, derr.RequestValidationError(r.Context(), errors))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - eosq",
				Method:         "/v0/blocks",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////

			return
		}

		skip, _ := strconv.Atoi(r.FormValue("skip"))
		limit, _ := strconv.Atoi(r.FormValue("limit"))

		dbBlocks, err := db.ListBlocks(r.Context(), uint32(skip), limit)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get blocks"))
			return
		}

		var blockSummaries []*mdl.BlockSummary
		for _, block := range dbBlocks {
			blkSummary, err := mdl.ToV1BlockSummary(block)
			if err != nil {
				eosws.WriteError(w, r, err)
				return
			}
			blockSummaries = append(blockSummaries, blkSummary)
		}

		eosws.WriteJSON(w, r, blockSummaries)

		count := int64(len(dbBlocks))
		if count == 0 {
			count = 1
		}
		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API - eosq",
			Method:         "/v0/blocks",
			RequestsCount:  1,
			ResponsesCount: count,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}

func GetBlockHandler(db eosws.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		vars := mux.Vars(r)
		id := vars["blockID"]

		err := validator.HexRowRule("blockID", "", "", id)
		if err != nil || len(id) != 64 {
			eosws.WriteError(w, r, derr.RequestValidationError(r.Context(), url.Values{
				"blockID": []string{"The blockID parameters is not a valid hexadecimal"},
			}))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - eosq",
				Method:         "/v0/blocks/{blockID}",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////

			return
		}

		dbBlock, err := db.GetBlock(r.Context(), id)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get block"))
			return
		}

		blockSummary, err := mdl.ToV1BlockSummary(dbBlock)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to to convert block"))
			return
		}

		dbSiblingBlocks, err := db.ListSiblingBlocks(r.Context(), blockSummary.BlockNum, 5)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get sibling blocks"))
			return
		}

		blockSummary.SiblingBlocks = make([]*mdl.BlockSummary, len(dbSiblingBlocks))
		for i, dbSiblingBlock := range dbSiblingBlocks {
			v1Block, err := mdl.ToV1BlockSummary(dbSiblingBlock)
			if err != nil {
				eosws.WriteError(w, r, derr.Wrap(err, "failed to convert block"))
				return
			}
			blockSummary.SiblingBlocks[i] = v1Block
		}

		eosws.WriteJSON(w, r, blockSummary)

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API - eosq",
			Method:         "/v0/blocks/{blockID}",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}

func GetBlockTransactionsHandler(db eosws.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		vars := mux.Vars(r)
		id := vars["blockID"]

		errors := eosws.ValidateListRequest(r)
		if len(errors) > 0 {
			eosws.WriteError(w, r, derr.RequestValidationError(r.Context(), errors))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - eosq",
				Method:         "/v0/blocks/{blockID}/transactions",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////

			return
		}

		cursor, _ := parseCursor(r.FormValue("cursor"))
		limit, _ := strconv.Atoi(r.FormValue("limit"))

		dbTransactionList, err := db.ListTransactionsForBlockID(r.Context(), id, cursor, limit)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get block transactions"))
			return
		}

		eosws.WriteJSON(w, r, dbTransactionList)

		count := int64(len(dbTransactionList.Transactions))
		if count == 0 {
			count = 1
		}

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API - eosq",
			Method:         "/v0/blocks/{blockID}/transactions",
			RequestsCount:  1,
			ResponsesCount: count,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}
