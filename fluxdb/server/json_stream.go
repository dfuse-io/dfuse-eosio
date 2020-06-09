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
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	eos "github.com/eoscanada/eos-go"
	"github.com/francoispqt/gojay"
)

func (s *onTheFlyABISerializer) MarshalJSON() ([]byte, error) {
	jsonData, err := s.abi.DecodeTableRowTyped(s.tableTypeName, s.rowDataToDecode)
	if err != nil {
		// This can be both a problem from our standpoint as well as a bigger problem showing a bug in our decoder
		return json.Marshal(map[string]interface{}{
			"hex":   eos.HexBytes(s.rowDataToDecode),
			"error": fmt.Sprintf("ABI from block %d, row struct %q, data: %q, err: %s", s.abiAtBlockNum, s.tableTypeName, hex.EncodeToString(s.rowDataToDecode), err),
		})
	}

	// FIXME: something faster than that?
	return json.Marshal(map[string]interface{}{
		"json": json.RawMessage(jsonData),
	})
}

func (r *getTableRowsResponse) MarshalJSONObject(enc *gojay.Encoder) {
	r.commonStateResponse.MarshalJSONObject(enc)
	r.readTableResponse.MarshalJSONObject(enc)
}

func (r *getTableRowsResponse) IsNil() bool { return r == nil }

func (r *getMultiTableRowsResponse) MarshalJSONObject(enc *gojay.Encoder) {
	r.commonStateResponse.MarshalJSONObject(enc)

	enc.AddArrayKey("tables", gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
		lastIdx := len(r.Tables) - 1
		for idx, table := range r.Tables {
			if err := enc.EncodeObject(table); err != nil {
				// the error should bubble up through the `gojay.Encoder`.
				return
			}
			if idx != lastIdx {
				enc.AppendByte(',')
			}
		}
	}))
}

func (r *getMultiTableRowsResponse) IsNil() bool { return r == nil }

func (r *commonStateResponse) MarshalJSONObject(enc *gojay.Encoder) {
	if r.UpToBlockID != "" {
		enc.AddStringKey("up_to_block_id", r.UpToBlockID)
		enc.AddIntKey("up_to_block_num", int(fluxdb.BlockNum(r.UpToBlockID)))
	}
	enc.AddStringKey("last_irreversible_block_id", r.LastIrreversibleBlockID)
	enc.AddIntKey("last_irreversible_block_num", int(fluxdb.BlockNum(r.LastIrreversibleBlockID)))
}

func (r *commonStateResponse) IsNil() bool { return r == nil }

func (r *getTableResponse) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddStringKey("account", r.Account)
	enc.AddStringKey("scope", r.Scope)

	r.readTableResponse.MarshalJSONObject(enc)
}

func (r *getTableResponse) IsNil() bool { return r == nil }

func (r *getTableRowResponse) MarshalJSONObject(enc *gojay.Encoder) {
	r.commonStateResponse.MarshalJSONObject(enc)

	if r.Row == nil {
		enc.AddNullKey("row")
	} else {
		enc.AddObjectKey("row", r.Row)
	}
}

func (r *getTableRowResponse) IsNil() bool { return r == nil }

func (r *readTableResponse) MarshalJSONObject(enc *gojay.Encoder) {
	if r.ABI != nil {
		d, _ := json.Marshal(r.ABI)
		j := gojay.EmbeddedJSON(d)

		enc.AddEmbeddedJSONKey("abi", &j)
	}

	enc.AddArrayKey("rows", gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
		lastIdx := len(r.Rows) - 1
		for idx, row := range r.Rows {
			if err := enc.EncodeObject(row); err != nil {
				// the error should bubble up through the `gojay.Encoder`.
				return
			}
			if idx != lastIdx {
				enc.AppendByte(',')
			}
		}
	}))
}

func (r *readTableResponse) IsNil() bool { return r == nil }

func (r *tableRow) MarshalJSONObject(enc *gojay.Encoder) {
	// TODO: check the `Data` type, depending on type:
	enc.AddStringKey("key", r.Key)

	if r.Payer != "" {
		enc.AddStringKey("payer", r.Payer)
	}

	switch v := r.Data.(type) {
	case []byte:
		enc.AddStringKey("hex", hex.EncodeToString(v))
	case *onTheFlyABISerializer:
		s := v

		jsonData, err := s.abi.DecodeTableRowTyped(s.tableTypeName, s.rowDataToDecode)
		if err != nil {
			// TRACK THIS..
			enc.AddStringKey("hex", hex.EncodeToString(s.rowDataToDecode))
			enc.AddStringKey("error", fmt.Sprintf("ABI from block %d, row struct %q, err: %s", s.abiAtBlockNum, s.tableTypeName, err))
		} else {
			jsonData := gojay.EmbeddedJSON(jsonData)
			enc.AddEmbeddedJSONKey("json", &jsonData)
		}
	}

	if r.BlockNum != 0 {
		enc.AddUint32Key("block", r.BlockNum)
	}
}

func (r *tableRow) IsNil() bool { return r == nil }
