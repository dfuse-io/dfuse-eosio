package eosq

import (
	"testing"

	"gotest.tools/assert"
)

func TestSanitizeAPIEndpoint(t *testing.T) {

	cases := []struct {
		name         string
		in           string
		expectHost   string
		expectSecure bool
	}{
		{
			name:         "port only",
			in:           ":8080",
			expectHost:   "localhost:8080",
			expectSecure: false,
		},
		{
			name:         "no protocol",
			in:           "localhost",
			expectHost:   "localhost",
			expectSecure: false,
		},
		{
			name:         "no protocol with port",
			in:           "localhost:8080",
			expectHost:   "localhost:8080",
			expectSecure: false,
		},
		{
			name:         "https with port",
			in:           "https://my.domain:8443",
			expectHost:   "my.domain:8443",
			expectSecure: true,
		},
		{
			name:         "https",
			in:           "https://my.domain",
			expectHost:   "my.domain",
			expectSecure: true,
		},
		{
			name:         "http with sub",
			in:           "http://my.domain/sub",
			expectHost:   "my.domain/sub",
			expectSecure: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host, secure := sanitizeAPIEndpoint(c.in)

			assert.Equal(t, c.expectHost, host)
			assert.Equal(t, c.expectSecure, secure)

		})
	}

}
