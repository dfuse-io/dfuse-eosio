package accounthist

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/logging"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		kvStore.Close()
		os.RemoveAll(tmp)
	}

	return kvStore, closer
}

func assertEqualHex(t *testing.T, expected string, actual []byte, msgAndArgs ...interface{}) {
	assert.Equal(t, expected, hex.EncodeToString(actual))
}
