package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_encodeName(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		expectOut string
	}{
		{
			name:      "clean name",
			in:        "battlefield1",
			expectOut: "battlefield1",
		},
		{
			name:      "dirty name",
			in:        "........ehbo5",
			expectOut: "________ehbo5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectOut, encodeName(test.in))
		})
	}
}

func Test_decodeName(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		expectOut string
	}{
		{
			name:      "clean name",
			in:        "battlefield1",
			expectOut: "battlefield1",
		},
		{
			name:      "dirty name",
			in:        "________ehbo5",
			expectOut: "........ehbo5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectOut, decodeName(test.in))
		})
	}
}
