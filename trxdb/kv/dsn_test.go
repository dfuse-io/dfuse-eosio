package kv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/streamingfast/logging"
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
		expectOpt      *dsnOptions
		expectError    bool
	}{
		{
			name:           "simple dsn",
			dsn:            "test://host.ca/path",
			expectCleanDSN: "test://host.ca/path",
			expectOpt: &dsnOptions{
				reads:  []string{"all"},
				writes: []string{"all"},
			},
		},
		{
			name:           "dsn with read options dsn",
			dsn:            "test://host.ca/path?read=blk,trx",
			expectCleanDSN: "test://host.ca/path",
			expectOpt: &dsnOptions{
				reads:  []string{"blk", "trx"},
				writes: []string{"all"},
			},
		},
		{
			name:           "dsn with write options dsn",
			dsn:            "test://host.ca/path?write=trx",
			expectCleanDSN: "test://host.ca/path",
			expectOpt: &dsnOptions{
				reads:  []string{"all"},
				writes: []string{"trx"},
			},
		},
		{
			name:           "dsn with read and write options",
			dsn:            "test://host.ca/path?read=blk,trx&write=none",
			expectCleanDSN: "test://host.ca/path",
			expectOpt: &dsnOptions{
				reads:  []string{"blk", "trx"},
				writes: []string{"none"},
			},
		},
		{
			name:           "dsn with read none and write options",
			dsn:            "test://host.ca/path?read=none&write=blk",
			expectCleanDSN: "test://host.ca/path",
			expectOpt: &dsnOptions{
				reads:  []string{"none"},
				writes: []string{"blk"},
			},
		},
		{
			name:           "dsn with only read",
			dsn:            "bigkv://dev.dev/test-trxdb-blocks?createTable=true&read=account,block,timeline,last_written_blk",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-blocks?createTable=true",
			expectOpt: &dsnOptions{
				reads:  []string{"account", "block", "timeline", "last_written_blk"},
				writes: []string{"all"},
			},
		},
		{
			name:           "dsn with read nonde and write options",
			dsn:            "bigkv://dev.dev/test-trxdb-trxs?createTable=true&read=transaction&write=transaction",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-trxs?createTable=true",
			expectOpt: &dsnOptions{
				reads:  []string{"transaction"},
				writes: []string{"transaction"},
			},
		},
		{
			name:           "test with block marker enabled",
			dsn:            "bigkv://dev.dev/test-trxdb-blocks?createTable=true&blk_marker=true&read=account,block,timeline,last_written_blk",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-blocks?createTable=true",
			expectOpt: &dsnOptions{
				reads:  []string{"account", "block", "timeline", "last_written_blk"},
				writes: []string{"all"},
			},
		},
		{
			name:           "test with block marker wrongly enabled, it should be true",
			dsn:            "bigkv://dev.dev/test-trxdb-blocks?createTable=true&blk_marker=test&read=account,block,timeline,last_written_blk",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-blocks?createTable=true",
			expectOpt: &dsnOptions{
				reads:  []string{"account", "block", "timeline", "last_written_blk"},
				writes: []string{"all"},
			},
		},
		{
			name:           "test with block marker disabled",
			dsn:            "bigkv://dev.dev/test-trxdb-blocks?createTable=true&blk_marker=false&read=account,block,timeline,last_written_blk",
			expectCleanDSN: "bigkv://dev.dev/test-trxdb-blocks?createTable=true",
			expectOpt: &dsnOptions{
				reads:  []string{"account", "block", "timeline", "last_written_blk"},
				writes: []string{"all"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clean, opts, err := parseAndCleanDSN(test.dsn)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCleanDSN, clean)
				assert.Equal(t, test.expectOpt, opts)
			}
		})
	}

}
