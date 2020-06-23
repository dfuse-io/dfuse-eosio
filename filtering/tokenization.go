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

package filtering

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/search"
	"go.uber.org/zap"
)

// receiver account, account account, auth permission,
// --search-common-index-terms="receiver,account,auth,status,notif,input,data.from,data.to,data.bob,db.key,db.table,bob"

var fixedIndexedFields = []search.IndexedField{
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

var IndexedFields = []search.IndexedField{
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

func tokenizeEOSExecutedAction(actTrace *pbcodec.ActionTrace) (out map[string]interface{}) {
	out = make(map[string]interface{})
	out["receiver"] = actTrace.Receipt.Receiver
	out["account"] = actTrace.Account()
	out["action"] = actTrace.Name()
	out["auth"] = tokenizeEOSAuthority(actTrace.Action.Authorization)
	out["data"] = tokenizeEOSDataObject(actTrace.Action.JsonData)

	return out
}

func tokenizeEOSAuthority(authorizations []*pbcodec.PermissionLevel) (out []string) {
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
	for _, indexedField := range IndexedFields {
		if value, exists := jsonData[indexedField.Name]; exists {
			out[indexedField.Name] = value
		}
	}

	// FIXME: make sure this is HASHED
	hashKeys(jsonData, out, hashedEOSDataIndexedFields)

	// TODO: make sure we don't send strings that are more than 100 chars in the index..
	// some things put pixels in there.. if it matches the whitelist *bam* !

	return out
}

func tokenizeEvent(config eventsConfig, authKey string, data string) url.Values {
	out, err := url.ParseQuery(data)
	if err != nil {
		zlog.Debug("error parsing dfuse events 'data' field", zap.Error(err))
		return nil
	}

	isRestricted := !config.unrestricted

	for k, vals := range out {
		if isRestricted && len(k) > 16 {
			zlog.Debug("dfuse events field name too long", zap.String("field_prefix", k[:16]))
			return nil
		}

		if isRestricted {
			// For free keys, limit to 64 chars chars the key
			for _, v := range vals {
				if len(v) > 64 {
					zlog.Debug("dfuse events field value too long", zap.String("field", k), zap.Int("value_size", len(v)))
					return nil
				}
			}
		}
	}

	if isRestricted && len(out) > 3 {
		zlog.Debug("dfuse events has more than 3 fields", zap.Int("field_count", len(out)))
		return nil
	}

	return out
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
