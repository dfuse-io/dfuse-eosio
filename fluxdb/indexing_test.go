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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTableIndexFromBinary(t *testing.T) {
	type expected struct {
		tableIndex *TableIndex
		err        error
	}

	tests := []struct {
		name       string
		tableKey   string
		atBlockNum uint32
		buffer     []byte
		expected   expected
	}{
		{
			"auth_link_valid",
			"al:0000000000000009",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x02, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  2,
				Map: map[string]uint32{
					"0000000000000002:0000000000000003": 4,
				},
			}, nil},
		},
		{
			"account_resource_limit_valid",
			"arl:eosio",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x01, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  1,
				Map: map[string]uint32{
					"00": 4,
				},
			}, nil},
		},
		{
			"block_resource_limit_valid",
			"brl",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x02, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x01, 0x00, 0x00, 0x00, 0x03, // Table row mapping 1
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  2,
				Map: map[string]uint32{
					"01": 3,
				},
			}, nil},
		},
		{
			"key_account_valid",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x02, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  2,
				Map: map[string]uint32{
					"0000000000000002:0000000000000003": 4,
				},
			}, nil},
		},
		{
			"key_account_valid/empty",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP",
			0,
			[]byte{
				0x00, 0x00, 0x00, 0x00, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
			},
			expected{&TableIndex{
				AtBlockNum: 0,
				Squelched:  0,
				Map:        map[string]uint32{},
			}, nil},
		},
		{
			"table_data_valid",
			"td:0000000000000009:0000000000000008:0000000000000007",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x02, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x05, // Table row mapping 2
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  2,
				Map: map[string]uint32{
					"0000000000000002": 4,
					"0000000000000003": 5,
				},
			}, nil},
		},
		{
			"table_scope_valid",
			"ts:0000000000000009:0000000000000008",
			6,
			[]byte{
				0x00, 0x00, 0x00, 0x02, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x05, // Table row mapping 2
			},
			expected{&TableIndex{
				AtBlockNum: 6,
				Squelched:  2,
				Map: map[string]uint32{
					"0000000000000002": 4,
					"0000000000000003": 5,
				},
			}, nil},
		},

		{
			"auth_link_misalign",
			"al:0000000000000009",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 20 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},
		{
			"account_resource_limit_misalign",
			"arl:eosio",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 5 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},
		{
			"block_resource_limit_misalign",
			"brl",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 5 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},
		{
			"key_account_misalign",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 20 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},
		{
			"table_data_misalign",
			"td:0000000000000009:0000000000000008:0000000000000007",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 12 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},
		{
			"table_scope_misalign",
			"ts:0000000000000009:0000000000000008",
			0,
			[]byte{0x00},
			expected{nil, errors.New("unable to unmarshal table index: 12 bytes alignment + 16 bytes metadata is off (has 1 bytes)")},
		},

		{
			"unkown_table_key_prefix",
			"not_there:0000000000000009",
			0,
			[]byte{},
			expected{nil, errors.New(`unknown primary key byte count for table key "not_there:0000000000000009"`)},
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tableIndex, err := NewTableIndexFromBinary(ctx, test.tableKey, test.atBlockNum, test.buffer)

			require.Equal(t, test.expected.err, err)
			if test.expected.err == nil {
				assert.Equal(t, test.expected.tableIndex, tableIndex)
			}
		})
	}
}

func TestTableIndexMarshalBinary(t *testing.T) {
	type expected struct {
		bytes []byte
		err   error
	}

	tests := []struct {
		name       string
		tableKey   string
		tableIndex *TableIndex
		expected   expected
	}{
		{
			"auth_link",
			"al:0000000000000009",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  1,
				Map: map[string]uint32{
					"0000000000000002:0000000000000003": 4,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x01, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			}, nil},
		},
		{
			"account_resource_limit",
			"arl:eosio",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  1,
				Map: map[string]uint32{
					"01": 4,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x01, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x01, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			}, nil},
		},
		{
			"block_resource_limit",
			"brl",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  1,
				Map: map[string]uint32{
					"00": 4,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x01, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			}, nil},
		},
		{
			"key_account",
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  4,
				Map: map[string]uint32{
					"0000000000000002:0000000000000003": 4,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x04, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			}, nil},
		},
		{
			"table_data",
			"td:0000000000000008:0000000000000007:0000000000000009",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  4,
				Map: map[string]uint32{
					"0000000000000002": 4,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x04, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, // Table row mapping 1
			}, nil},
		},
		{
			"table_scope",
			"ts:0000000000000008:0000000000000007",
			&TableIndex{
				AtBlockNum: 2,
				Squelched:  4,
				Map: map[string]uint32{
					"0000000000000002": 1,
				},
			},
			expected{[]byte{
				0x00, 0x00, 0x00, 0x04, // Squelched count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Reserved
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, // Table row mapping 1
			}, nil},
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bytes, err := test.tableIndex.MarshalBinary(ctx, test.tableKey)

			require.Equal(t, test.expected.err, err)
			if test.expected.err == nil {
				assert.Equal(t, test.expected.bytes, bytes)
			}
		})
	}
}

