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

package codec

import (
	"context"
	"os"
	"testing"

	"github.com/dfuse-io/dstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BlockstoreStartBlockResolver_Canceled(t *testing.T) {
	bs := dstore.NewMockStore(nil)
	resolver := BlockstoreStartBlockResolver(bs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	found, irrID, err := resolver(ctx, 10)
	require.Equal(t, context.Canceled, err)
	assert.Equal(t, uint64(0), found)
	assert.Equal(t, "", irrID)
}

func Test_BlockstoreStartBlockResolver(t *testing.T) {
	sourceStoreURL := os.Getenv("TEST_SOURCE_STORE_URL") // based on eos-mainnet chain
	if sourceStoreURL == "" {
		t.Skip()
	}
	bs, err := dstore.NewDBinStore(sourceStoreURL)
	require.NoError(t, err)

	resolver := BlockstoreStartBlockResolver(bs)
	found, irrID, err := resolver(context.Background(), 1000001)
	require.NoError(t, err)
	assert.Equal(t, uint64(999673), found)
	assert.Equal(t, "", irrID)

}
