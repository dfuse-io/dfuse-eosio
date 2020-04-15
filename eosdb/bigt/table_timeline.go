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

package bigt

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
)

type TimelineTable struct {
	*basebigt.BaseTable

	ColMetaExists string
}

func NewTimelineTable(name string, client *bigtable.Client) *TimelineTable {
	return &TimelineTable{
		BaseTable: basebigt.NewBaseTable(name, []string{"meta"}, client),

		ColMetaExists: "meta:exists",
	}
}

func (tbl *TimelineTable) ReadRows(ctx context.Context, rowRange bigtable.RowSet, opts ...bigtable.ReadOption) (out []string, err error) {
	opts = append(opts, bigtable.RowFilter(bigtable.FamilyFilter("meta")))
	err = tbl.BaseTable.ReadRows(ctx, rowRange, func(row bigtable.Row) bool {
		out = append(out, row.Key())
		return true
	}, opts...)
	if err != nil {
		return out, fmt.Errorf("read timeline rows: %s", err)
	}

	return
}

func (tbl *TimelineTable) PutMetaExists(key string) {
	tbl.SetKey(key, tbl.ColMetaExists, []byte(""))
}
