package kv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
)

func init() {
	if os.Getenv("DEBUG") != "" || os.Getenv("TRACE") == "true" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}

	store.RegisterTestKVDBDriver()
}

func Test_isWriter(t *testing.T) {
	tests := []struct {
		name        string
		writes      []string
		expectValue bool
	}{
		{
			name:        "empty writer",
			writes:      []string{},
			expectValue: false,
		},
		{
			name:        "writer all",
			writes:      []string{"all"},
			expectValue: true,
		},
		{
			name:        "writer none",
			writes:      []string{"none"},
			expectValue: false,
		},
		{
			name:        "writer none",
			writes:      []string{"none", "none"},
			expectValue: false,
		},
		{
			name:        "granular writer",
			writes:      []string{"blk", "trx"},
			expectValue: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectValue, isWriter(test.writes))
		})
	}

}

func Test_New(t *testing.T) {
	tests := []struct {
		name                         string
		dsns                         []string
		expectError                  bool
		expectBlkReadStorePresent    bool
		expectBlkReadStoreDriverDsn  string
		expectTrxReadStorePresent    bool
		expectTrxReadStoreDriverDsn  string
		expectBlkWriteStorePresent   bool
		expectBlkWriteStoreDriverDsn string
		expectTrxWriteStorePresent   bool
		expectTrxWriteStoreDriverDsn string
		expectIrrBlockStorePresent   bool
		expectIrrBlockStoreDriverDsn string
	}{
		{
			name:                         "sunny path",
			dsns:                         []string{"test://dev.dev/test-trxdb-trxs?createTable=true"},
			expectBlkReadStorePresent:    true,
			expectBlkReadStoreDriverDsn:  "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectBlkWriteStorePresent:   true,
			expectBlkWriteStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectTrxWriteStorePresent:   true,
			expectTrxWriteStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectTrxReadStorePresent:    true,
			expectTrxReadStoreDriverDsn:  "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectIrrBlockStorePresent:   true,
			expectIrrBlockStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
		},
		{
			name:                       "all none",
			dsns:                       []string{"test://dev.dev/test-trxdb-trxs?createTable=true&write=none&read=none"},
			expectBlkReadStorePresent:  false,
			expectBlkWriteStorePresent: false,
			expectTrxWriteStorePresent: false,
			expectTrxReadStorePresent:  false,
			expectIrrBlockStorePresent: false,
		},
		{
			name:                         "single dsn with only a read permission specified",
			dsns:                         []string{"test://dev.dev/test-trxdb-blocks?createTable=true&read=blk"},
			expectBlkReadStorePresent:    true,
			expectBlkReadStoreDriverDsn:  "test://dev.dev/test-trxdb-blocks?createTable=true",
			expectTrxReadStorePresent:    false,
			expectBlkWriteStorePresent:   true,
			expectBlkWriteStoreDriverDsn: "test://dev.dev/test-trxdb-blocks?createTable=true",
			expectTrxWriteStorePresent:   true,
			expectTrxWriteStoreDriverDsn: "test://dev.dev/test-trxdb-blocks?createTable=true",
			expectIrrBlockStorePresent:   true,
			expectIrrBlockStoreDriverDsn: "test://dev.dev/test-trxdb-blocks?createTable=true",
		},
		{
			name: "single dsn with read and write permission specified",
			dsns: []string{
				"test://dev.dev/test-trxdb-trxs?createTable=true&read=trx&write=trx",
			},
			expectBlkReadStorePresent:    false,
			expectTrxReadStorePresent:    true,
			expectTrxReadStoreDriverDsn:  "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectBlkWriteStorePresent:   false,
			expectTrxWriteStorePresent:   true,
			expectTrxWriteStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectIrrBlockStorePresent:   true,
			expectIrrBlockStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
		},
		{
			name: "multiple dsn",
			dsns: []string{
				"test://dev.dev/test-trxdb-trxs?createTable=true&read=trx&write=trx",
				"test://dev.dev/test-trxdb-blocks?createTable=true&read=blk&write=none",
			},
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/test-trxdb-blocks?createTable=true",
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectBlkWriteStorePresent:  false,
			//expectBlkWriteStoreDriverDsn: "",
			expectTrxWriteStorePresent:   true,
			expectTrxWriteStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
			expectIrrBlockStorePresent:   true,
			expectIrrBlockStoreDriverDsn: "test://dev.dev/test-trxdb-trxs?createTable=true",
		},
		{
			name: "two writer dsn is not allowed",
			dsns: []string{
				"test://dev.dev/test-trxdb-trxs?createTable=true&read=trx&write=trx",
				"test://dev.dev/test-trxdb-blocks?createTable=true&read=blk&write=blk",
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trxdb, err := New(test.dsns)
			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if db, ok := trxdb.(*DB); ok {
				// Block Read Store Test
				if test.expectBlkReadStorePresent {
					assert.NotNil(t, db.blksReadStore)
					if d, ok := db.blksReadStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectBlkReadStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.blksReadStore)
				}

				// Block Write Store Test
				if test.expectBlkWriteStorePresent {
					assert.NotNil(t, db.blkWriteStore)
					if d, ok := db.blkWriteStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectBlkWriteStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.blkWriteStore)
				}

				// Trx Write Store Test
				if test.expectTrxWriteStorePresent {
					assert.NotNil(t, db.trxWriteStore)
					if d, ok := db.trxWriteStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectTrxWriteStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.trxWriteStore)
				}

				// Trx Read Store Test
				if test.expectTrxReadStorePresent {
					assert.NotNil(t, db.trxReadStore)
					if d, ok := db.trxReadStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectTrxReadStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.trxReadStore)
				}

				// IRR Store Test
				if test.expectIrrBlockStorePresent {
					assert.NotNil(t, db.irrBlockStore)
					if d, ok := db.irrBlockStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectIrrBlockStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.irrBlockStore)
				}

			} else {
				panic("unexpected obj. trxdb.kv.New should return a *trxdb.kv.DB object")
			}

		})
	}

}
