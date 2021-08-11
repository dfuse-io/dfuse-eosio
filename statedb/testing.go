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
	"context"
	"testing"

	"github.com/streamingfast/bstream"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/fluxdb"
	"github.com/stretchr/testify/require"
)

func writeBatchOfRequests(t *testing.T, db *fluxdb.FluxDB, requests ...*fluxdb.WriteRequest) {
	require.NoError(t, db.WriteBatch(context.Background(), requests))
}

func writeABI(t *testing.T, blockID string, contract string, abi *eos.ABI) *fluxdb.WriteRequest {
	packedABI, err := eos.MarshalBinary(abi)
	require.NoError(t, err, "marshal binary abi")

	return writePackedABI(t, blockID, contract, packedABI)
}

func writeEmptyABI(t *testing.T, blockID string, contract string) *fluxdb.WriteRequest {
	return writeABI(t, blockID, contract, &eos.ABI{})
}

func writePackedABI(t *testing.T, blockID string, contract string, packedABI []byte) *fluxdb.WriteRequest {
	ref := bstream.NewBlockRefFromID(blockID)

	entry, err := NewContractABISinglet(contract).Entry(ref.Num(), packedABI)
	require.NoError(t, err)

	return &fluxdb.WriteRequest{
		Height:   ref.Num(),
		BlockRef: ref,
		SingletEntries: []fluxdb.SingletEntry{
			entry,
		},
	}
}

func tabletRows(blockID string, rows ...fluxdb.TabletRow) *fluxdb.WriteRequest {
	ref := bstream.NewBlockRefFromID(blockID)

	return &fluxdb.WriteRequest{
		Height:     ref.Num(),
		BlockRef:   ref,
		TabletRows: rows,
	}
}
