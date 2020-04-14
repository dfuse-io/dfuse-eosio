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
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/codecs/deos"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/eoscanada/eos-go"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocessBlock_TableOps(t *testing.T) {
	blk := newBlock("0000003a", []string{"1", "2"})
	blk.TransactionTraces[0].TableOps = []*pbdeos.TableOp{
		{Operation: pbdeos.TableOp_OPERATION_INSERT, ActionIndex: 0, Payer: "eosio", Code: "eosio", Scope: "scope", TableName: "table1"},
		{Operation: pbdeos.TableOp_OPERATION_INSERT, ActionIndex: 0, Payer: "john", Code: "john", Scope: "scope2", TableName: "table3"},
		{Operation: pbdeos.TableOp_OPERATION_REMOVE, ActionIndex: 0, Payer: "eosio", Code: "eosio", Scope: "scope", TableName: "table1"},
	}

	blk.TransactionTraces[1].TableOps = []*pbdeos.TableOp{
		{Operation: pbdeos.TableOp_OPERATION_REMOVE, ActionIndex: 0, Payer: "another", Code: "another", Scope: "scope1", TableName: "table1"},
	}

	bstreamBlock, err := deos.BlockFromProto(blk)
	require.NoError(t, err)
	req, err := PreprocessBlock(bstreamBlock)
	require.NoError(t, err)

	tableScopeRows := req.(*WriteRequest).TableScopes
	tableScopeRowKey := func(row *TableScopeRow) string {
		return strings.Join([]string{
			eos.NameToString(row.Account),
			eos.NameToString(row.Table),
			eos.NameToString(row.Scope),
			eos.NameToString(row.Payer),
		}, "/")
	}

	sort.Slice(tableScopeRows, func(i, j int) bool {
		return tableScopeRowKey(tableScopeRows[i]) < tableScopeRowKey(tableScopeRows[j])
	})

	assert.Equal(t, []*TableScopeRow{
		{N("another"), N("scope1"), N("table1"), true, N("another")},
		{N("eosio"), N("scope"), N("table1"), true, N("eosio")},
		{N("john"), N("scope2"), N("table3"), false, N("john")},
	}, tableScopeRows)
}

func TestPreprocessBlock_DbOps(t *testing.T) {
	tests := []struct {
		name   string
		input  []*pbdeos.DBOp
		expect []*TableDataRow
	}{
		{
			name: "nothing if update doesn't change",
			input: []*pbdeos.DBOp{
				testDBOp("UPD", "eosio/scope/table1/key1", "payer/payer", "data/data"),
			},
			expect: nil,
		},
		{
			name: "two different keys, two different writes",
			input: []*pbdeos.DBOp{
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
				testDBOp("INS", "eosio/scope/table1/key2", "/payer2", "/d2"),
			},
			expect: []*TableDataRow{
				{N("eosio"), N("scope"), N("table1"), N("key1"), N("payer1"), false, []byte("d1")},
				{N("eosio"), N("scope"), N("table1"), N("key2"), N("payer2"), false, []byte("d2")},
			},
		},
		{
			name: "two updt, one sticks",
			input: []*pbdeos.DBOp{
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
			},
			expect: []*TableDataRow{
				{N("eosio"), N("scope"), N("table1"), N("key1"), N("payer1"), false, []byte("d2")},
			},
		},
		{
			name: "remove, take it out",
			input: []*pbdeos.DBOp{
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d0/"),
			},
			expect: []*TableDataRow{
				{N("eosio"), N("scope"), N("table1"), N("key1"), N(""), true, nil},
			},
		},
		{
			name: "UPD+UPD+REM, keep the rem",
			input: []*pbdeos.DBOp{
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d2/"),
			},
			expect: []*TableDataRow{
				{N("eosio"), N("scope"), N("table1"), N("key1"), N(""), true, nil},
			},
		},
		{
			name: "UPD+REM+INS+REM, still keep the rem",
			input: []*pbdeos.DBOp{
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d2"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d2/"),
			},
			expect: []*TableDataRow{
				{N("eosio"), N("scope"), N("table1"), N("key1"), N(""), true, nil},
			},
		},
		{
			name: "gobble up INS+DEL",
			input: []*pbdeos.DBOp{
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
			},
			expect: nil,
		},
		{
			name: "gobble up multiple INS+DEL",
			input: []*pbdeos.DBOp{
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
			},
			expect: nil,
		},
		{
			name: "gobble up INS+UPD+UPD+DEL",
			input: []*pbdeos.DBOp{
				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d2/d3"),
				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d3/"),
			},
			expect: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			blk := newBlock("0000003a", []string{"1", "2"})
			blk.TransactionTraces[0].DbOps = test.input

			bstreamBlock, err := deos.BlockFromProto(blk)
			require.NoError(t, err)

			req, err := PreprocessBlock(bstreamBlock)
			require.NoError(t, err)

			dataRows := req.(*WriteRequest).TableDatas

			assert.Equal(t, test.expect, dataRows)
		})
	}

}

