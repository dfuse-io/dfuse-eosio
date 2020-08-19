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

package statedb

// import (
// 	"sort"
// 	"strings"
// 	"testing"

// 	"github.com/dfuse-io/dfuse-eosio/codec"
// 	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
// 	timestamp "github.com/golang/protobuf/ptypes/timestamp"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestPreprocessBlock_DbOps(t *testing.T) {
// 	tests := []struct {
// 		name   string
// 		input  []*pbcodec.DBOp
// 		expect []TabletRow
// 	}{
// 		{
// 			name: "nothing if update doesn't change",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer/payer", "data/data"),
// 			},
// 			expect: nil,
// 		},
// 		{
// 			name: "two different keys, two different writes",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
// 				testDBOp("INS", "eosio/scope/table1/key2", "/payer2", "/d2"),
// 			},
// 			expect: []TabletRow{
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key1", "payer1", []byte("d1"), false),
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key2", "payer2", []byte("d2"), false),
// 			},
// 		},
// 		{
// 			name: "two updt, one sticks",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
// 			},
// 			expect: []TabletRow{
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key1", "payer1", []byte("d2"), false),
// 			},
// 		},
// 		{
// 			name: "remove, take it out",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d0/"),
// 			},
// 			expect: []TabletRow{
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key1", "", nil, true),
// 			},
// 		},
// 		{
// 			name: "UPD+UPD+REM, keep the rem",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d2/"),
// 			},
// 			expect: []TabletRow{
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key1", "", nil, true),
// 			},
// 		},
// 		{
// 			name: "UPD+REM+INS+REM, still keep the rem",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d0/d1"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d2"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d2/"),
// 			},
// 			expect: []TabletRow{
// 				mustCreateContractStateTabletRow("eosio", "scope", "table1", 0, "key1", "", nil, true),
// 			},
// 		},
// 		{
// 			name: "gobble up INS+DEL",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
// 			},
// 			expect: nil,
// 		},
// 		{
// 			name: "gobble up multiple INS+DEL",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d1/"),
// 			},
// 			expect: nil,
// 		},
// 		{
// 			name: "gobble up INS+UPD+UPD+DEL",
// 			input: []*pbcodec.DBOp{
// 				testDBOp("INS", "eosio/scope/table1/key1", "/payer1", "/d1"),
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d1/d2"),
// 				testDBOp("UPD", "eosio/scope/table1/key1", "payer1/payer1", "d2/d3"),
// 				testDBOp("REM", "eosio/scope/table1/key1", "payer1/", "d3/"),
// 			},
// 			expect: nil,
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {

// 			blk := newBlock("0000003a", []string{"1", "2"})
// 			blk.TransactionTraces()[0].DbOps = test.input

// 			bstreamBlock, err := codec.BlockFromProto(blk)
// 			require.NoError(t, err)

// 			req, err := PreprocessBlock(bstreamBlock)
// 			require.NoError(t, err)

// 			assert.ElementsMatch(t, test.expect, req.(*WriteRequest).TabletRows)
// 		})
// 	}
// }

// func testDBOp(op string, path, payers, datas string) *pbcodec.DBOp {
// 	chunks := strings.SplitN(path, "/", 4)
// 	payerChunks := strings.SplitN(payers, "/", 2)
// 	dataChunks := strings.SplitN(datas, "/", 2)

// 	out := &pbcodec.DBOp{
// 		Code:       chunks[0],
// 		Scope:      chunks[1],
// 		TableName:  chunks[2],
// 		PrimaryKey: chunks[3],
// 		OldPayer:   payerChunks[0],
// 		NewPayer:   payerChunks[1],
// 		OldData:    []byte(dataChunks[0]),
// 		NewData:    []byte(dataChunks[1]),
// 	}
// 	switch op {
// 	case "INS":
// 		out.Operation = pbcodec.DBOp_OPERATION_INSERT
// 	case "REM":
// 		out.Operation = pbcodec.DBOp_OPERATION_REMOVE
// 	case "UPD":
// 		out.Operation = pbcodec.DBOp_OPERATION_UPDATE
// 	default:
// 		panic("wtf-happy? I know not that thing")
// 	}
// 	return out
// }

// func newBlock(blockID string, trxIDs []string) *pbcodec.Block {
// 	traces := make([]*pbcodec.TransactionTrace, len(trxIDs))
// 	for i, trxID := range trxIDs {
// 		traces[i] = &pbcodec.TransactionTrace{
// 			Id: trxID,
// 		}
// 	}

// 	blk := &pbcodec.Block{
// 		Id:                          blockID,
// 		UnfilteredTransactionTraces: traces,
// 		Header: &pbcodec.BlockHeader{
// 			Timestamp: &timestamp.Timestamp{Seconds: 1569604302},
// 		},
// 	}
// 	return blk
// }

// func newPermOp(operation string, actionIndex int, oldPerm, newPerm *pbcodec.PermissionObject) *pbcodec.PermOp {
// 	pbcodecOperation := pbcodec.PermOp_OPERATION_UNKNOWN
// 	switch operation {
// 	case "INS":
// 		pbcodecOperation = pbcodec.PermOp_OPERATION_INSERT
// 	case "UPD":
// 		pbcodecOperation = pbcodec.PermOp_OPERATION_UPDATE
// 	case "REM":
// 		pbcodecOperation = pbcodec.PermOp_OPERATION_REMOVE
// 	}

// 	return &pbcodec.PermOp{
// 		Operation:   pbcodecOperation,
// 		ActionIndex: uint32(actionIndex),
// 		OldPerm:     oldPerm,
// 		NewPerm:     newPerm,
// 	}
// }

// func newPermOpData(account string, permission string, publicKeys []string) *pbcodec.PermissionObject {
// 	authKeys := make([]*pbcodec.KeyWeight, len(publicKeys))
// 	for i, publicKey := range publicKeys {
// 		authKeys[i] = &pbcodec.KeyWeight{PublicKey: publicKey, Weight: 1}
// 	}

// 	return &pbcodec.PermissionObject{
// 		Owner: account,
// 		Name:  permission,
// 		Authority: &pbcodec.Authority{
// 			Keys: authKeys,
// 		},
// 	}
// }

// func sortedFluxRows(rows []TabletRow, blockNum uint32) []TabletRow {
// 	sort.Slice(rows, func(i, j int) bool {
// 		return rows[i].Key() < rows[j].Key()
// 	})

// 	return rows
// }

// func mustCreateContractTableScopeTabletRow(contract, table string, blockNum uint32, scope string, payer string, isDeletion bool) TabletRow {
// 	row, err := NewContractTableScopeTablet(contract, table).NewRow(blockNum, scope, payer, isDeletion)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return row
// }

// func mustCreateContractStateTabletRow(contract, scope, table string, blockNum uint32, primaryKey string, payer string, data []byte, isDeletion bool) TabletRow {
// 	row, err := NewContractStateTablet(contract, scope, table).NewRow(blockNum, primaryKey, payer, data, isDeletion)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return row
// }

// func mustCreateAuthLinkTabletRow(account string, blockNum uint32, contract, action, permission string, isDeletion bool) TabletRow {
// 	row, err := NewAuthLinkTablet(account).NewRow(blockNum, contract, action, permission, isDeletion)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return row
// }

// func mustCreateKeyAccountTabletRow(publicKey string, blockNum uint32, account, permission string, isDeletion bool) *KeyAccountRow {
// 	tablet := NewKeyAccountTablet(publicKey)
// 	k, err := tablet.NewRow(blockNum, account, permission, isDeletion)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return k
// }
