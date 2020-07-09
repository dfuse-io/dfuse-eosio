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

package search

import (
	"fmt"
	"regexp"
	"strings"
)

type IndexedTerms struct {
	Receiver    bool
	Account     bool
	Action      bool
	Auth        bool
	Scheduled   bool
	Status      bool
	Notif       bool
	Input       bool
	Event       bool
	RAMConsumed bool
	RAMReleased bool
	DBTable     bool
	DBKey       bool

	Base map[string]bool
	Data map[string]bool
}

type fieldCategory int

const (
	fieldCategoryBase fieldCategory = iota
	fieldCategoryData
)

var splitTermRegexp = regexp.MustCompile("(,|\\s+)")

// DefaultIndexedTerms holds terms to index from last hosted dfuse infra
const DefaultIndexedTerms = "receiver, account, action, auth, scheduled, status, notif, input, event, ram.consumed, ram.released, db.key, db.table, data.account, data.active, data.active_key, data.actor, data.amount, data.auth, data.authority, data.bid, data.bidder, data.canceler, data.creator, data.executer, data.from, data.is_active, data.is_priv, data.isproxy, data.issuer, data.level, data.location, data.maximum_supply, data.name, data.newname, data.owner, data.parent, data.payer, data.permission, data.producer, data.producer_key, data.proposal_name, data.proposal_hash, data.proposer, data.proxy, data.public_key, data.producers, data.quant, data.quantity, data.ram_payer, data.receiver, data.requested, data.requirement, data.symbol, data.threshold, data.to, data.transfer, data.voter, data.voter_name, data.weight, data.abi, data.code"

func NewIndexedTerms(specs string) (out *IndexedTerms, err error) {
	if specs == "*" || specs == "" {
		specs = DefaultIndexedTerms
	}

	out = &IndexedTerms{
		Base: map[string]bool{},
		Data: map[string]bool{},
	}

	terms := splitTermRegexp.Split(specs, -1)
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}

		category := fieldCategoryBase

		switch term {
		case "receiver":
			out.Receiver = true
		case "account":
			out.Account = true
		case "action":
			out.Action = true
		case "auth":
			out.Auth = true
		case "scheduled":
			out.Scheduled = true
		case "status":
			out.Status = true
		case "notif":
			out.Notif = true
		case "input":
			out.Input = true
		case "event":
			out.Event = true
		case "ram.consumed":
			out.RAMConsumed = true
		case "ram.released":
			out.RAMReleased = true
		case "db.table":
			out.DBTable = true
		case "db.key":
			out.DBKey = true
		default:
			if strings.HasPrefix(term, "data.") {
				category = fieldCategoryData
				out.Data[out.NormalizeDataField(term)] = true
			} else {
				return nil, fmt.Errorf("invalid indexed term specs %q: unknown field %q", specs, term)
			}
		}

		if category == fieldCategoryBase {
			out.Base[term] = true
		}
	}

	return out, nil
}

func (t *IndexedTerms) IsIndexed(fieldName string) bool {
	if t.Base[fieldName] {
		return true
	}

	if strings.HasPrefix(fieldName, "data.") {
		return t.Data[t.NormalizeDataField(fieldName)]
	}

	return strings.HasPrefix(fieldName, "event.")
}

// NormalizeDataTerm extracts the the first child element from the data name (i.e. from `data.first.second.third`
// to `first` where `data` is the parent name, `first.second.third` is the child hierarchy and `first`
// is the first child element of `data`).
func (t *IndexedTerms) NormalizeDataField(fieldName string) string {
	segments := strings.Split(fieldName, ".")
	if len(segments) < 2 {
		return fieldName
	}

	return segments[1]
}
