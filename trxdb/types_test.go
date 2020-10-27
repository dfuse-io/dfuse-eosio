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

func TestNewIndexableCategories(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    IndexableCategories
		expectedErr error
	}{
		{
			"no indexing when caret element",
			"-",
			NoIndexing,
			nil,
		},
		{
			"full when empty",
			"",
			FullIndexing,
			nil,
		},
		{
			"full when single star element",
			"*",
			FullIndexing,
			nil,
		},
		{
			"partial elements",
			"account, block",
			[]pbtrxdb.IndexableCategory{
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT,
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK,
			},
			nil,
		},
		{
			"all elements",
			"account, block, timeline, transaction",
			[]pbtrxdb.IndexableCategory{
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT,
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK,
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE,
				pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION,
			},
			nil,
		},

		{
			"error when multiple with star element",
			"account, *",
			nil,
			errors.New(`invalid value "*", valid values are "account, block, timeline, transaction"`),
		},
		{
			"error when unknown element",
			"value",
			nil,
			errors.New(`invalid value "value", valid values are "account, block, timeline, transaction"`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := NewIndexableCategories(test.in)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				assert.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}
