package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//"/storage/megered-blocks"
//:// -> assume passit dreiclty
//NO -> "/" directly
//relative + datadir

func Test_buildStoreURL(t *testing.T) {
	tests := []struct {
		name           string
		dataDir        string
		storeURL       string
		expectStoreURL string
	}{
		{
			name:           "google storage path",
			dataDir:        "/Users/john/.dfusebox",
			storeURL:       "gs://test-bucket/eos-local/v1",
			expectStoreURL: "gs://test-bucket/eos-local/v1",
		},
		{
			name:           "absolute local path",
			dataDir:        "/Users/john/.dfusebox",
			storeURL:       "/Users/john/data/app",
			expectStoreURL: "/Users/john/data/app",
		},
		{
			name:           "absolute local path",
			dataDir:        "/Users/john/.dfusebox",
			storeURL:       "app/storage/blocks",
			expectStoreURL: "/Users/john/.dfusebox/app/storage/blocks",
		},
	}

	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			assert.Equal(t, test.expectStoreURL, buildStoreURL(test.dataDir, test.storeURL))
		})
	}

}
