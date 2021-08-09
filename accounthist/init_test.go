package accounthist

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/dfuse-io/logging"
	"github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.TestingOverride()
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
				zlog.Debug("badger key",
					zap.Stringer("key", RowKey(it.Item().Key)),
					zap.String("value", hex.EncodeToString(it.Item().Value)),
				)
			}

			require.NoError(t, it.Err())
		}

		kvStore.Close()
		os.RemoveAll(tmp)
	}

	return kvStore, closer
}
