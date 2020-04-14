// Copyright 2019 dfuse Platform Inc.
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

package search

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"

	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/dfuse-io/search"
	"go.uber.org/zap"
)

func init() {
	zlog.Info("initializing indexed fields cache")
	InitEOSIndexedFields()
}

var fixedEOSIndexedFields = []search.IndexedField{
	{"receiver", search.AccountType},
	{"account", search.AccountType},
	{"action", search.ActionType},
	{"auth", search.PermissionType},
	{"block_num", search.BlockNumType},
	{"trx_idx", search.TransactionIDType},
	{"scheduled", search.BooleanType},
	{"status", search.FreeFormType},
	{"notif", search.BooleanType},
	{"input", search.BooleanType},
	{"event", search.FreeFormType},
}

var EOSIndexedFields = []search.IndexedField{
	{"account", search.AccountType},
	{"active", search.FreeFormType},
	{"active_key", search.FreeFormType},
	{"actor", search.FreeFormType},
	{"amount", search.AssetType},
	{"auth", search.FreeFormType},
	{"authority", search.FreeFormType},
	{"bid", search.FreeFormType},
	{"bidder", search.AccountType},
	{"canceler", search.AccountType},
	{"creator", search.AccountType},
	{"executer", search.AccountType},
	{"from", search.AccountType},
	{"is_active", search.BooleanType},
	{"is_priv", search.BooleanType},
	{"isproxy", search.BooleanType},
	{"issuer", search.AccountType},
	{"level", search.FreeFormType},
	{"location", search.FreeFormType},
	{"maximum_supply", search.AssetType},
	{"name", search.NameType},
	{"newname", search.NameType},
	{"owner", search.AccountType},
	{"parent", search.AccountType},
	{"payer", search.AccountType},
	{"permission", search.PermissionType},
	{"producer", search.AccountType},
	{"producer_key", search.FreeFormType},
	{"proposal_name", search.NameType},
	{"proposal_hash", search.FreeFormType},
	{"proposer", search.AccountType},
	{"proxy", search.FreeFormType},
	{"public_key", search.FreeFormType},
	{"producers", search.FreeFormType},
	{"quant", search.FreeFormType},
	{"quantity", search.FreeFormType},
	{"ram_payer", search.AccountType},
	{"receiver", search.AccountType},
	{"requested", search.BooleanType},
	{"requirement", search.FreeFormType},
	{"symbol", search.FreeFormType},
	{"threshold", search.FreeFormType},
	{"to", search.AccountType},
	{"transfer", search.FreeFormType},
	{"voter", search.AccountType},
	{"voter_name", search.NameType},
	{"weight", search.FreeFormType},
}

//TODO: sha256 actual bytes (hex decode, etc.)
var hashedEOSDataIndexedFields = []search.IndexedField{
	{"abi", search.HexType},
	{"code", search.HexType}, // only for action = setcode
}

func tokenizeEOSExecutedAction(actTrace *pbdeos.ActionTrace) (out map[string]interface{}) {
	out = make(map[string]interface{})
	out["receiver"] = actTrace.Receipt.Receiver
	out["account"] = actTrace.Account()
	out["action"] = actTrace.Name()
	out["auth"] = tokenizeEOSAuthority(actTrace.Action.Authorization)
	out["data"] = tokenizeEOSDataObject(actTrace.Action.JsonData)

	return out
}

func tokenizeEOSAuthority(authorizations []*pbdeos.PermissionLevel) (out []string) {
	for _, auth := range authorizations {
		actor := auth.Actor
		perm := auth.Permission
		out = append(out, actor, fmt.Sprintf("%s@%s", actor, perm))
	}

	return
}

func tokenizeEOSDataObject(data string) map[string]interface{} {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return nil
	}

	out := make(map[string]interface{})
	for _, indexedField := range EOSIndexedFields {
		if value, exists := jsonData[indexedField.Name]; exists {
			out[indexedField.Name] = value
		}
	}

	hashKeys(jsonData, out, hashedEOSDataIndexedFields)

	// TODO: make sure we don't send strings that are more than 100 chars in the index..
	// some things put pixels in there.. if it matches the whitelist *bam* !

	return out
}

