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

package dgraphql

import (
	"testing"

	types "github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	"github.com/dfuse-io/dgraphql"
	pbgraphql "github.com/dfuse-io/pbgo/dfuse/graphql/v1"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaginator(t *testing.T) {
	tests := []struct {
		name            string
		firstReq        *types.Uint32
		lastReq         *types.Uint32
		before          *string
		after           *string
		limit           uint32
		cursorFactory   func() proto.Message
		expectPaginator *Paginator
		expectError     bool
	}{
		{
			name: "simple paginator",
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     0,
				last:      0,
			},
		},
		{
			name:     "paginator with first without limit ",
			firstReq: getUint32(10),
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     10,
				last:      0,
			},
		},
		{
			name:     "paginator with first below limit ",
			firstReq: getUint32(10),
			limit:    20,
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     10,
				last:      0,
			},
		},
		{
			name:     "paginator with first at limit ",
			firstReq: getUint32(20),
			limit:    20,
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     20,
				last:      0,
			},
		},
		{
			name:        "paginator with first greater then limit ",
			firstReq:    getUint32(30),
			limit:       20,
			expectError: true,
		},
		{
			name:    "paginator with last below limit ",
			lastReq: getUint32(10),
			limit:   20,
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     0,
				last:      10,
			},
		},
		{
			name:    "paginator with last at limit ",
			lastReq: getUint32(20),
			limit:   20,
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     0,
				last:      20,
			},
		},
		{
			name:        "paginator with last greater limit ",
			lastReq:     getUint32(30),
			limit:       20,
			expectError: true,
		},
		{
			name:        "paginator with first & last ",
			firstReq:    getUint32(20),
			lastReq:     getUint32(30),
			limit:       20,
			expectError: true,
		},
		{
			name:  "paginator without first & last with limit",
			limit: 20,
			expectPaginator: &Paginator{
				beforeKey: "",
				afterKey:  "",
				first:     20,
				last:      0,
			},
		},
		{
			name: "paginator with valid before cursor ",
			before: s(dgraphql.MustProtoToOpaqueCursor(&pbgraphql.TransactionCursor{
				Ver:             1,
				TransactionHash: "abababab",
			}, "test_transaction_cursor")),
			cursorFactory: func() proto.Message {
				return &pbgraphql.TransactionCursor{}
			},
			expectPaginator: &Paginator{
				beforeKey: "abababab",
			},
		},
		{
			name: "paginator with valid before and after cursor ",
			before: s(dgraphql.MustProtoToOpaqueCursor(&pbgraphql.TransactionCursor{
				Ver:             1,
				TransactionHash: "abababab",
			}, "test_transaction_cursor")),
			after: s(dgraphql.MustProtoToOpaqueCursor(&pbgraphql.TransactionCursor{
				Ver:             1,
				TransactionHash: "cdcdcdcd",
			}, "test_transaction_cursor")),
			cursorFactory: func() proto.Message {
				return &pbgraphql.TransactionCursor{}
			},
			expectPaginator: &Paginator{
				beforeKey: "abababab",
				afterKey:  "cdcdcdcd",
			},
		},
		{
			name:   "paginator with in-valid before cursor ",
			before: s("adsf"),
			cursorFactory: func() proto.Message {
				return &pbgraphql.TransactionCursor{}
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paginator, err := NewPaginator(test.firstReq, test.lastReq, test.before, test.after, test.limit, test.cursorFactory)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectPaginator, paginator)
			}
		})
	}
}

func TestPaginateWitoutLimit(t *testing.T) {
	tests := []struct {
		name           string
		results        PagineableStrings
		beforeCursor   string
		afterCursor    string
		expectElements PagineableStrings
	}{
		{
			name:           "results without any cursor or first & last",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			expectElements: PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
		},
		{
			name:           "results with a before cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "ccc",
			expectElements: PagineableStrings([]string{"aaa", "bbb"}),
		},
		{
			name:           "results with an after cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:    "bbb",
			expectElements: PagineableStrings([]string{"ccc", "ddd"}),
		},
		{
			name:           "results with a before and after cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "bbb",
			afterCursor:    "bbb",
			expectElements: PagineableStrings([]string{}),
		},
		{
			name:           "results with a before and after cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "bbb",
			afterCursor:    "ccc",
			expectElements: PagineableStrings([]string{}),
		},
		{
			name:           "results with a before and after cursor inverted",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "ddd",
			afterCursor:    "aaa",
			expectElements: PagineableStrings([]string{"bbb", "ccc"}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Paginator{
				beforeKey: test.beforeCursor,
				afterKey:  test.afterCursor,
				first:     0,
				last:      0,
			}
			newCollection := p.Paginate(test.results)
			assert.ElementsMatch(t, test.expectElements, newCollection)
		})
	}
}

