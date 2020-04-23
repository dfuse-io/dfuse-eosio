package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//"/storage/megered-blocks"
//:// -> assume passit dreiclty
//NO -> "/" directly
//relative + datadir

func Test_getDirsToMake(t *testing.T) {
	tests := []struct {
		name       string
		storeURL   string
		expectDirs []string
	}{
		{
			name:       "google storage path",
			storeURL:   "gs://test-bucket/eos-local/v1",
			expectDirs: nil,
		},
		{
			name:       "relative local path",
			storeURL:   "myapp/blocks",
			expectDirs: []string{"myapp/blocks"},
		},
		{
			name:       "absolute local path",
			storeURL:   "/data/myapp/blocks",
			expectDirs: []string{"/data/myapp/blocks"},
		},
	}

	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			assert.Equal(t, test.expectDirs, getDirsToMake(test.storeURL))
		})
	}

}
