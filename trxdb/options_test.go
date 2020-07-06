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
	"errors"
	"testing"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithIndexableRows(t *testing.T) {
	fullIndexing := map[pbtrxdb.IndexableRow]bool{
		pbtrxdb.IndexableRow_INDEXABLE_ROW_ACCOUNT:      true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_BLOCK:        true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_DTRX:         true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_IMPLICIT_TRX: true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_TRX:          true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_TRX_TRACE:    true,
		pbtrxdb.IndexableRow_INDEXABLE_ROW_TIMELINE:     true,
	}

	tests := []struct {
		name        string
		in          []string
		expected    map[pbtrxdb.IndexableRow]bool
		expectedErr error
	}{
		{
			"full when empty",
			nil,
			fullIndexing,
			nil,
		},
		{
			"full when single star element",
			[]string{"*"},
			fullIndexing,
			nil,
		},
		{
			"partial elements",
			[]string{"account", "block"},
			map[pbtrxdb.IndexableRow]bool{
				pbtrxdb.IndexableRow_INDEXABLE_ROW_ACCOUNT: true,
				pbtrxdb.IndexableRow_INDEXABLE_ROW_BLOCK:   true,
			},
			nil,
		},
		{
			"all elements",
			[]string{"account", "block", "dtrx", "implicit_trx", "timeline", "trx", "trx_trace"},
			fullIndexing,
			nil,
		},

		{
			"error when multiple with  star element",
			[]string{"account", "*"},
			nil,
			errors.New(`invalid value "*", valid values are "account, block, dtrx, implicit_trx, timeline, trx, trx_trace"`),
		},
		{
			"error when unknown element",
			[]string{"value"},
			nil,
			errors.New(`invalid value "value", valid values are "account, block, dtrx, implicit_trx, timeline, trx, trx_trace"`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			indexableRows := WithIndexableRows(test.in)
			actual, err := indexableRows.(IndexableRows).ToMap()

			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}
