package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrater_accountStorage(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{"single char", "a", "dir/a"},
		{"two chars", "ab", "dir/ab"},
		{"three chars", "abc", "dir/ab/abc"},
		{"four chars", "abcd", "dir/ab/abcd"},
		{"five chars", "abcde", "dir/ab/cd/abcde"},
		{"nine chars", "abcdefghi", "dir/ab/cd/abcdefghi"},
		{"twelve chars", "abcdefghijkl", "dir/ab/cd/abcdefghijkl"},
		{"thirteen chars", "abcdefghijklm", "dir/ab/cd/abcdefghijklm"},
	}

	migrater := &migrater{
		exportDir: "dir",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := migrater.accountStorage(test.in)
			assert.Equal(t, test.expected, actual)
		})
	}
}