func TestIndexPrimaryKeyReader(t *testing.T) {
	type expected struct {
		primaryKey string
		err        error
	}

	tests := []struct {
		name             string
		primaryKeyReader indexPrimaryKeyReader
		buffer           []byte
		expected         expected
	}{
		{
			"auth_link_valid",
			authLinkIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			expected{"0000000000000001:0000000000000002", nil},
		},
		{
			"auth_link_missing_bytes",
			authLinkIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected{"", errors.New("auth link primary key reader: not enough bytes to read, 7 bytes left, wants 16")},
		},

		{
			"account_resource_limit_valid",
			accountResourceLimitIndexPrimaryKeyReader,
			[]byte{0x01},
			expected{"01", nil},
		},
		{
			"account_resource_limit_bytes",
			accountResourceLimitIndexPrimaryKeyReader,
			[]byte{},
			expected{"", errors.New("account resource limit primary key reader: not enough bytes to read, 0 bytes left, wants 1")},
		},

		{
			"block_resource_limit_valid",
			blockResourceLimitIndexPrimaryKeyReader,
			[]byte{0x00},
			expected{"00", nil},
		},
		{
			"block_resource_limit_bytes",
			blockResourceLimitIndexPrimaryKeyReader,
			[]byte{},
			expected{"", errors.New("block resource limit primary key reader: not enough bytes to read, 0 bytes left, wants 1")},
		},

		{
			"key_account_valid",
			keyAccountIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			expected{"0000000000000001:0000000000000002", nil},
		},
		{
			"key_account_missing_bytes",
			keyAccountIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected{"", errors.New("key account primary key reader: not enough bytes to read, 7 bytes left, wants 16")},
		},

		{
			"table_row_valid",
			tableDataIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected{"0000000000000001", nil},
		},
		{
			"table_row_missing_bytes",
			tableDataIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected{"", errors.New("table data primary key reader: not enough bytes to read uint64, 7 bytes left, wants 8")},
		},

		{
			"table_scope_valid",
			tableScopeIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected{"0000000000000001", nil},
		},
		{
			"table_scope_missing_bytes",
			tableScopeIndexPrimaryKeyReader,
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected{"", errors.New("table scope primary key reader: not enough bytes to read uint64, 7 bytes left, wants 8")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			primaryKey, err := test.primaryKeyReader(test.buffer)

			if test.expected.err == nil {
				assert.Equal(t, test.expected.primaryKey, primaryKey)
			} else {
				require.NotNil(t, err)
				require.Equal(t, test.expected.err.Error(), err.Error())
			}
		})
	}
}

