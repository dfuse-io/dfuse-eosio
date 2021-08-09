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

import (
	"fmt"
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/fluxdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockMapper(t *testing.T) {
	validABI := &eos.ABI{}

	tests := []struct {
		name            string
		input           *pbcodec.Block
		expectedEntries []string
		expectedRows    []string
	}{
		{
			name: "nothing if update doesn't change",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "data/data"),
			)),
			expectedRows: nil,
		},
		{
			name: "two different keys, two different writes",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "/............1", "/d1"),
				ct.DBOp(t, "UPD", "eosio/scope/table1/key2", "/............2", "/d2"),
			)),
			expectedRows: []string{
				`cst:eosio:scope:table1:0000000000000001:key1 => {"payer":"1","data":"6431"}`,
				`cst:eosio:scope:table1:0000000000000001:key2 => {"payer":"2","data":"6432"}`,
			},
		},
		{
			name: "two update, one sticks",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d0/d1"),
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d1/d2"),
			)),
			expectedRows: []string{
				`cst:eosio:scope:table1:0000000000000001:key1 => {"payer":"1","data":"6432"}`,
			},
		},
		{
			name: "remove, take it out",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d0/"),
			)),
			expectedRows: []string{
				`cst:eosio:scope:table1:0000000000000001:key1 => {}`,
			},
		},
		{
			name: "UPD+UPD+REM, keep the rem",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d0/d1"),
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d1/d2"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d2/"),
			)),
			expectedRows: []string{
				`cst:eosio:scope:table1:0000000000000001:key1 => {}`,
			},
		},
		{
			name: "UPD+UPD+REM, keep the rem",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d0/d1"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d1/"),
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "/............1", "/d2"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d2/"),
			)),
			expectedRows: []string{
				`cst:eosio:scope:table1:0000000000000001:key1 => {}`,
			},
		},
		{
			name: "gobble up INS+REM",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "/............1", "/d1"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d1/"),
			)),
			expectedRows: nil,
		},
		{
			name: "gobble up multiple INS+DEL",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "............1/", "/d1"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d1/"),
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "/............1", "/d2"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d2/"),
			)),
			expectedRows: nil,
		},
		{
			name: "gobble up INS+UPD+UPD+DEL",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.DBOp(t, "INS", "eosio/scope/table1/key1", "............1/", "/d1"),
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1/", "d1/d2"),
				ct.DBOp(t, "UPD", "eosio/scope/table1/key1", "............1/............1", "d2/d3"),
				ct.DBOp(t, "REM", "eosio/scope/table1/key1", "............1/", "d3/"),
			)),
			expectedRows: nil,
		},

		{
			name: "valid ABI gives a singlet entry",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.ActionTraceSetABI(t, "eosio", validABI),
			)),
			expectedEntries: []string{
				`abi:eosio:fffffffffffffffe => {"rawAbi":"000000000000000000"}`,
			},
		},
		{
			name: "invalid ABI is not an error and is ignored",
			input: ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.ActionTraceSetABI(t, "eosio", nil, ct.UndecodedActionData),
			)),
			expectedEntries: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blk := ct.ToBstreamBlock(t, test.input)
			mapper := &BlockMapper{}

			req, err := mapper.Map(blk)
			require.NoError(t, err)

			var stringEntries []string
			for _, entry := range req.SingletEntries {
				stringEntries = append(stringEntries, entryToString(t, entry))
			}
			assert.ElementsMatch(t, test.expectedEntries, stringEntries)

			var stringRows []string
			for _, row := range req.TabletRows {
				stringRows = append(stringRows, rowToString(t, row))
			}

			assert.ElementsMatch(t, test.expectedRows, stringRows)
		})
	}
}

func entryToString(t *testing.T, entry fluxdb.SingletEntry) string {
	return genericElementToString(t, entry.String(), entry)
}

func rowToString(t *testing.T, row fluxdb.TabletRow) string {
	return genericElementToString(t, row.String(), row)
}

type fluxDBElement interface {
	MarshalValue() ([]byte, error)
}

type protoableFluxDBElement interface {
	ToProto() (proto.Message, error)
}

func genericElementToString(t *testing.T, key string, element fluxDBElement) string {
	if v, ok := element.(protoableFluxDBElement); ok {
		message, err := v.ToProto()
		require.NoError(t, err)

		out, err := jsonpb.MarshalToString(message)
		require.NoError(t, err)

		return fmt.Sprintf("%s => %s", key, out)
	}

	value, err := element.MarshalValue()
	require.NoError(t, err)

	return fmt.Sprintf("%s => %x", key, value)
}
