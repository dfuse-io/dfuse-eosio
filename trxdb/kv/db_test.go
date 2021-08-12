package kv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamingfast/logging"
	"go.uber.org/zap"

	"github.com/streamingfast/kvdb/store"
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
		expectBlkReadStoreDriver           string
		expectTrxReadStoreDriver           string
		expectIrrReadStoreDriver           string
		expectEnableBlkWrite               bool
		expectEnableTrxWrite               bool
		expectWriteStoreDriver             string
		expectLastWrittenBlockStorePresent bool
		expectLastWrittenBlockStoreDsn     string
	}{
		{
			name: "sunny path",
			dsns: []string{"test://dev.dev/aaa?createTable=true"},
			// READS
			expectBlkReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectTrxReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:   true,
			expectEnableTrxWrite:   true,
			expectWriteStoreDriver: "test://dev.dev/aaa?createTable=true",
			// Last written Block Store
			expectLastWrittenBlockStorePresent: true,
			expectLastWrittenBlockStoreDsn:     "test://dev.dev/aaa?createTable=true",
		},
		{
			name: "all none",
			dsns: []string{"test://dev.dev/aaa?createTable=true&write=none&read=none"},
			// READ
			// WRITE
			expectEnableBlkWrite: false,
			expectEnableTrxWrite: false,
			// Last written Block Store
			expectLastWrittenBlockStorePresent: false,
		},
		{
			name: "single dsn with only a read permission specified",
			dsns: []string{"test://dev.dev/aaa?createTable=true&read=blk"},
			// READ
			expectBlkReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:   true,
			expectEnableTrxWrite:   true,
			expectWriteStoreDriver: "test://dev.dev/aaa?createTable=true",
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
			expectTrxReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:   false,
			expectEnableTrxWrite:   true,
			expectWriteStoreDriver: "test://dev.dev/aaa?createTable=true",
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
			expectBlkReadStoreDriver: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:   false,
			expectEnableTrxWrite:   true,
			expectWriteStoreDriver: "test://dev.dev/aaa?createTable=true",
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
			expectBlkReadStoreDriver: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite:   false,
			expectEnableTrxWrite:   true,
			expectWriteStoreDriver: "test://dev.dev/aaa?createTable=true",
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
			expectBlkReadStoreDriver: "test://dev.dev/bbb?createTable=true",
			expectTrxReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite: false,
			expectEnableTrxWrite: false,
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
			expectBlkReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			expectIrrReadStoreDriver: "test://dev.dev/aaa?createTable=true",
			// WRITE
			expectEnableBlkWrite: false,
			expectEnableTrxWrite: false,
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
			db, err := New(test.dsns)
			if test.expectError {
				require.Error(t, err)
				return
			}

			// Block Read Store Test
			if test.expectBlkReadStoreDriver != "" {
				assert.NotNil(t, db.blkReadStore)
				if d, ok := db.blkReadStore.(*store.TestKVDBDriver); ok {
					assert.Equal(t, test.expectBlkReadStoreDriver, d.DSN)
				} else {
					panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
				}
			} else {
				assert.Nil(t, db.blkReadStore)
			}

			// Write Store Test
			if test.expectWriteStoreDriver != "" {
				assert.NotNil(t, db.writeStore)
				if d, ok := db.writeStore.(*store.TestKVDBDriver); ok {
					assert.Equal(t, test.expectWriteStoreDriver, d.DSN)
				} else {
					panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
				}
			} else {
				assert.Nil(t, db.writeStore)
			}
			assert.Equal(t, test.expectEnableTrxWrite, db.enableTrxWrite)
			assert.Equal(t, test.expectEnableBlkWrite, db.enableBlkWrite)

			// Trx Read Store Test
			if test.expectTrxReadStoreDriver != "" {
				assert.NotNil(t, db.trxReadStore)
				if d, ok := db.trxReadStore.(*store.TestKVDBDriver); ok {
					assert.Equal(t, test.expectTrxReadStoreDriver, d.DSN)
				} else {
					panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
				}
			} else {
				assert.Nil(t, db.trxReadStore)
			}

			// IRR Store Test
			if test.expectIrrReadStoreDriver != "" {
				assert.NotNil(t, db.irrReadStore)
				if d, ok := db.irrReadStore.(*store.TestKVDBDriver); ok {
					assert.Equal(t, test.expectIrrReadStoreDriver, d.DSN)
				} else {
					panic("kvdb test driver expected to be of type *store.TestKVDBDriver")
				}
			} else {
				assert.Nil(t, db.irrReadStore)
			}
		})
	}
}
