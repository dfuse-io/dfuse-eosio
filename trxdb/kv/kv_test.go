package kv

import (
	"fmt"
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
	if os.Getenv("TEST_LOG") != "" {
		zlog = logging.MustCreateLoggerWithLevel("test", zap.NewAtomicLevelAt(zap.DebugLevel))
		logging.Set(zlog)
	}
}

const testBadgerTestDBFilePath = "/tmp/dfuse-test-badger"

func TestAll(t *testing.T) {

	factory := newTestDBFactory(t, testBadgerTestDBFilePath)
	trxdbtest.TestAll(t, "kv", factory)

	//trxdbtest.TestAllTimelineExplorer(t, "kv", factory)
	//trxdbtest.TestAllAccountsReader(t, "kv", factory)
	//trxdbtest.TestAllTransactionsReader(t, "kv", factory)
	//trxdbtest.TestGetLastWrittenBlockID(t, factory)
	//trxdbtest.TestGetBlock(t, factory)
	//trxdbtest.TestGetBlockByNum(t, factory)
	//trxdbtest.TestGetClosestIrreversibleIDAtBlockNum(t, factory)
	//trxdbtest.TestGetIrreversibleIDAtBlockID(t, factory)
	//trxdbtest.TestListSiblingBlocks(t, factory)
	//trxdbtest.TestGetTransactionTraces(t, factory)
	//trxdbtest.TestGetTransactionEvents(t, factory)
}

func newTestDBFactory(t *testing.T, testDbFilename string) trxdbtest.DriverFactory {
	return func() (trxdb.Driver, trxdbtest.DriverCleanupFunc) {
		return func(t *testing.T, testDbFilename string) trxdb.Driver {
				dsn := fmt.Sprintf("badger://%s", testDbFilename)
				rawdb, err := New(dsn)
				require.NoError(t, err)

				return rawdb
			}(t, testDbFilename), func() {
				err := os.RemoveAll(testDbFilename)
				if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
					require.NoError(t, err)
				}
				zlog.Debug("delete database", zap.String("", testDbFilename))
				dbCachePool = make(map[string]trxdb.Driver)
				zlog.Debug("db cache cleared")
			}
	}
}
