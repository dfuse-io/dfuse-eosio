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

package fluxdb

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/kv"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunking(t *testing.T) {
	primaryKey := "........ehbo5"
	row := mustCreateContractStateTabletRow("eosio", "eoscanadacom", "account", 1, primaryKey, "payer", nil, false)
	assert.Equal(t,
		"cst/eosio:eoscanadacom:account/00000001/........ehbo5",
		row.Key())

	assert.Equal(t, primaryKey, row.PrimaryKey())
}

func Test_chunkKeyRevBlockNum(t *testing.T) {
	tests := []struct {
		key           string
		prefixKey     string
		expected      uint32
		expectedError error
	}{
		{"ts:0000:ffffebd1", "ts:0000:", 5166, nil},
		{"ts:0000:ffffebd1:00000000:000000", "ts:0000:", 5166, nil},

		{"ta:0000:ffffebd", "ts:0000:", 0, errors.New("key ta:0000:ffffebd should start with prefix key ts:0000:")},
		{"ts:0000:ffffebd", "ts:0000:", 0, errors.New("key ts:0000:ffffebd is too small too contains block num, should have at least 8 characters more than prefix")},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual, err := chunkKeyRevBlockNum(test.key, test.prefixKey)
			if test.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}

func NewTestDB(t *testing.T) (*FluxDB, func()) {
	kvStore, err := kv.NewStore("badger:///tmp/dfuse-test-badger?createTables=true")
	require.NoError(t, err)

	db := New(kvStore)
	closer := func() {
		db.Close()
	}

	return db, closer
}
