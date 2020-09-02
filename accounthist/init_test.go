package accounthist

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.TestingOverride()

	// 02 0000000080a024c5 ff ffffffffffffffff
	// 02 0000000000a124c5 ff ffffffffffffffff
}

func getKVTestFactory(t *testing.T) (store.KVStore, func()) {
	tmp, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	kvStore, err := store.New(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
	require.NoError(t, err)

	closer := func() {
		if traceEnabled {
			endKey := make([]byte, 512)
			for i := 0; i < len(endKey); i++ {
				endKey[i] = 0xFF
			}

			it := kvStore.Scan(context.Background(), []byte{}, endKey, int(math.MaxInt64))
			for it.Next() {
				zlog.Debug("badger key", zap.Stringer("key", Key(it.Item().Key)), zap.Stringer("value", Key(it.Item().Value)))
			}

			require.NoError(t, it.Err())
		}

		kvStore.Close()
		os.RemoveAll(tmp)
	}

	return kvStore, closer
}

func assertEqualHex(t *testing.T, expected string, actual []byte, msgAndArgs ...interface{}) {
	assert.Equal(t, expected, hex.EncodeToString(actual))
}
