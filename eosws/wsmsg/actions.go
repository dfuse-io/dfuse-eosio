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
	"encoding/json"
	"fmt"
	v0 "github.com/dfuse-io/eosws-go/mdl/v0"
	"time"

	eos "github.com/eoscanada/eos-go"
)

func init() {
	RegisterIncomingMessage("get_action_traces", GetActionTraces{})
	RegisterIncomingMessage("get_actions", GetActionTraces{})

	RegisterOutgoingMessage("action_trace", ActionTrace{})
	RegisterOutgoingMessage("formatted_transaction", FormattedTransaction{})
}

type GetActionTraces struct {
	CommonIn

	// TODO: Allow filtering by authorization accounts on DB ops
	Data struct {
		Receiver   eos.AccountName `json:"receiver"`    // deprecated (keep plural form)
		Account    eos.AccountName `json:"account"`     // deprecated (keep plural form)
		ActionName eos.ActionName  `json:"action_name"` // deprecated (keep plural form)

		Receivers   string `json:"receivers"`
		Accounts    string `json:"accounts"`
		ActionNames string `json:"action_names"`

		WithInlineTraces bool `json:"with_inline_traces"`
		WithDBOps        bool `json:"with_dbops"`
		WithRAMOps       bool `json:"with_ramops"`
		WithDTrxOps      bool `json:"with_dtrxops"`
		WithTableOps     bool `json:"with_tableops"`
	} `json:"data"`
}

func (m *GetActionTraces) Validate(ctx context.Context) error {
	if m.Data.Account == "" && m.Data.Accounts == "" {
		return fmt.Errorf("'data.accounts' required")
	}
	if !m.Listen {
		return fmt.Errorf("'listen' required")
	}
	if m.Fetch {
		return fmt.Errorf("'fetch' not supported")
	}

	return nil
}

/// Action

type ActionTrace struct {
	CommonOut
	Data struct {
		BlockNum      uint32    `json:"block_num"`
		BlockID       string    `json:"block_id"`
		BlockTime     time.Time `json:"block_time"`
		TransactionID string    `json:"trx_id"`
		ActionIndex   int       `json:"idx"`
		//ActionDepth   int             `json:"depth"`
		Trace json.RawMessage `json:"trace"`

		DBOps    []*v0.DBOp    `json:"dbops,omitempty"`
		RAMOps   []*v0.RAMOp   `json:"ramops,omitempty"`
		DTrxOps  []*v0.DTrxOp  `json:"dtrxops,omitempty"`
		TableOps []*v0.TableOp `json:"tableops,omitempty"`
	} `json:"data"`
}

func NewActionTrace(trxid string, actionIndex int, trace json.RawMessage) *ActionTrace {
	t := &ActionTrace{}
	t.Data.TransactionID = trxid
	t.Data.ActionIndex = actionIndex
	//t.Data.ActionDepth = depth
	t.Data.Trace = trace
	return t
}

/// FormattedTransaction

type FormattedTransaction struct {
	CommonOut
	Data json.RawMessage `json:"data"`
}