func TestIndexPrimaryKeyWriter(t *testing.T) {
	type expected struct {
		buffer []byte
		err    error
	}

	tests := []struct {
		name             string
		primaryKeyWriter indexPrimaryKeyWriter
		primaryKey       string
		buffer           []byte
		expected         expected
	}{
		{
			"auth_link_valid",
			authLinkIndexPrimaryKeyWriter,
			"0000000000000001:0000000000000002",
			make([]byte, 18),
			expected{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00}, nil},
		},
		{
			"auth_link_wrong_chunk_count",
			authLinkIndexPrimaryKeyWriter,
			"0000000000000001",
			nil,
			expected{nil, errors.New("auth link primary key should have 2 chunks, got 1")},
		},
		{
			"auth_link_invalid_chunk_1",
			authLinkIndexPrimaryKeyWriter,
			"000000000000000G:0000000000000001",
			nil,
			expected{nil, errors.New(`auth link primary key writer, chunk #1: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},
		{
			"auth_link_invalid_chunk_2",
			authLinkIndexPrimaryKeyWriter,
			"0000000000000001:000000000000000G",
			make([]byte, 8),
			expected{nil, errors.New(`auth link primary key writer, chunk #2: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},

		{
			"account_resource_limit_valid",
			accountResourceLimitIndexPrimaryKeyWriter,
			"01",
			make([]byte, 1),
			expected{[]byte{0x01}, nil},
		},
		{
			"account_resource_limit_valid_invalid_key",
			accountResourceLimitIndexPrimaryKeyWriter,
			"0G",
			nil,
			expected{nil, errors.New(`account resource limit primary key writer: unable to transform primary key to byte: strconv.ParseUint: parsing "0G": invalid syntax`)},
		},

		{
			"block_resource_limit_valid",
			blockResourceLimitIndexPrimaryKeyWriter,
			"00",
			make([]byte, 1),
			expected{[]byte{0x00}, nil},
		},
		{
			"block_resource_limit_valid_invalid_key",
			blockResourceLimitIndexPrimaryKeyWriter,
			"0G",
			nil,
			expected{nil, errors.New(`block resource limit primary key writer: unable to transform primary key to byte: strconv.ParseUint: parsing "0G": invalid syntax`)},
		},

		{
			"key_account_valid",
			keyAccountIndexPrimaryKeyWriter,
			"0000000000000001:0000000000000002",
			make([]byte, 18),
			expected{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00}, nil},
		},
		{
			"key_account_wrong_chunk_count",
			keyAccountIndexPrimaryKeyWriter,
			"0000000000000001",
			nil,
			expected{nil, errors.New("key account primary key should have 2 chunks, got 1")},
		},
		{
			"key_account_invalid_chunk_1",
			keyAccountIndexPrimaryKeyWriter,
			"000000000000000G:0000000000000001",
			nil,
			expected{nil, errors.New(`key account primary key writer, chunk #1: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},
		{
			"key_account_invalid_chunk_2",
			keyAccountIndexPrimaryKeyWriter,
			"0000000000000001:000000000000000G",
			make([]byte, 8),
			expected{nil, errors.New(`key account primary key writer, chunk #2: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},

		{
			"table_data_valid",
			tableDataIndexPrimaryKeyWriter,
			"0000000000000001",
			make([]byte, 10),
			expected{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}, nil},
		},
		{
			"table_data_invalid_key",
			tableDataIndexPrimaryKeyWriter,
			"000000000000000G",
			nil,
			expected{nil, errors.New(`table data primary key writer: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},

		{
			"table_scope_valid",
			tableScopeIndexPrimaryKeyWriter,
			"0000000000000001",
			make([]byte, 10),
			expected{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}, nil},
		},
		{
			"table_scope_invalid_key",
			tableScopeIndexPrimaryKeyWriter,
			"000000000000000G",
			nil,
			expected{nil, errors.New(`table scope primary key writer: unable to transform primary key to uint64: strconv.ParseUint: parsing "000000000000000G": invalid syntax`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.primaryKeyWriter(test.primaryKey, test.buffer)

			if test.expected.err == nil {
				assert.Equal(t, test.expected.buffer, test.buffer)
			} else {
				require.NotNil(t, err)
				require.Equal(t, test.expected.err.Error(), err.Error())
			}
		})
	}
}

func TestShouldTriggerIndexing(t *testing.T) {
	tests := []struct {
		label          string
		indexRowCount  int
		mutationsCount int
		expect         bool
	}{
		{
			label:          "no indexing",
			mutationsCount: 1,
			expect:         false,
		},
		{
			label:          "flush on 999",
			mutationsCount: 999,
			expect:         false,
		},
		{
			label:          "flush on 1000",
			mutationsCount: 1000,
			expect:         true,
		},
		{
			label:          "single row table, 1500 mutations",
			indexRowCount:  1,
			mutationsCount: 1500,
			expect:         true,
		},
		{
			label:          "55000 row table, 1500 mutations",
			indexRowCount:  55000,
			mutationsCount: 1500,
			expect:         false,
		},
		{
			label:          "55000 row table, 5500 mutations",
			indexRowCount:  55000,
			mutationsCount: 5500,
			expect:         true,
		},
		{
			label:          "55000 row table, 3000 mutations",
			indexRowCount:  75000,
			mutationsCount: 3000,
			expect:         false,
		},
		{
			label:          "110000 row table, 8000 mutations",
			indexRowCount:  110000,
			mutationsCount: 8000,
			expect:         false,
		},
		{
			label:          "110000 row table, 11000 mutations",
			indexRowCount:  110000,
			mutationsCount: 11000,
			expect:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			cache := &indexCache{
				lastCounters: make(map[string]int),
				lastIndexes:  make(map[string]*TableIndex),
			}
			cache.lastCounters["a"] = test.mutationsCount
			if test.indexRowCount != 0 {
				t := &TableIndex{Map: make(map[string]uint32)}
				for i := 0; i < test.indexRowCount; i++ {
					t.Map[fmt.Sprintf("%08x", i)] = 0
				}
				cache.lastIndexes["a"] = t
			}
			res := cache.shouldTriggerIndexing("a")
			assert.Equal(t, test.expect, res)
		})
	}
}
