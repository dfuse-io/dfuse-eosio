package trxdb

import (
	"errors"
	"testing"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	Register("test", func(dsn string, logger *zap.Logger) (Driver, error) {
		return &testDriver{dsn: dsn}, nil
	})
}

func TestNewSwitchDB(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    func(tt *testing.T, actual *SwitchDB)
		expectedErr error
	}{
		{
			"all empty is an error",
			"",
			nil,
			errors.New("no switching info configured, this is invalid"),
		},
		{
			"multi writer is an error",
			"test://host.ca/path?write=block test://host.ca/path?write=account",
			nil,
			errors.New("writing driver: a writing driver has already been configured, only a single writing driver can be specified per instance, configuration is invalid"),
		},
		{
			"multi reader on same category is an error",
			"test://host.ca/path?read=block test://host.ca/path?read=account,block",
			nil,
			errors.New(`reading driver: category "INDEXABLE_CATEGORY_BLOCK" is already mapped to a driver, configuration is invalid`),
		},

		{
			"single full reading, full writing",
			"test://host.ca/path?read=*&write=*",
			func(tt *testing.T, actual *SwitchDB) {
				readDriver := &testDriver{dsn: "test://host.ca/path", options: nil}
				writeDriver := &testDriver{dsn: "test://host.ca/path", options: nil, writeOnly: FullIndexing}

				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION])
				assert.Equal(tt, writeDriver, actual.writingDriver)
			},
			nil,
		},
		{
			"one full reading, different full writing",
			"test://host.ca/read?read=* test://host.ca/write?write=*",
			func(tt *testing.T, actual *SwitchDB) {
				readDriver := &testDriver{dsn: "test://host.ca/read", options: nil}
				writeDriver := &testDriver{dsn: "test://host.ca/write", options: nil, writeOnly: FullIndexing}

				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE])
				assert.Equal(tt, readDriver, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION])
				assert.Equal(tt, writeDriver, actual.writingDriver)
			},
			nil,
		},
		{
			"multi reading, no writing",
			"test://host.ca/read1?read=account,transaction test://host.ca/read2?read=block+timeline",
			func(tt *testing.T, actual *SwitchDB) {
				readDriver1 := &testDriver{dsn: "test://host.ca/read1", options: nil}
				readDriver2 := &testDriver{dsn: "test://host.ca/read2", options: nil}

				assert.Equal(tt, readDriver1, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT])
				assert.Equal(tt, readDriver2, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK])
				assert.Equal(tt, readDriver2, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE])
				assert.Equal(tt, readDriver1, actual.readingRoutingMap[pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION])
				assert.Nil(tt, actual.writingDriver)
			},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := NewSwitchDB(test.in)
			if test.expectedErr == nil {
				require.NoError(t, err)
				test.expected(t, actual)
			} else {
				assert.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}
