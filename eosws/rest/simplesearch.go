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
	"regexp"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/streamingfast/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/dmetering"
)

func SimpleSearchHandler(db eosws.DB, blockmetaClient *pbblockmeta.Client) http.Handler {
	hexRegex := regexp.MustCompile(`^[0-9a-fA-F]+$`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.FormValue("q")
		if query == "" {
			eosws.WriteError(w, r, derr.RequestValidationError(ctx, url.Values{"q": []string{"query parameter should not be empty"}}))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - eosq",
				Method:         "/v0/simple_search",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}
		lQuery := strings.ToLower(query)

		if len(query) >= 14 {
			block, err := db.GetBlock(ctx, lQuery)
			if err == nil {
				eosws.WriteJSON(w, r, map[string]interface{}{
					"type": "block",
					"data": block,
				})
				//////////////////////////////////////////////////////////////////////
				// Billable event on REST API endpoint
				// WARNING: Ingress / Egress bytess is taken care by the middleware
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "eosws",
					Kind:           "REST API - eosq",
					Method:         "/v0/simple_search",
					RequestsCount:  1,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////

				return
			}

			txResp, err := db.GetTransaction(ctx, lQuery)
			if err == nil {
				eosws.WriteJSON(w, r, map[string]interface{}{
					"type": "transaction",
					"data": txResp,
				})
				//////////////////////////////////////////////////////////////////////
				// Billable event on REST API endpoint
				// WARNING: Ingress / Egress bytess is taken care by the middleware
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "eosws",
					Kind:           "REST API - eosq",
					Method:         "/v0/simple_search",
					RequestsCount:  1,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////
				return
			}
		}

		if num, err := strconv.Atoi(query); err == nil {
			blocks, err := db.GetBlockByNum(ctx, uint32(num))

			if err == nil {
				var block = blocks[0]
				if len(blocks) > 1 {

					for _, blockRef := range blocks {
						if blockRef.Irreversible {
							block = blockRef
						}
					}
				}
				eosws.WriteJSON(w, r, map[string]interface{}{
					"type": "block",
					"data": map[string]string{
						"id": block.Id,
					},
				})
				//////////////////////////////////////////////////////////////////////
				// Billable event on REST API endpoint
				// WARNING: Ingress / Egress bytess is taken care by the middleware
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "eosws",
					Kind:           "REST API - eosq",
					Method:         "/v0/simple_search",
					RequestsCount:  1,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////
				return
			}

		}

		if len(query) <= 13 {
			acctResponse, err := db.GetAccount(ctx, lQuery)
			if err == nil {

				account := mdl.ToV1Account(acctResponse)
				account.AccountResp.AccountName = eos.AccountName(acctResponse.Account)

				eosws.WriteJSON(w, r, map[string]interface{}{
					"type": "account",
					"data": account,
				})
				//////////////////////////////////////////////////////////////////////
				// Billable event on REST API endpoint
				// WARNING: Ingress / Egress bytess is taken care by the middleware
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "eosws",
					Kind:           "REST API - eosq",
					Method:         "/v0/simple_search",
					RequestsCount:  1,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////
				return
			}
		}

		if len(query) == 64 && hexRegex.MatchString(query) {
			eosws.WriteJSON(w, r, map[string]interface{}{
				"type": "transaction",
				"data": map[string]interface{}{"id": lQuery},
			})
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - eosq",
				Method:         "/v0/simple_search",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}

		t, err := dateparse.ParseStrict(query)
		if err == nil {
			btResp, err := blockmetaClient.BlockAfter(ctx, t, true)
			if err == nil && pbblockmeta.Timestamp(btResp.Time).Sub(t).Seconds() < 5 {
				eosws.WriteJSON(w, r, map[string]interface{}{
					"type": "block",
					"data": map[string]interface{}{"id": btResp.Id},
				})
				//////////////////////////////////////////////////////////////////////
				// Billable event on REST API endpoint
				// WARNING: Ingress / Egress bytess is taken care by the middleware
				//////////////////////////////////////////////////////////////////////
				dmetering.EmitWithContext(dmetering.Event{
					Source:         "eosws",
					Kind:           "REST API - eosq",
					Method:         "/v0/simple_search",
					RequestsCount:  1,
					ResponsesCount: 1,
				}, ctx)
				//////////////////////////////////////////////////////////////////////
				return
			}
		}

		eosws.WriteError(w, r, derr.HTTPNotFoundError(ctx, nil, derr.C("simple_search_not_found"), "no results found for query"))

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API - eosq",
			Method:         "/v0/simple_search",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}
