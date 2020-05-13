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
func TestNodeosVersion_NewFromString(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    nodeosVersion
		expectedErr error
	}{
		{"standard no suffix", "v2.0.5", nodeosVersion{"v2.0.5", 2, 0, 5, ""}, nil},
		{"standard with suffix then dash", "v2.0.5-beta-1", nodeosVersion{"v2.0.5-beta-1", 2, 0, 5, "beta-1"}, nil},
		{"standard with suffix then dot", "v2.0.5-rc.1", nodeosVersion{"v2.0.5-rc.1", 2, 0, 5, "rc.1"}, nil},
		{"standard with suffix with number", "v2.0.5-rc1", nodeosVersion{"v2.0.5-rc1", 2, 0, 5, "rc1"}, nil},
		{"standard with dm suffix, dash", "v2.0.5-dm-12.0", nodeosVersion{"v2.0.5-dm-12.0", 2, 0, 5, "dm-12.0"}, nil},
		{"standard with dm suffix, dot", "v2.0.5-dm.12.0", nodeosVersion{"v2.0.5-dm.12.0", 2, 0, 5, "dm.12.0"}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := newNodeosVersionFromString(test.in)
			if test.expectedErr == nil {
				assert.Equal(t, test.expected, actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}

func TestNodeosVersion_SupportsDeepMind(t *testing.T) {
	tests := []struct {
		name         string
		version      nodeosVersion
		majorVersion int
		expected     bool
	}{
		{"with suffix, major fit", nodeosVersion{"", 2, 0, 5, "dm.12.0"}, 12, true},
		{"with suffix, major fit, minor higher", nodeosVersion{"", 2, 0, 5, "dm.12.1"}, 12, true},

		{"no suffix", nodeosVersion{"", 2, 0, 5, ""}, 12, false},
		{"invalid suffix", nodeosVersion{"", 2, 0, 5, "rc-1"}, 12, false},
		{"invalid suffix includes major", nodeosVersion{"", 2, 0, 5, ".12.0"}, 12, false},
		{"with suffix, major lower, minor 0", nodeosVersion{"", 2, 0, 5, "dm.11.1"}, 12, false},
		{"with suffix, major lower, minor higher than major", nodeosVersion{"", 2, 0, 5, "dm.11.12"}, 12, false},
		{"with suffix, major higher, minor 0", nodeosVersion{"", 2, 0, 5, "dm.13.0"}, 12, false},
		{"with suffix, major higher, minor higher than major", nodeosVersion{"", 2, 0, 5, "dm.13.12"}, 12, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.version.supportsDeepMind(test.majorVersion), "Version suffix %s does not support deep mind major version %d", test.version.suffix, test.majorVersion)
		})
	}
}
