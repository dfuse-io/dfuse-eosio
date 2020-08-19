// Copyright 2019 dfuse Platform Inc.
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

package trxdbtest

import (
	"context"
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dbWritterTests = []DriverTestFunc{
	TestPutBlock,
	TestUpdateNowIrreversibleBlock,
}

func TestPutBlock(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		block         *pbcodec.Block
		expectErr     bool
		expectBlockID string
	}{
		{
			name:          "golden path",
			block:         ct.Block(t, "00000002aa"),
			expectBlockID: "00000002aa",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db, clean := driverFactory()
			defer clean()

			require.NoError(t, db.PutBlock(ctx, test.block))
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetBlock(ctx, test.block.Id)
			require.NoError(t, err)
			assert.Equal(t, test.block.Id, resp.Block.Id)
		})
	}
}

func TestUpdateNowIrreversibleBlock(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		block         *pbcodec.Block
		expectErr     bool
		expectBlockID string
	}{
		{
			name:          "golden path",
			block:         ct.Block(t, "00000002aa"),
			expectBlockID: "00000002aa",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db, clean := driverFactory()
			defer clean()

			require.NoError(t, db.PutBlock(ctx, test.block))
			require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, test.block))
			require.NoError(t, db.Flush(ctx))

			resp, err := db.GetBlock(ctx, test.block.Id)
			require.NoError(t, err)
			assert.True(t, resp.Irreversible)
		})
	}
}
