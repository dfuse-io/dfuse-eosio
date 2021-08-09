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

	"github.com/araddon/dateparse"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	eos "github.com/eoscanada/eos-go"
	"github.com/streamingfast/dmetering"
)

/*
/blocks
/blocks/{123}/by_whatever ->

/block_id/ ->
*/

func BlockTimeHandler(blockmetaClient *pbblockmeta.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		blockTime := r.FormValue("time")

		t, err := dateparse.ParseStrict(blockTime)
		if err != nil {
			eosws.WriteError(w, r, derr.RequestValidationError(ctx, url.Values{"time": []string{"timeshould be in a recognizable time format (ex: 2012-11-01T22:08:41+00:00)"}}))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API",
				Method:         "/v0/block_id/by_time",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}

		comparator := r.FormValue("comparator")

		var btResp *pbblockmeta.BlockResponse
		switch comparator {
		case "eq":
			btResp, err = blockmetaClient.BlockAt(ctx, t)
		case "gte":
			btResp, err = blockmetaClient.BlockAfter(ctx, t, true)
		case "gt":
			btResp, err = blockmetaClient.BlockAfter(ctx, t, false)
		case "lte":
			btResp, err = blockmetaClient.BlockBefore(ctx, t, true)
		case "lt":
			btResp, err = blockmetaClient.BlockBefore(ctx, t, false)
		default:
			eosws.WriteError(w, r, derr.RequestValidationError(ctx, url.Values{"comparator": []string{"should be one of 'gt', 'gte', 'lt', 'lte' or 'eq'"}}))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API",
				Method:         "/v0/block_id/by_time",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////

			return
		}

		if err != nil {
			eosws.WriteError(w, r, derr.HTTPNotFoundError(ctx, nil, derr.C("block_time_not_found"), "no results found for query"))
			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API",
				Method:         "/v0/block_id/by_time",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
			return
		}

		eosws.WriteJSON(w, r, map[string]interface{}{
			"block": map[string]interface{}{
				"num":  eos.BlockNum(btResp.Id),
				"id":   btResp.Id,
				"time": pbblockmeta.Timestamp(btResp.Time),
			},
		})

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "eosws",
			Kind:           "REST API",
			Method:         "/v0/block_id/by_time",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})
}
