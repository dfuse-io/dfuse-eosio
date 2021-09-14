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

package wsmsg

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/streamingfast/validator"
)

func init() {
	RegisterIncomingMessage("get_account", GetAccount{})
	RegisterOutgoingMessage("account", Account{})
}

// INCOMING
type GetAccountData struct {
	Name string `json:"name"`
}

type GetAccount struct {
	CommonIn
	Data *GetAccountData `json:"data"`
}

func (m *GetAccount) Validate(ctx context.Context) error {
	if !m.Fetch {
		return fmt.Errorf("'fetch' is required")
	}

	if m.Listen {
		return fmt.Errorf("'listen' is not supported")
	}

	if m.IrreversibleOnly {
		return fmt.Errorf("'irreversible_only' is not supported")
	}

	if m.Data.Name == "" {
		return fmt.Errorf("'data.name' is required")
	}

	if err := validator.EOSNameRule("data.name", "", "", m.Data.Name); err != nil {
		return err
	}

	return nil
}

type Account struct {
	CommonOut
	Data struct {
		Account *mdl.Account `json:"account"`
	} `json:"data"`
}

// OUTGOING
func NewAccount(account *mdl.Account) *Account {
	out := &Account{}
	out.Data.Account = account
	return out
}
