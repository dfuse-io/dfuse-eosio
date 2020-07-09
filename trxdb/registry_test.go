package trxdb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func init() {
	Register("aaa", func(dsn []string) (DB, error) {
		return nil, nil
	})

	Register("bbb", func(dsn []string) (DB, error) {
		return nil, nil
	})
}

func Test_splitDSN(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectDsns  []string
		expectError bool
	}{
		{
			name:       "sunny path",
			dsn:        "aaa://path?foo=bar",
			expectDsns: []string{"aaa://path?foo=bar"},
		},
		{
			name:       "two DSN",
			dsn:        "aaa://path?foo=bar aaa://secondPath?foo2=bar3",
			expectDsns: []string{"aaa://path?foo=bar", "aaa://secondPath?foo2=bar3"},
		},
		{
			name:        "two different dsn",
			dsn:         "aaa://path?foo=bar bbb://secondPath?foo2=bar3",
			expectError: true,
		},
		{
			name:        "no dsn",
			dsn:         "",
			expectError: true,
		},
		{
			name:        "invalid dsn",
			dsn:         "driverpath?foor=bar",
			expectError: true,
		},
		{
			name:        "invalid second dsn",
			dsn:         "aaa://path?foo=bar bbb?foor=bar",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsns, _, err := splitDsn(test.dsn)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectDsns, dsns)
			}
		})
	}

}
