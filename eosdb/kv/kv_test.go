package kv

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/eosdb/eosdbtest"
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
	eosdbtest.TestAll(t, "kv", factory)

	//eosdbtest.TestAllTimelineExplorer(t, "kv", factory)
	//eosdbtest.TestAllAccountsReader(t, "kv", factory)
	//eosdbtest.TestAllTransactionsReader(t, "kv", factory)
	//eosdbtest.TestGetLastWrittenBlockID(t, factory)
	//eosdbtest.TestGetBlock(t, factory)
	//eosdbtest.TestGetBlockByNum(t, factory)
	//eosdbtest.TestGetClosestIrreversibleIDAtBlockNum(t, factory)
	//eosdbtest.TestGetIrreversibleIDAtBlockID(t, factory)
	//eosdbtest.TestListSiblingBlocks(t, factory)
	//eosdbtest.TestGetTransactionTraces(t, factory)
	//eosdbtest.TestGetTransactionEvents(t, factory)
}

func newTestDBFactory(t *testing.T, testDbFilename string) eosdbtest.DriverFactory {
	return func() (eosdb.Driver, eosdbtest.DriverCleanupFunc) {
		return func(t *testing.T, testDbFilename string) eosdb.Driver {
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
				dbCachePool = make(map[string]eosdb.Driver)
				zlog.Debug("db cache cleared")
			}
	}
}