func testDBOp(op string, path, payers, datas string) *pbdeos.DBOp {
	chunks := strings.SplitN(path, "/", 4)
	payerChunks := strings.SplitN(payers, "/", 2)
	dataChunks := strings.SplitN(datas, "/", 2)

	out := &pbdeos.DBOp{
		Code:       chunks[0],
		Scope:      chunks[1],
		TableName:  chunks[2],
		PrimaryKey: chunks[3],
		OldPayer:   payerChunks[0],
		NewPayer:   payerChunks[1],
		OldData:    []byte(dataChunks[0]),
		NewData:    []byte(dataChunks[1]),
	}
	switch op {
	case "INS":
		out.Operation = pbdeos.DBOp_OPERATION_INSERT
	case "REM":
		out.Operation = pbdeos.DBOp_OPERATION_REMOVE
	case "UPD":
		out.Operation = pbdeos.DBOp_OPERATION_UPDATE
	default:
		panic("wtf-happy? I know not that thing")
	}
	return out
}

func TestPreprocessBlock_PermOps(t *testing.T) {
	blk := newBlock("0000003a", []string{"1", "2"})
	blk.TransactionTraces[0].PermOps = []*pbdeos.PermOp{
		newPermOp("INS", 0, nil, newPermOpData("eosio", "owner", []string{"k1", "k2"})),
		newPermOp("INS", 1, nil, newPermOpData("eosio", "active", []string{"k2"})),
		newPermOp("REM", 0, newPermOpData("eosio", "owner", []string{"k2"}), nil),
	}

	blk.TransactionTraces[1].PermOps = []*pbdeos.PermOp{
		newPermOp("INS", 0, nil, newPermOpData("eosio", "owner", []string{"k3"})),
	}

	bstreamBlock, err := deos.BlockFromProto(blk)
	require.NoError(t, err)
	req, err := PreprocessBlock(bstreamBlock)
	require.NoError(t, err)

	keyAccountRows := req.(*WriteRequest).KeyAccounts
	key := func(row *KeyAccountRow) string {
		return fmt.Sprintf("%s:%s:%s", row.PublicKey, eos.NameToString(row.Account), eos.NameToString(row.Permission))
	}

	sort.Slice(keyAccountRows, func(i, j int) bool {
		return key(keyAccountRows[i]) < key(keyAccountRows[j])
	})

	assert.Equal(t, []*KeyAccountRow{
		{"k1", N("eosio"), N("owner"), false},
		{"k2", N("eosio"), N("active"), false},
		{"k2", N("eosio"), N("owner"), true},
		{"k3", N("eosio"), N("owner"), false},
	}, keyAccountRows)
}

func newBlock(blockID string, trxIDs []string) *pbdeos.Block {
	traces := make([]*pbdeos.TransactionTrace, len(trxIDs))
	for i, trxID := range trxIDs {
		traces[i] = &pbdeos.TransactionTrace{
			Id: trxID,
		}
	}

	blk := &pbdeos.Block{
		Id:                blockID,
		TransactionTraces: traces,
		Header: &pbdeos.BlockHeader{
			Timestamp: &timestamp.Timestamp{Seconds: 1569604302},
		},
	}
	return blk
}

func newPermOp(operation string, actionIndex int, oldPerm, newPerm *pbdeos.PermissionObject) *pbdeos.PermOp {
	pbdeosOperation := pbdeos.PermOp_OPERATION_UNKNOWN
	switch operation {
	case "INS":
		pbdeosOperation = pbdeos.PermOp_OPERATION_INSERT
	case "UPD":
		pbdeosOperation = pbdeos.PermOp_OPERATION_UPDATE
	case "REM":
		pbdeosOperation = pbdeos.PermOp_OPERATION_REMOVE
	}

	return &pbdeos.PermOp{
		Operation:   pbdeosOperation,
		ActionIndex: uint32(actionIndex),
		OldPerm:     oldPerm,
		NewPerm:     newPerm,
	}
}

func newPermOpData(account string, permission string, publicKeys []string) *pbdeos.PermissionObject {
	authKeys := make([]*pbdeos.KeyWeight, len(publicKeys))
	for i, publicKey := range publicKeys {
		authKeys[i] = &pbdeos.KeyWeight{PublicKey: publicKey, Weight: 1}
	}

	return &pbdeos.PermissionObject{
		Owner: account,
		Name:  permission,
		Authority: &pbdeos.Authority{
			Keys: authKeys,
		},
	}
}
