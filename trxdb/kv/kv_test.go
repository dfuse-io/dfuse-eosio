package kv

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dfuse-eosio/trxdb/trxdbtest"
	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" || os.Getenv("TRACE") == "true" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func TestAll(t *testing.T) {
	factory := newTestDBFactory(t)
	trxdbtest.TestAll(t, "kv", factory)
}

func newTestDBFactory(t *testing.T) trxdbtest.DriverFactory {
	return func() (trxdb.Driver, trxdbtest.DriverCleanupFunc) {
		dir, err := ioutil.TempDir("", "dfuse-trxdb-kv")
		require.NoError(t, err)

		db, err := New(fmt.Sprintf("badger://%s", dir), zlog)
		require.NoError(t, err)

		return db, func() {
			err := os.RemoveAll(dir)
			if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
				require.NoError(t, err)
			}
		}
	}
}
