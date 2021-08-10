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

	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/validator"
	"github.com/gorilla/mux"
	"github.com/streamingfast/dmetering"
)

func GetTransactionHandler(db eosws.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		pathVariables := mux.Vars(r)
		request := &getTransactionRequest{
			TransactionID: pathVariables["id"],
		}

		errors := validateGetTransactionRequest(request)
		if len(errors) > 0 {
			eosws.WriteError(w, r, derr.RequestValidationError(r.Context(), errors))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API",
				Method:         "/v0/transactions/{id}",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}

		transaction, err := db.GetTransaction(r.Context(), request.TransactionID)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed to get transaction"))
			return
		}

		transactionLifecycle, err := mdl.ToV1TransactionLifecycle(transaction)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrap(err, "failed transform transaction to model"))
			return
		}

		eosws.WriteJSON(w, r, transactionLifecycle)

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API",
			Method:         "/v0/transactions/{id}",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}

type getTransactionRequest struct {
	TransactionID string `json:"id"`
}

func validateGetTransactionRequest(request *getTransactionRequest) url.Values {
	return validator.ValidateStruct(request, validator.Rules{
		"id": []string{"required", "eos.trxID"},
	})
}
