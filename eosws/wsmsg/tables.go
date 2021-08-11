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

	"github.com/streamingfast/bstream/forkable"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
	eos "github.com/eoscanada/eos-go"
)

func init() {
	RegisterIncomingMessage("get_table_rows", GetTableRows{})
	RegisterOutgoingMessage("table_snapshot", TableSnapshot{})
	RegisterOutgoingMessage("table_delta", TableDelta{})
}

// INCOMING
type GetTableRowsData struct {
	JSON bool `json:"json,omitempty"`

	Code      eos.AccountName `json:"code"`
	Scope     *eos.Name       `json:"scope"`
	TableName eos.TableName   `json:"table"`
}

type GetTableRows struct {
	CommonIn
	Data GetTableRowsData `json:"data"`
}

func (m *GetTableRows) Validate(ctx context.Context) error {
	// CHECK if the tx data is appropriate.
	if !m.Listen && !m.Fetch {
		return fmt.Errorf("one of 'listen' or 'fetch' required (both supported)")
	}
	if m.Data.Code == "" {
		return fmt.Errorf("'data.code' required")
	}
	if m.Data.TableName == "" {
		return fmt.Errorf("'data.table' required")
	}
	if m.Data.Scope == nil {
		return fmt.Errorf("'data.scope' required")
	}
	if m.IrreversibleOnly {
		return fmt.Errorf("'irreversible_only' is not supported")
	}

	return nil
}

// OUTGOING

type TableDelta struct {
	CommonOut
	Data struct {
		BlockNum uint32   `json:"block_num"`
		DBOp     *v1.DBOp `json:"dbop"`
		Step     string   `json:"step"`
	} `json:"data"`
}

func NewTableDelta(blockNum uint32, dbop *v1.DBOp, stepType forkable.StepType) *TableDelta {
	out := &TableDelta{}
	out.Data.BlockNum = blockNum
	out.Data.DBOp = dbop
	out.Data.Step = stepType.String()
	return out
}

type TableSnapshot struct {
	CommonOut
	Data struct {
		Rows []json.RawMessage `json:"rows"`
		// insert the schtuffs here..
	} `json:"data"`
}