func TestPaginateWithLimit(t *testing.T) {
	tests := []struct {
		name           string
		results        PagineableStrings
		beforeCursor   string
		afterCursor    string
		first          uint32
		last           uint32
		expectElements PagineableStrings
	}{
		{
			name:           "results with first and without any cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			first:          2,
			expectElements: PagineableStrings([]string{"aaa", "bbb"}),
		},
		{
			name:           "results with last and without any cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			last:           2,
			expectElements: PagineableStrings([]string{"ccc", "ddd"}),
		},
		{
			name:           "results with first & last and without any cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			first:          2,
			last:           2,
			expectElements: PagineableStrings([]string{"aaa", "bbb"}),
		},
		{
			name:           "results with first & before cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "ddd",
			first:          2,
			expectElements: PagineableStrings([]string{"aaa", "bbb"}),
		},
		{
			name:           "results with last & before cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "ddd",
			last:           2,
			expectElements: PagineableStrings([]string{"bbb", "ccc"}),
		},
		{
			name:           "results with a first and an after cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:    "aaa",
			first:          2,
			expectElements: PagineableStrings([]string{"bbb", "ccc"}),
		},
		{
			name:           "results with a last and an after cursor",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:    "aaa",
			last:           2,
			expectElements: PagineableStrings([]string{"ccc", "ddd"}),
		},
		{
			name:           "results with a before and after cursor and a first",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "bbb",
			afterCursor:    "bbb",
			first:          2,
			expectElements: PagineableStrings([]string{}),
		},
		{
			name:           "results with a before and after cursor and a last",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:   "bbb",
			afterCursor:    "bbb",
			last:           2,
			expectElements: PagineableStrings([]string{}),
		},
		{
			name:           "results with a before and after cursor inverted with first",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd", "eee"}),
			beforeCursor:   "eee",
			afterCursor:    "aaa",
			first:          2,
			expectElements: PagineableStrings([]string{"bbb", "ccc"}),
		},
		{
			name:           "results with a before and after cursor inverted with last",
			results:        PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd", "eee"}),
			beforeCursor:   "eee",
			afterCursor:    "aaa",
			last:           2,
			expectElements: PagineableStrings([]string{"ccc", "ddd"}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Paginator{
				beforeKey: test.beforeCursor,
				afterKey:  test.afterCursor,
				first:     test.first,
				last:      test.last,
			}
			newCollection := p.Paginate(test.results)
			assert.ElementsMatch(t, test.expectElements, newCollection)
		})
	}
}

func TestPaginateNextAndPreviousPage(t *testing.T) {
	tests := []struct {
		name                  string
		results               PagineableStrings
		beforeCursor          string
		afterCursor           string
		first                 uint32
		last                  uint32
		expectHasNextpage     bool
		expectHasPreviouspage bool
	}{
		{
			name:                  "results with first and without any cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			first:                 2,
			expectHasNextpage:     true,
			expectHasPreviouspage: false,
		},
		{
			name:                  "results with last and without any cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			last:                  2,
			expectHasNextpage:     false,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with first & last and without any cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			first:                 2,
			last:                  2,
			expectHasNextpage:     true,
			expectHasPreviouspage: false,
		},
		{
			name:                  "results with first & before cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "ddd",
			first:                 2,
			expectHasNextpage:     true,
			expectHasPreviouspage: false,
		},
		{
			name:                  "results with last & before cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "ddd",
			last:                  2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a first and an after cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:           "aaa",
			first:                 2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a last and an after cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:           "aaa",
			last:                  2,
			expectHasNextpage:     false,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor and a first",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "bbb",
			afterCursor:           "bbb",
			first:                 2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor and a last",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "bbb",
			afterCursor:           "bbb",
			last:                  2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor inverted with first",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd", "eee"}),
			beforeCursor:          "eee",
			afterCursor:           "aaa",
			first:                 2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor inverted with last",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd", "eee"}),
			beforeCursor:          "eee",
			afterCursor:           "aaa",
			last:                  2,
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results without any cursor or first & last",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			expectHasNextpage:     false,
			expectHasPreviouspage: false,
		},
		{
			name:                  "results with a before cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "ccc",
			expectHasNextpage:     true,
			expectHasPreviouspage: false,
		},
		{
			name:                  "results with an after cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			afterCursor:           "bbb",
			expectHasNextpage:     false,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "bbb",
			afterCursor:           "bbb",
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "bbb",
			afterCursor:           "ccc",
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
		{
			name:                  "results with a before and after cursor inverted",
			results:               PagineableStrings([]string{"aaa", "bbb", "ccc", "ddd"}),
			beforeCursor:          "ddd",
			afterCursor:           "aaa",
			expectHasNextpage:     true,
			expectHasPreviouspage: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Paginator{
				beforeKey: test.beforeCursor,
				afterKey:  test.afterCursor,
				first:     test.first,
				last:      test.last,
			}
			p.Paginate(test.results)
			assert.Equal(t, test.expectHasNextpage, p.HasNextPage)
			assert.Equal(t, test.expectHasPreviouspage, p.HasPreviousPage)
		})
	}
}

func getUint32(num uint32) *types.Uint32 {
	n := types.Uint32(num)
	return &n
}

func s(str string) *string {
	return &str
}
