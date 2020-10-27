package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndexedTerms(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    *IndexedTerms
		expectedErr error
	}{
		{"single", "receiver", &IndexedTerms{Receiver: true, Base: map[string]bool{"receiver": true}, Data: map[string]bool{}}, nil},
		{"multiple spaces", "receiver account", &IndexedTerms{Receiver: true, Account: true, Base: map[string]bool{"receiver": true, "account": true}, Data: map[string]bool{}}, nil},
		{"multiple comma", "receiver, account", &IndexedTerms{Receiver: true, Account: true, Base: map[string]bool{"receiver": true, "account": true}, Data: map[string]bool{}}, nil},
		{"data fields", "data.to", &IndexedTerms{Base: map[string]bool{}, Data: map[string]bool{"to": true}}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := NewIndexedTerms(test.in)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}
