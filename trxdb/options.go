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
	"sort"
	"strings"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
)

var FullIndexing = map[pbtrxdb.IndexableRow]bool{}
var ValidIndexingRowKeys []string

func init() {
	for key := range pbtrxdb.IndexableRow_name {
		FullIndexing[pbtrxdb.IndexableRow(key)] = true
	}

	for key := range pbtrxdb.IndexableRow_value {
		ValidIndexingRowKeys = append(ValidIndexingRowKeys, strings.ToLower(strings.Replace(key, "INDEXABLE_ROW_", "", 1)))
	}
	sort.Sort(sort.StringSlice(ValidIndexingRowKeys))

	return
}

type Option interface {
	trxDBOption()
}

type IndexableRows []string

func WithIndexableRows(in []string) Option {
	return IndexableRows(in)
}

func (i IndexableRows) trxDBOption() {}

func (i IndexableRows) ToMap() (out map[pbtrxdb.IndexableRow]bool, err error) {
	if len(i) == 0 || len(i) == 1 && i[0] == "*" {
		return FullIndexing, nil
	}

	out = map[pbtrxdb.IndexableRow]bool{}
	for _, in := range i {
		value, err := i.toIndexableRow(in)
		if err != nil {
			return nil, err
		}

		out[value] = true
	}

	return
}

func (i IndexableRows) toIndexableRow(in string) (pbtrxdb.IndexableRow, error) {
	value, found := pbtrxdb.IndexableRow_value["INDEXABLE_ROW_"+strings.ToUpper(in)]
	if !found {
		return 0, fmt.Errorf("invalid indexable row value %q, valid values are %q", in, strings.Join(ValidIndexingRowKeys, ", "))
	}

	return pbtrxdb.IndexableRow(value), nil
}
