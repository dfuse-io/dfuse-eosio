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

package trxdb

import (
	"fmt"
	"regexp"
	"strings"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
)

var NoIndexing IndexableCategories = nil
var FullIndexing IndexableCategories

func init() {
	for key := range pbtrxdb.IndexableCategory_name {
		FullIndexing = append(FullIndexing, pbtrxdb.IndexableCategory(key))
	}

	return
}

var splitCategoriesRegex = regexp.MustCompile("(,|\\s+|\\|)")

type IndexableCategories []pbtrxdb.IndexableCategory

func NewIndexableCategories(in string) (out IndexableCategories, err error) {
	normalizedIn := strings.TrimSpace(in)
	if normalizedIn == "" || normalizedIn == "*" {
		return FullIndexing, nil
	}

	if normalizedIn == "-" {
		return NoIndexing, nil
	}

	for _, rawCategory := range splitCategoriesRegex.Split(in, -1) {
		rawCategory = strings.TrimSpace(rawCategory)
		if rawCategory == "" {
			continue
		}

		category, err := toIndexableCategory(rawCategory)
		if err != nil {
			return nil, err
		}

		out = append(out, category)
	}

	return out, nil
}

func (i IndexableCategories) ToMap() map[pbtrxdb.IndexableCategory]bool {
	out := map[pbtrxdb.IndexableCategory]bool{}
	for key := range pbtrxdb.IndexableCategory_name {
		out[pbtrxdb.IndexableCategory(key)] = true
	}

	return out
}

func (i IndexableCategories) AsHumanKeys() (out []string) {
	for _, category := range i {
		out = append(out, category.HumanKey())
	}
	return
}

func toIndexableCategory(in string) (pbtrxdb.IndexableCategory, error) {
	value, found := pbtrxdb.IndexableCategoryFromHumanKey(in)
	if !found {
		return 0, fmt.Errorf("invalid value %q, valid values are %q", in, strings.Join(pbtrxdb.IndexableCategoryHumanKeys(), ", "))
	}

	return pbtrxdb.IndexableCategory(value), nil
}
