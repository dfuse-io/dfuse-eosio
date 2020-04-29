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

package hidalgo

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"github.com/hidal-go/hidalgo/kv"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gotest.tools/assert"
)

// Those tests works after having injected about 200 blocks from eos-dev1
var testHidalgoDatabasePath = ""

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}

	testHidalgoDatabasePath = os.Getenv("HIDALGO_DB_PATH")
}

func Test_PrintAllKeys(t *testing.T) {
	runOnlyIfPathIsSet(t)

	ctx := context.Background()
	store := testCreateStore(t)

	PrintTable(t, ctx, store, store.tblABIs, false)
	PrintTable(t, ctx, store, store.tblLast, true)
}

func Test_FetchLastWrittenBlock(t *testing.T) {
	runOnlyIfPathIsSet(t)

	ctx := context.Background()
	store := testCreateStore(t)

	out, err := store.FetchLastWrittenBlock(ctx, "block")
	require.NoError(t, err)

	fmt.Println("Last written block: " + out.String())
}

func Test_FetchABI(t *testing.T) {
	runOnlyIfPathIsSet(t)

	ctx := context.Background()
	store := testCreateStore(t)

	_, packedABI, err := store.FetchABI(ctx, "5530ea0000000000:", "5530ea0000000000:fffff768", "5530ea0000000000:ffffffff")
	require.NoError(t, err)

	abi := &eos.ABI{}
	abi.SetFitNodeos(true)
	err = eos.UnmarshalBinary(packedABI, abi)
	require.NoError(t, err)

	//fmt.Println("Got a packed ABI: " + abi.Version + " @ " + strconv.FormatUint(uint64(blockNum), 10))
}

func PrintTable(t *testing.T, ctx context.Context, store *KVStore, bucket string, asString bool) {
	fmt.Printf("Table %q\n", bucket)
	err := kv.View(store.db, func(tx kv.Tx) error {
		return kv.Each(ctx, tx, kv.SKey(bucket), func(kvKey kv.Key, value kv.Value) error {
			_, key := keyToString(kvKey)

			if asString {
				fmt.Printf("- %s => %s\n", key, string(value))
			} else {
				fmt.Printf("- %s => %s\n", key, valueToString(value))
			}
			return nil
		})
	})

	fmt.Println()
	require.NoError(t, err)
}

func PrintTablePrefix(t *testing.T, ctx context.Context, store *KVStore, bucket string, asString bool, prefix string) {
	fmt.Printf("Table %q\n", bucket)
	err := store.scanPrefix(ctx, bucket, prefix, func(key string, value []byte) error {
		if asString {
			fmt.Printf("- %s => %s\n", key, string(value))
		} else {
			fmt.Printf("- %s => %s\n", key, valueToString(value))
		}
		return nil
	})

	fmt.Println()
	require.NoError(t, err)
}

func PrintTableRange(t *testing.T, ctx context.Context, store *KVStore, bucket string, asString bool, keyStart, keyEnd string) {
	fmt.Printf("Table %q\n", bucket)
	err := store.scanRange(ctx, bucket, keyStart, keyEnd, func(key string, value []byte) error {
		if asString {
			fmt.Printf("- %s => %s\n", key, string(value))
		} else {
			fmt.Printf("- %s => %s\n", key, valueToString(value))
		}
		return nil
	})

	fmt.Println()
	require.NoError(t, err)
}

func valueToString(value kv.Value) string {
	bytes := []byte(value)
	if len(bytes) > 64 {
		bytes = bytes[0:64]
	}

	return hex.EncodeToString(bytes)
}

func testCreateStore(t *testing.T) *KVStore {
	url := fmt.Sprintf("bbolt://%s?createTables=true", testHidalgoDatabasePath)
	store, err := NewKVStore(context.Background(), url)
	require.NoError(t, err)

	return store
}

func runOnlyIfPathIsSet(t *testing.T) {
	if testHidalgoDatabasePath == "" {
		t.Skip("Test skip, provide HIDALGO_DB_PATH environment variable to activate those test")
	}
}

func Test_parseDNS(t *testing.T) {
	tests := []struct {
		name              string
		dns               string
		expectError       bool
		expectPath        string
		expectCreateTable bool
	}{
		{
			name:        "golden path",
			dns:         "bbolt://fluxdb.bbolt",
			expectError: false,
			expectPath:  "fluxdb.bbolt",
		},
		{
			name:              "with created tables",
			dns:               "bbolt://fluxdb.bbolt?createTables=true",
			expectError:       false,
			expectPath:        "fluxdb.bbolt",
			expectCreateTable: true,
		},
		{
			name:              "local dir",
			dns:               "bbolt:///Users/john/dfuse-data/fluxdb/fluxdb.bbolt?createTables=true",
			expectError:       false,
			expectPath:        "/Users/john/dfuse-data/fluxdb/fluxdb.bbolt",
			expectCreateTable: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsn, err := parseDNS(test.dns)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, dsn.path, test.expectPath)
				assert.Equal(t, dsn.createTable, test.expectCreateTable)
			}

		})
	}

}
