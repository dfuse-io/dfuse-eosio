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

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetKeyAccounts(stateClient pbstatedb.StateClient) http.Handler {
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
			eosws.WriteJSON(w, r, &GetAccountByPubKeyResponses{
				AccountNames: []string{},
			})
			return

		}

		resp, err := stateClient.GetKeyAccounts(ctx, &pbstatedb.GetKeyAccountsRequest{
			BlockNum:  0,
			PublicKey: pubKey,
		})

		if err != nil {
			if status.Code(err) == codes.NotFound {
				eosws.WriteJSON(w, r, &GetAccountByPubKeyResponses{
					AccountNames: []string{},
				})
				return
			}

			eosws.WriteError(w, r, derr.Wrap(err, fmt.Sprintf("failed to retrieve account for key: %s", pubKey)))
		}

		eosws.WriteJSON(w, r, GetAccountByPubKeyResponses{
			BlockNum:     uint32(resp.BlockNum),
			AccountNames: resp.Accounts,
		})
	})
}

type GetAccountByPubKeyResponses struct {
	BlockNum     uint32   `json:"block_num"`
	AccountNames []string `json:"account_names"`
}
