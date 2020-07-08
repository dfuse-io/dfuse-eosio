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
		name                               string
		dsns                               []string
		expectError                        bool
		expectBlkReadStorePresent          bool
		expectBlkReadStoreDriverDsn        string
		expectTrxReadStorePresent          bool
		expectTrxReadStoreDriverDsn        string
		expectEnableBlkWrite               bool
		expectEnableTrxWrite               bool
		expectWriteStorePresent            bool
		expectWriteStoreDriverDsn          string
		expectLastWrittenBlockStorePresent bool
		expectLastWrittenBlockStoreDsn     string
	}{
		{
			name: "sunny path",
			dsns: []string{"test://dev.dev/aaa?createTable=true"},
			// READS
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:      true,
			expectEnableTrxWrite:      true,
			expectWriteStorePresent:   true,
			expectWriteStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "all none",
			dsns: []string{"test://dev.dev/aaa?createTable=true&write=none&read=none"},
			// READ
			expectBlkReadStorePresent: false,
			expectTrxReadStorePresent: false,
			// WRITE
			expectEnableBlkWrite:    false,
			expectEnableTrxWrite:    false,
			expectWriteStorePresent: false,
			// Last written Block Store
			expectLastWrittenBlockStorePresent: false,
		},
		{
			name: "single dsn with only a read permission specified",
			dsns: []string{"test://dev.dev/aaa?createTable=true&read=blk"},
			// READ
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			expectTrxReadStorePresent:   false,
			// WRITE
			expectEnableBlkWrite:      true,
			expectEnableTrxWrite:      true,
			expectWriteStorePresent:   true,
			expectWriteStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "single dsn with read and write permission specified",
			dsns: []string{
				"test://dev.dev/aaa?createTable=true&read=trx&write=trx",
			},
			// READ
			expectBlkReadStorePresent:   false,
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:      false,
			expectEnableTrxWrite:      true,
			expectWriteStorePresent:   true,
			expectWriteStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "multiple dsn with one write and one read",
			dsns: []string{
				"test://dev.dev/aaa?createTable=true&read=trx&write=trx",
				"test://dev.dev/bbb?createTable=true&read=blk&write=none",
			},
			// READ
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:      false,
			expectEnableTrxWrite:      true,
			expectWriteStorePresent:   true,
			expectWriteStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "multiple dsn with one write and one read and last_written_blk is on read",
			dsns: []string{
				"test://dev.dev/aaa?createTable=true&read=trx&write=trx",
				"test://dev.dev/bbb?createTable=true&read=blk,last_written_blk&write=none",
			},
			// READ
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:      false,
			expectEnableTrxWrite:      true,
			expectWriteStorePresent:   true,
			expectWriteStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/bbb?createTable=true",
		},
		{
			name: "multiple dsn with two reads trx store is the defaylt last written block",
			dsns: []string{
				"test://dev.dev/aaa?createTable=true&read=trx&write=none",
				"test://dev.dev/bbb?createTable=true&read=blk&write=none",
			},
			// READ
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStorePresent:   true,
			expectTrxReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:    false,
			expectEnableTrxWrite:    false,
			expectWriteStorePresent: false,
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "single dsn with one read trx store is the default last written block",
			dsns: []string{
				"test://dev.dev/aaa?createTable=true&read=blk&write=none",
			},
			// READ
			expectBlkReadStorePresent:   true,
			expectBlkReadStoreDriverDsn: "test://dev.dev/aaa?createTable=true",
			expectTrxReadStorePresent:   false,
			// WRITE
			expectEnableBlkWrite:    false,
			expectEnableTrxWrite:    false,
			expectWriteStorePresent: false,
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
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
					assert.NotNil(t, db.blkReadStore)
					if d, ok := db.blkReadStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectBlkReadStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.blkReadStore)
				}

				// Write Store Test
				if test.expectWriteStorePresent {
					assert.NotNil(t, db.writeStore)
					if d, ok := db.writeStore.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectWriteStoreDriverDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					assert.Nil(t, db.writeStore)
				}
				assert.Equal(t, test.expectEnableTrxWrite, db.enableTrxWrite)
				assert.Equal(t, test.expectEnableBlkWrite, db.enableBlkWrite)

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
				if test.expectLastWrittenBlockStorePresent {
					str, err := db.getLastWrittenBlockStore()
					require.NoError(t, err)
					assert.NotNil(t, str)
					if d, ok := str.(*store.TestKVDBDriver); ok {
						assert.Equal(t, test.expectLastWrittenBlockStoreDsn, d.DSN)
					} else {
						panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
					}
				} else {
					_, err := db.getLastWrittenBlockStore()
					require.Error(t, err)
				}

			} else {
				panic("unexpected obj. trxdb.kv.New should return a *trxdb.kv.DB object")
			}

		})
	}

}
