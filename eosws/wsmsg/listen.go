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
)

func init() {
	RegisterIncomingMessage("unlisten", Unlisten{})
	RegisterOutgoingMessage("unlistened", Unlistened{})
	RegisterOutgoingMessage("listening", Listening{})
}

type Unlisten struct {
	CommonIn
	Data struct {
		ReqID string `json:"req_id"`
	} `json:"data"`
}

func (m *Unlisten) Validate(ctx context.Context) error {
	if m.Data.ReqID == "" {
		return fmt.Errorf("'req_id' is required")
	}

	return nil
}

type Unlistened struct {
	CommonOut
	Data struct {
		Success bool `json:"success"`
	} `json:"data"`
}

func NewUnlistened() *Unlistened {
	out := &Unlistened{}
	out.Data.Success = true
	return out
}

type Listening struct {
	CommonOut
	Data struct {
		NextBlock uint32 `json:"next_block"`
	} `json:"data"`
}

func NewListening(nextBlock uint32) *Listening {
	l := &Listening{}
	l.Data.NextBlock = nextBlock
	return l
}
