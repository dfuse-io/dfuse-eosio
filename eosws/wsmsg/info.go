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
	"fmt"
	"time"
)

func init() {
	RegisterIncomingMessage("get_head_info", GetHeadInfo{})
	RegisterOutgoingMessage("head_info", HeadInfo{})
}

type GetHeadInfo struct {
	CommonIn
}

func (m *GetHeadInfo) Validate() error {
	if !m.Fetch && !m.Listen {
		return fmt.Errorf("one of 'listen' or 'fetch' required (both supported)")
	}

	if m.IrreversibleOnly {
		return fmt.Errorf("'irreversible_only' is not supported")
	}

	return nil
}

type HeadInfo struct {
	CommonOut
	Data *HeadInfoData `json:"data"`
}

type HeadInfoData struct {
	LastIrreversibleBlockNum uint32    `json:"last_irreversible_block_num"`
	LastIrreversibleBlockId  string    `json:"last_irreversible_block_id"`
	HeadBlockNum             uint32    `json:"head_block_num"`
	HeadBlockId              string    `json:"head_block_id"`
	HeadBlockTime            time.Time `json:"head_block_time"`
	HeadBlockProducer        string    `json:"head_block_producer"`
}