func tokenizeEvent(key string, data string) url.Values {
	out, err := url.ParseQuery(data)
	if err != nil {
		zlog.Debug("error parsing dfuseiohooks", zap.Error(err))
		return nil
	}

	var authKey bool

	for k, vals := range out {
		// 16 chars keys for everyone
		if len(k) > 16 {
			zlog.Debug("dfuse hooks event field name too long", zap.String("field_prefix", k[:16]))
			return nil
		}

		if !authKey {
			// For free keys, limit to 64 chars chars the key
			for _, v := range vals {
				if len(v) > 64 {
					zlog.Debug("dfuse hooks event field value too long", zap.String("field", k), zap.Int("value_size", len(v)))
					return nil
				}
			}
		}
	}

	if !authKey && len(out) > 3 {
		zlog.Debug("dfuse hooks event has more than 3 fields", zap.Int("field_count", len(out)))
		return nil
	}

	return out
}

var cachedEOSIndexedFields []*search.IndexedField
var cachedEOSIndexedFieldsMap map[string]*search.IndexedField

// InitIndexedFields initialize the list of indexed fields of the service
func InitEOSIndexedFields() {
	fields := make([]*search.IndexedField, 0, len(fixedEOSIndexedFields)+len(EOSIndexedFields)+len(hashedEOSDataIndexedFields))

	for _, field := range fixedEOSIndexedFields {
		fields = append(fields, &search.IndexedField{field.Name, field.ValueType})
	}

	for _, field := range EOSIndexedFields {
		fields = append(fields, &search.IndexedField{"data." + field.Name, field.ValueType})
	}

	for _, field := range hashedEOSDataIndexedFields {
		fields = append(fields, &search.IndexedField{"data." + field.Name, field.ValueType})
	}

	fields = append(fields,
		&search.IndexedField{"ram.consumed", search.FreeFormType},
		&search.IndexedField{"ram.released", search.FreeFormType},
	)

	fields = append(fields,
		&search.IndexedField{"db.table", search.FreeFormType},

		// Disabled so that if user complains, we can easily add it back. This should be
		// removed if we do not index `db.key` anymore.
		// &IndexedField{"db.key", search.FreeFormType},
	)

	// Let's cache the fields so we do not re-compute them everytime.
	cachedEOSIndexedFields = fields

	// Let's compute the fields map from the actual fields slice
	cachedEOSIndexedFieldsMap = map[string]*search.IndexedField{}
	for _, field := range cachedEOSIndexedFields {
		cachedEOSIndexedFieldsMap[field.Name] = field
	}
}

// GetIndexedFields returns the list of indexed fields of the service, from the
// cached list of indexed fields. Function `InitIndexedFields` must be called prior
// using this function.
func GetEOSIndexedFields() []*search.IndexedField {
	if cachedEOSIndexedFields == nil {
		zlog.Panic("the indexed fields cache is nil, you must initialize it prior calling this method")
	}

	return cachedEOSIndexedFields
}

func GetEOSIndexedFieldsMap() map[string]*search.IndexedField {
	if cachedEOSIndexedFieldsMap == nil {
		zlog.Panic("the indexed fields map cache is nil, you must initialize it prior calling this method")
	}

	return cachedEOSIndexedFieldsMap
}

func hashKeys(in, out map[string]interface{}, fields []search.IndexedField) {
	for _, field := range fields {
		f, found := in[field.Name]
		if !found {
			continue
		}

		val, ok := f.(string)
		if !ok {
			continue
		}

		res, err := hex.DecodeString(val)
		if err != nil {
			continue
		}

		h := sha256.New()
		_, _ = h.Write(res)
		out[field.Name] = hex.EncodeToString(h.Sum(nil))
	}
}
