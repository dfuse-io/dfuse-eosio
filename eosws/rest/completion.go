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
	"net/http"
	"net/url"

	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/completion"
	"github.com/streamingfast/derr"
)

const minPrefixLength = 1

func GetCompletionHandler(completionInstance completion.Completion) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		prefix := r.FormValue("prefix")
		if prefix == "" {
			eosws.WriteError(w, r, derr.RequestValidationError(ctx, url.Values{
				"prefix": []string{"The prefix field is required"},
			}))
			return
		}

		if len(prefix) < minPrefixLength {
			eosws.WriteError(w, r, derr.RequestValidationError(ctx, url.Values{
				"prefix": []string{fmt.Sprintf("The prefix field does not respect minimum length, got %d, expected %d", len(prefix), minPrefixLength)},
			}))
			return
		}

		suggestionSections, err := completionInstance.Complete(prefix, 5)
		if err != nil {
			eosws.WriteError(w, r, derr.Wrapf(err, "unable to complete prefix [%s]", prefix))
			return
		}

		eosws.WriteJSON(w, r, suggestionSections)

		//////////////////////////////////////////////////////////////////////
		// Billable event on REST API endpoint
		// WARNING: Normally there would be a billing event here, but let's call this service "free"
		// It is latency-sensitive and by-design meant to be called in fast, short, repetitive bursts...
		//////////////////////////////////////////////////////////////////////
	})
}
