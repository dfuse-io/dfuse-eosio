package kv

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dfuse-eosio/trxdb/kv"
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

	if !trxdb.IsRegistered("badger") {
		trxdb.Register("badger", kv.New)
	}
}

// Those tests are here only because of a loop in package resolving. The SwitchDB implementation
// cannot live in its own package because it depends on `trxdb` package (and)
func TestAll(t *testing.T) {
	switchFactory := newTestSwitchDBFactory(t)
	trxdbtest.TestAll(t, "switch", switchFactory)
}

func newTestSwitchDBFactory(t *testing.T) trxdbtest.DriverFactory {
	return func() (trxdb.Driver, trxdbtest.DriverCleanupFunc) {
		dir, err := ioutil.TempDir("", "dfuse-trxdb-switch")
		require.NoError(t, err)

		readingDSN := fmt.Sprintf("badger://%s?read=*", dir)
		writingDSN := fmt.Sprintf("badger://%s?write=*", dir)

		db, err := trxdb.NewSwitchDB(readingDSN + " " + writingDSN)

		return db, func() {
			err := os.RemoveAll(dir)
			if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
				require.NoError(t, err)
			}
		}
	}
}
