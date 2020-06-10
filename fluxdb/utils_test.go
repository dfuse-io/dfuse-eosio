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
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"cloud.google.com/go/bigtable/bttest"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/kv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplodeRowKey(t *testing.T) {
	type expected struct {
		tableKey string
		blockNum uint32
		primKey  string
		err      error
	}

	tests := []struct {
		name     string
		rowKey   string
		expected expected
	}{
		{
			"auth_link_row",
			"al:0000000000000003:00000004:0000000000000001:0000000000000002",
			expected{"al:0000000000000003", 4, "0000000000000001:0000000000000002", nil},
		},
		{
			"auth_link_row/wrong_part_count",
			"al:0000000000000007",
			expected{err: errors.New("auth link row key should have 5 parts, got 2")},
		},
		{
			"auth_link_row/wrong_block_num_length",
			"al:0000000000000003:0000004:0000000000000001:0000000000000002",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"auth_link_row/wrong_block_num",
			"al:0000000000000003:0000000G:0000000000000001:0000000000000002",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},

		{
			"account_resource_limit",
			"arl:eosio:00000004:limits",
			expected{"arl:eosio", 4, "limits", nil},
		},
		{
			"account_resource_limit/wrong_part_count",
			"arl:limits",
			expected{err: errors.New("account resource limit row key should have 4 parts, got 2")},
		},
		{
			"account_resource_limit/wrong_block_num_length",
			"arl:eosio:0000004:limits",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"account_resource_limit/wrong_block_num",
			"arl:eosio:0000000G:limits",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},

		{
			"block_resource_limit",
			"brl:00000004:config",
			expected{"brl", 4, "config", nil},
		},
		{
			"block_resource_limit/wrong_part_count",
			"brl:config",
			expected{err: errors.New("block resource limit row key should have 3 parts, got 2")},
		},
		{
			"block_resource_limit/wrong_block_num_length",
			"brl:0000004:config",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"block_resource_limit/wrong_block_num",
			"brl:0000000G:config",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},

		{
			"key_account",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP:00000004:0000000000000005:0000000000000006",
			expected{"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP", 4, "0000000000000005:0000000000000006", nil},
		},
		{
			"key_account/wrong_part_count",
			"ka2:0000000000000007",
			expected{err: errors.New("key account row key should have 5 parts, got 2")},
		},
		{
			"key_account/wrong_block_num_length",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP:0000004:0000000000000005:0000000000000006",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"key_account/wrong_block_num",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP:0000000G:0000000000000005:0000000000000006",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},

		{
			"table_data",
			"td:0000000000000001:0000000000000002:0000000000000003:00000004:0000000000000005",
			expected{"td:0000000000000001:0000000000000002:0000000000000003", 4, "0000000000000005", nil},
		},
		{
			"table_data/wrong_part_count",
			"td:0000000000000007",
			expected{err: errors.New("table data row key should have 6 parts, got 2")},
		},
		{
			"table_data/wrong_block_num_length",
			"td:0000000000000001:0000000000000002:0000000000000003:0000004:0000000000000005",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"table_data/wrong_block_num",
			"td:0000000000000001:0000000000000002:0000000000000003:0000000G:0000000000000005",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},

		{
			"table_scope",
			"ts:0000000000000001:0000000000000002:00000004:0000000000000005",
			expected{"ts:0000000000000001:0000000000000002", 4, "0000000000000005", nil},
		},
		{
			"table_scope/wrong_part_count",
			"ts:0000000000000007",
			expected{err: errors.New("table scope row key should have 5 parts, got 2")},
		},
		{
			"table_scope/wrong_block_num_length",
			"ts:0000000000000001:0000000000000002:0000004:0000000000000005",
			expected{err: errors.New("block chunk should have length of 8")},
		},
		{
			"table_scope/wrong_block_num",
			"ts:0000000000000001:0000000000000002:0000000G:0000000000000005",
			expected{err: &strconv.NumError{Func: "ParseUint", Num: "0000000G", Err: errors.New("invalid syntax")}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tableKey, blockNum, primKey, err := explodeWritableRowKey(test.rowKey)

			require.Equal(t, test.expected.err, err)

			if test.expected.err == nil {
				assert.Equal(t, test.expected.tableKey, tableKey)
				assert.Equal(t, test.expected.blockNum, blockNum)
				assert.Equal(t, test.expected.primKey, primKey)
			}
		})
	}
}

func TestChunking(t *testing.T) {
	row := mustCreateContractStateTabletRow("eosio", "eoscanadcom", "account", 1, "........ehbo5", "payer", nil, false)
	assert.Equal(t,
		"td:0000000000000000:0000000000000000:0000000000000000:00000001:0000000000000400",
		row.Key())

	primKey, ok := chunkKeyUint64(row.Key(), 5)
	assert.Equal(t, primKey, row.PrimaryKey())
	assert.True(t, ok)
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
	srv, err := bttest.NewServer("localhost:0")
	require.NoError(t, err)

	kvStore, err := kv.NewStore(context.Background(), "bigkv://dev.dev/test?createTables=true")
	require.NoError(t, err)

	db := New(kvStore)
	closer := func() {
		srv.Close()
		db.Close()
	}

	return db, closer
}
