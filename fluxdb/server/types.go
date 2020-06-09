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

package server

import (
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	eos "github.com/eoscanada/eos-go"
)

//
/// HTTP Requests
//

type readRequestCommon struct {
	BlockNum     uint32 `json:"block_num"`
	Key          string `json:"key"`
	KeyType      string `json:"key_type"`
	Offset       int    `json:"offset"`
	Limit        int    `json:"limit"`
	ToJSON       bool   `json:"json"`
	WithABI      bool   `json:"with_abi"`
	WithBlockNum bool   `json:"with_block_num"`
}

//
/// HTTP Responses
//

type commonStateResponse struct {
	UpToBlockID              string `json:"up_to_block_id,omitempty"`
	UpToBlockNum             uint32 `json:"up_to_block_num,omitempty"`
	LastIrreversibleBlockID  string `json:"last_irreversible_block_id,omitempty"`
	LastIrreversibleBlockNum uint32 `json:"last_irreversible_block_num,omitempty"`
}

func newCommonGetResponse(upToBlockID string, lastIrreversibleBlockID string) *commonStateResponse {
	return &commonStateResponse{
		UpToBlockID:              upToBlockID,
		UpToBlockNum:             fluxdb.BlockNum(upToBlockID),
		LastIrreversibleBlockID:  lastIrreversibleBlockID,
		LastIrreversibleBlockNum: fluxdb.BlockNum(lastIrreversibleBlockID),
	}
}

type tableRow struct {
	Key      string
	Data     interface{}
	Payer    string
	BlockNum uint32
}

type readTableRowResponse struct {
	ABI *eos.ABI  `json:"abi"`
	Row *tableRow `json:"row"`
}

type readTableResponse struct {
	ABI  *eos.ABI    `json:"abi"`
	Rows []*tableRow `json:"rows"`
}

type onTheFlyABISerializer struct {
	abi             *eos.ABI
	abiAtBlockNum   uint32
	tableTypeName   string
	rowDataToDecode []byte
}

type getTableRowsResponse struct {
	*commonStateResponse
	*readTableResponse
}

type getMultiTableRowsResponse struct {
	*commonStateResponse

	Tables []*getTableResponse `json:"tables,omitempty"`
}

type getTableResponse struct {
	Account string `json:"account"`
	Scope   string `json:"scope"`
	*readTableResponse
}
