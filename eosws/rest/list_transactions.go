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
	"strconv"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dmetering"
)

func ListTransactionsHandler(db eosws.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

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
				Method:         "/v0/transactions",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}

		cursor, _ := parseCursor(r.FormValue("cursor"))
		limit, _ := strconv.Atoi(r.FormValue("limit"))

		dbTransactionList, err := db.ListMostRecentTransactions(r.Context(), cursor, limit)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get transactions"))
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
			Method:         "/v0/transactions",
			RequestsCount:  1,
			ResponsesCount: count,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}
