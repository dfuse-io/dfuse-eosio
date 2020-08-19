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
	"net/url"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

// This represents `data.` fields for which theirs value should be the hash of the content
var dataFieldsToHash = []string{"abi", "code"}

type tokenizer struct {
	indexedTerms *IndexedTerms
}

func (t *tokenizer) tokenize(actTrace *pbcodec.ActionTrace) (out map[string]interface{}) {
	out = make(map[string]interface{})

	if t.indexedTerms.Receiver {
		out["receiver"] = actTrace.Receipt.Receiver
	}

	if t.indexedTerms.Account {
		out["account"] = actTrace.Account()
	}

	if t.indexedTerms.Action {
		out["action"] = actTrace.Name()
	}

	if t.indexedTerms.Auth {
		tokens := t.tokenizeAuthority(actTrace.Action.Authorization)
		if len(tokens) > 0 {
			out["auth"] = tokens
		}
	}

	if len(t.indexedTerms.Data) > 0 {
		tokens := t.tokenizeData(actTrace.Action.JsonData)
		if len(tokens) > 0 {
			out["data"] = tokens
		}
	}

	return out
}

func (t *tokenizer) tokenizeAuthority(authorizations []*pbcodec.PermissionLevel) (out []string) {
	if len(authorizations) <= 0 {
		return nil
	}

	out = make([]string, len(authorizations)*2)
	for i, auth := range authorizations {
		out[i*2] = auth.Actor
		out[i*2+1] = auth.Authorization()
	}

	return
}

func (t *tokenizer) tokenizeData(data string) map[string]interface{} {
	if data == "" {
		return nil
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return nil
	}

	out := make(map[string]interface{})
	for dataFieldName, dataFieldValue := range jsonData {
		if t.indexedTerms.IsIndexed("data." + dataFieldName) {
			normalizedField := t.indexedTerms.NormalizeDataField(dataFieldName)
			normalizedValue, skipField := normalizeDataValue(normalizedField, dataFieldValue)
			if skipField {
				continue
			}

			out[normalizedField] = normalizedValue
		}
	}

	return out
}

func (t *tokenizer) tokenizeEvent(config eventsConfig, authKey string, data string) url.Values {
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

func normalizeDataValue(name string, value interface{}) (normalized interface{}, skipField bool) {
	if isDataFieldToHash(name) {
		val, ok := value.(string)
		if !ok {
			skipField = true
			return
		}

		bytes, err := hex.DecodeString(val)
		if err != nil {
			skipField = true
			return
		}

		h := sha256.New()
		_, _ = h.Write(bytes)
		return hex.EncodeToString(h.Sum(nil)), false
	}

	return value, false
}

func isDataFieldToHash(name string) bool {
	for _, fieldToHash := range dataFieldsToHash {
		if fieldToHash == name {
			return true
		}
	}

	return false
}
