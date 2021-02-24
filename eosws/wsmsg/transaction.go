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

	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
)

func init() {
	RegisterIncomingMessage("get_transaction", GetTransaction{})
	RegisterIncomingMessage("get_transaction_lifecycle", GetTransaction{})

	RegisterOutgoingMessage("transaction_lifecycle", TransactionLifecycle{})
}

/// GetTransaction, incoming request

type GetTransaction struct {
	CommonIn

	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (m *GetTransaction) Validate(ctx context.Context) error {
	// We must support between 8,64 characters because we search by prefix
	if len(m.Data.ID) < 8 {
		return fmt.Errorf("transaction id too short")
	}

	if len(m.Data.ID) > 64 {
		return fmt.Errorf("transaction id too long")
	}

	// Odd length transaction id cannot be used as prefix for simplicity since we turned them into []byte later on
	if len(m.Data.ID)%2 != 0 {
		return fmt.Errorf("transaction id odd length")
	}

	if !m.Fetch && !m.Listen {
		return fmt.Errorf("one of 'listen' or 'fetch' required (both supported)")
	}
	if m.IrreversibleOnly {
		return fmt.Errorf("'irreversible_only' is not supported")
	}
	return nil
}

type TransactionLifecycle struct {
	CommonOut
	Data struct {
		Lifecycle *v1.TransactionLifecycle `json:"lifecycle"`
	} `json:"data"`
}

func NewTransactionLifecycle(transaction *v1.TransactionLifecycle) *TransactionLifecycle {
	out := &TransactionLifecycle{}
	out.Data.Lifecycle = transaction
	return out
}
