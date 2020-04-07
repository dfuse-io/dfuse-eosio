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

package eosws

import (
	"context"
	"time"

	"github.com/dfuse-io/derr"
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"github.com/dfuse-io/kvdb"
)

var AccountGetterInstance AccountGetter

type AccountGetter interface {
	GetAccount(ctx context.Context, name string) (out *eos.AccountResp, err error)
}

type APIAccountGetter struct {
	api *eos.API
}

func (g *APIAccountGetter) GetAccount(ctx context.Context, name string) (out *eos.AccountResp, err error) {
	out, err = g.api.GetAccount(ctx, eos.AccountName(name))
	if err == eos.ErrNotFound {
		return nil, DBAccountNotFoundError(ctx, name)
	}

	return
}

func NewApiAccountGetter(api *eos.API) *APIAccountGetter {
	return &APIAccountGetter{
		api: api,
	}
}

func (ws *WSConn) onAccount(ctx context.Context, msg *wsmsg.GetAccount) {
	_, ok := ws.AuthorizeRequest(ctx, msg)
	if !ok {
		return
	}

	accountFromDB, err := ws.db.GetAccount(ctx, msg.Data.Name)
	if err != nil && !isAccountNotFoundError(err) {
		ws.EmitErrorReply(ctx, msg, derr.Wrapf(err, "unable to retrieve account: %s", msg.Data.Name))
		return
	}

	accountFromAPI, err := ws.accountGetter.GetAccount(ctx, msg.Data.Name)
	if err != nil {
		ws.EmitErrorReply(ctx, msg, derr.Wrapf(err, "unable to retrieve account: %s", msg.Data.Name))
		return
	}

	account := mdl.ToV1Account(accountFromDB)
	account.AccountResp = accountFromAPI
	account.HasContract = accountFromAPI.LastCodeUpdate.Time != time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	metrics.DocumentResponseCounter.Inc()
	ws.EmitReply(ctx, msg, wsmsg.NewAccount(account))
}

func isAccountNotFoundError(err error) bool {
	if err == kvdb.ErrNotFound {
		return true
	}

	return derr.ToErrorResponse(context.Background(), err).Code == "data_account_not_found_error"
}
