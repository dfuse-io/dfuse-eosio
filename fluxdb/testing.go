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

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func writeBatchOfRequests(t *testing.T, db *FluxDB, requests ...*WriteRequest) {
	require.NoError(t, db.WriteBatch(context.Background(), requests))
}

func writeABI(t *testing.T, blockNum uint32, contract string, abi *eos.ABI) *WriteRequest {
	packedABI, err := eos.MarshalBinary(abi)
	require.NoError(t, err, "marshal binary abi")

	return writePackedABI(t, blockNum, contract, packedABI)
}

func writeEmptyABI(t *testing.T, blockNum uint32, contract string) *WriteRequest {
	return writeABI(t, blockNum, contract, &eos.ABI{})
}

func writePackedABI(t *testing.T, blockNum uint32, contract string, packedABI []byte) *WriteRequest {
	entry, err := NewContractABISiglet(contract).NewEntry(blockNum, packedABI)
	require.NoError(t, err)

	return &WriteRequest{
		BlockNum: blockNum,
		SigletEntries: []SigletEntry{
			entry,
		},
	}
}

func tabletRows(blockNum uint32, rows ...TabletRow) *WriteRequest {
	return &WriteRequest{
		BlockNum:   blockNum,
		TabletRows: rows,
	}
}
