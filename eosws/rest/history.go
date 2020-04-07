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
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/eoscanada/eos-go"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	fluxcli "github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	"github.com/tidwall/gjson"
)

func GetKeyAccounts(fluxClient fluxcli.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		pubKey := ""
		if r.Method == "GET" {
			pubKey = r.FormValue("public_key")
		} else {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				eosws.WriteError(w, r, derr.Wrap(err, "to read request body"))
				return
			}
			r.Body.Close()

			pubKey = gjson.GetBytes(body, "public_key").String()
		}

		if pubKey == "" {
			//val := url.Values{}
			//val.Set("public_key", pubKey)
			//eosws.WriteError(w, r, derr.RequestValidationError(ctx, val))
			eosws.WriteJSON(w, r, &fluxcli.GetAccountByPubKeyResponses{
				AccountNames: []eos.AccountName{},
			})
			return

		}

		resp, err := fluxClient.GetAccountByPubKey(ctx, 0, pubKey)

		if err != nil {
			response := derr.ToErrorResponse(ctx, err)
			if response.Code == "data_public_key_not_found_error" {
				eosws.WriteJSON(w, r, &fluxcli.GetAccountByPubKeyResponses{
					AccountNames: []eos.AccountName{},
				})
				return
			}
			eosws.WriteError(w, r, derr.Wrap(err, fmt.Sprintf("failed to retrieve account for key: %s", pubKey)))
		}

		eosws.WriteJSON(w, r, resp)
	})
}
