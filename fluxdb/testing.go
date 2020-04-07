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
	"testing"

	"github.com/dfuse-io/derr"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func executeWriteRequests(t *testing.T, db *FluxDB, requests ...*WriteRequest) {
	require.NoError(t, db.WriteBatch(context.Background(), requests))
}

func writeRequests(requests ...*WriteRequest) []*WriteRequest {
	return requests
}

func writePackedABI(blockNum uint32, account uint64, packedABI []byte) *WriteRequest {
	return &WriteRequest{
		BlockNum: blockNum,
		ABIs:     []*ABIRow{&ABIRow{account, blockNum, packedABI}},
	}
}

func writeABI(blockNum uint32, account uint64, abi *eos.ABI) *WriteRequest {
	bytes, err := eos.MarshalBinary(abi)
	if err != nil {
		panic(derr.Wrap(err, "unable to encode abi"))
	}

	if len(bytes) == 0 {
		panic("encoded ABI should have at least 1 byte, use writeEmptyABI if you don't care about the actual content.")
	}

	return writePackedABI(blockNum, account, bytes)
}

func writeEmptyABI(blockNum uint32, account uint64) *WriteRequest {
	return writePackedABI(blockNum, account, []byte("empty"))
}

func tableDataRows(blockNum uint32, rows ...*TableDataRow) *WriteRequest {
	return &WriteRequest{
		BlockNum:   blockNum,
		TableDatas: rows,
	}
}
