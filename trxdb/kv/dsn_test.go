package kv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dfuse-io/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func Test_parseDSN(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		expectCleanDSN string
		expectRead     []string
		expectWrite    []string
		expectError    bool
	}{
		{
			name:           "simple dsn",
			dsn:            "test://host.ca/path",
			expectCleanDSN: "test://host.ca/path",
			expectRead:     []string{"all"},
			expectWrite:    []string{"all"},
		},
		{
			name:           "dsn with read options dsn",
			dsn:            "test://host.ca/path?read=blk,trx",
			expectCleanDSN: "test://host.ca/path",
			expectRead:     []string{"blk", "trx"},
			expectWrite:    []string{"all"},
		},
		{
			name:           "dsn with write options dsn",
			dsn:            "test://host.ca/path?write=trx",
			expectCleanDSN: "test://host.ca/path",
			expectRead:     []string{"all"},
			expectWrite:    []string{"trx"},
		},
		{
			name:           "dsn with read and write options",
			dsn:            "test://host.ca/path?read=blk,trx&write=none",
			expectCleanDSN: "test://host.ca/path",
			expectRead:     []string{"blk", "trx"},
			expectWrite:    []string{"none"},
		},
		{
			name:           "dsn with read none and write options",
			dsn:            "test://host.ca/path?read=none&write=blk",
			expectCleanDSN: "test://host.ca/path",
			expectRead:     []string{"none"},
			expectWrite:    []string{"blk"},
		},
		{
			name:           "dsn with only read",
			dsn:            "bigkv://dev.dev/test-trxdb-blocks?createTable=true&read=account,block,timeline,last_written_blk",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-blocks?createTable=true",
			expectRead:     []string{"account", "block", "timeline", "last_written_blk"},
			expectWrite:    []string{"all"},
		},
		{
			name:           "dsn with read nonde and write options",
			dsn:            "bigkv://dev.dev/test-trxdb-trxs?createTable=true&read=transaction&write=transaction",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-trxs?createTable=true",
			expectRead:     []string{"transaction"},
			expectWrite:    []string{"transaction"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clean, read, write, err := parseAndCleanDSN(test.dsn)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCleanDSN, clean)
				assert.Equal(t, test.expectRead, read)
				assert.Equal(t, test.expectWrite, write)
			}
		})
	}

}
