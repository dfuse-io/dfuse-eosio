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
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthLink_BuildData(t *testing.T) {
	updatedRow := &AuthLinkRow{
		PermissionName: N(""),
	}

	updatedRowValue := updatedRow.buildData()

	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, updatedRowValue)
}

func TestAuthLink_RowKey(t *testing.T) {
	tests := []struct {
		row      *AuthLinkRow
		blockNum uint32
		expected string
	}{
		{
			&AuthLinkRow{false, N("eosio"), N("token"), N("transfer"), N("active")},
			0,
			"al:5530ea0000000000:00000000:cd20a98000000000:cdcd3c2d57000000",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := test.row.rowKey(test.blockNum)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestKeyAccount_RowKey(t *testing.T) {
	tests := []struct {
		row      *KeyAccountRow
		blockNum uint32
		expected string
	}{
		{
			&KeyAccountRow{"EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP", N("eosio"), N("active"), false},
			0,
			"ka2:EOS5MHPYyhjBjnQZejzZHqHewPWhGTfQWSVTWYEhDmJu4SXkzgweP:00000000:5530ea0000000000:3232eda800000000",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := test.row.rowKey(test.blockNum)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTableData_RowKey(t *testing.T) {
	tests := []struct {
		row      *TableDataRow
		blockNum uint32
		expected string
	}{
		{
			&TableDataRow{N("eosio"), N("scope"), N("table"), N("key"), N("payer"), false, nil},
			0,
			"td:5530ea0000000000:c98f150000000000:c229550000000000:00000000:82bc000000000000",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := test.row.rowKey(test.blockNum)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTableScope_RowKey(t *testing.T) {
	tests := []struct {
		row      *TableScopeRow
		blockNum uint32
		expected string
	}{
		{
			&TableScopeRow{N("eosio"), N("scope"), N("table"), false, N("payer")},
			0,
			"ts:5530ea0000000000:c98f150000000000:00000000:c229550000000000",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := test.row.rowKey(test.blockNum)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestPurgeShards(t *testing.T) {
	newReq := func() *WriteRequest {
		return &WriteRequest{
			ABIs: []*ABIRow{
				{},
				{},
			},
			AuthLinks: []*AuthLinkRow{
				{Account: N("mama")},
				{Account: N("papa")},
			},
			KeyAccounts: []*KeyAccountRow{
				{PublicKey: "EOS123"},
				{PublicKey: "EOS234"},
			},
			TableDatas: []*TableDataRow{
				{Account: N("mama"), Table: N("papa"), Scope: N("rita")},
				{Account: N("mama"), Table: N("papa"), Scope: N("foo")},
				{Account: N("foo"), Table: N("bar"), Scope: N("baz")},
			},
			TableScopes: []*TableScopeRow{
				{Account: N("mama"), Table: N("papa2")},
				{Account: N("mama"), Table: N("papa")},
				{Account: N("foo"), Table: N("bar")},
			},
		}
	}

	w := newReq()
	w.purgeShardedRows(0, 2)

	assert.Len(t, w.AuthLinks, 1)
	assert.Len(t, w.KeyAccounts, 2)
	assert.Equal(t, "EOS123", w.KeyAccounts[0].PublicKey)
	assert.Equal(t, "EOS234", w.KeyAccounts[1].PublicKey)
	assert.Len(t, w.TableDatas, 1)
	assert.Len(t, w.TableScopes, 2)
	assert.Len(t, w.ABIs, 2)

	w = newReq()
	w.purgeShardedRows(1, 2)

	assert.Len(t, w.AuthLinks, 1)
	assert.Len(t, w.KeyAccounts, 0)
	assert.Len(t, w.TableDatas, 2)
	assert.Len(t, w.TableScopes, 1)
	assert.Len(t, w.ABIs, 0)
}

func TestWhoHasWhatShardContentiousTableNames(t *testing.T) {
	include := func(tableKey string, shardCount uint32) uint32 {
		h := md5.New()
		_, _ = h.Write([]byte(tableKey))
		md5Hash := h.Sum(nil)

		bigInt := binary.LittleEndian.Uint32(md5Hash)

		elementShard := bigInt % shardCount

		return elementShard
	}

	// Belongs to shard 4, out of 100
	row := &TableDataRow{Account: N("eosio"), Table: N("global"), Scope: N("eosio")}
	assert.Equal(t, 0, int(include(row.tableKey(), 100)))

	row = &TableDataRow{Account: N("eosio"), Table: N("voters"), Scope: N("eosio")}
	assert.Equal(t, 52, int(include(row.tableKey(), 100)))

	row = &TableDataRow{Account: N("eosbetdice11"), Table: N("globalvars"), Scope: N("eosbetdice11")}
	assert.Equal(t, 92, int(include(row.tableKey(), 100)))

	// Belongs to shard ?
	assert.Equal(t, 22, int(include("brl", 100)))
}
