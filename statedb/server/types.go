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
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/streamingfast/bstream"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	eos "github.com/eoscanada/eos-go"
)

//
/// HTTP Requests
//

type readRequestCommon struct {
	BlockNum     uint64 `json:"block_num"`
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
	UpToBlockNum             uint64 `json:"up_to_block_num,omitempty"`
	LastIrreversibleBlockID  string `json:"last_irreversible_block_id,omitempty"`
	LastIrreversibleBlockNum uint64 `json:"last_irreversible_block_num,omitempty"`
}

func newCommonGetResponse(upToBlock, lastIrreversibleBlock bstream.BlockRef) *commonStateResponse {
	out := &commonStateResponse{
		LastIrreversibleBlockID:  lastIrreversibleBlock.ID(),
		LastIrreversibleBlockNum: lastIrreversibleBlock.Num(),
	}

	if upToBlock != nil {
		out.UpToBlockID = upToBlock.ID()
		out.UpToBlockNum = upToBlock.Num()
	}

	return out
}

type tableRow struct {
	Key      string
	Data     interface{}
	Payer    string
	BlockNum uint64
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
	serializationInfo *rowSerializationInfo
	rowDataToDecode   []byte
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

func toTableRow(row *statedb.ContractStateRow, keyConverter KeyConverter, serializationInfo *rowSerializationInfo, withBlockNum bool) (*tableRow, error) {
	primaryKey := statedb.ContractStatePrimaryKey(row.PrimaryKey())
	payer, data, err := row.Info()
	if err != nil {
		return nil, fmt.Errorf("unable to read contract state row %q value: %w", primaryKey, err)
	}

	primaryKeyString, err := convertKey(row.PrimaryKey(), keyConverter)
	if err != nil {
		return nil, fmt.Errorf("unable to convert key %s: %w", row.PrimaryKey(), err)
	}

	response := &tableRow{
		Key:   primaryKeyString,
		Payer: payer,
	}

	if withBlockNum {
		response.BlockNum = row.Height()
	}

	if serializationInfo != nil {
		response.Data = &onTheFlyABISerializer{
			serializationInfo: serializationInfo,
			rowDataToDecode:   data,
		}
	} else {
		response.Data = data
	}

	return response, nil
}

func convertKey(key []byte, keyConverter KeyConverter) (string, error) {
	if _, ok := keyConverter.(*NameKeyConverter); ok {
		return bytesToName(key), nil
	}

	return keyConverter.ToString(binary.BigEndian.Uint64(key))
}

func bytesToName(bytes []byte) string {
	return eos.NameToString(binary.BigEndian.Uint64(bytes))
}

// hexBytes is used within JSON exchange object to serialize as an hexadecimal
// encoded value
type hexBytes []byte

func (t hexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(t))
}

func (t *hexBytes) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*t, err = hex.DecodeString(s)
	return
}

func (t hexBytes) String() string {
	return hex.EncodeToString(t)
}
