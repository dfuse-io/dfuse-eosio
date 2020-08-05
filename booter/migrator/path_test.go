package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newAccountPath(t *testing.T) {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := newAccountPath("dir", test.in)
			require.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_nestedPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		entityName string
		expected   string
	}{
		{"single char", "dir", "a", "dir/a"},
		{"two chars", "dir", "ab", "dir/ab"},
		{"three chars", "dir", "abc", "dir/ab/abc"},
		{"four chars", "dir", "abcd", "dir/ab/abcd"},
		{"five chars", "dir", "abcde", "dir/ab/cd/abcde"},
		{"nine chars", "dir", "abcdefghi", "dir/ab/cd/abcdefghi"},
		{"twelve chars", "dir", "abcdefghijkl", "dir/ab/cd/abcdefghijkl"},
		{"thirteen chars", "dir", "abcdefghijklm", "dir/ab/cd/abcdefghijklm"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.expected, nestedPath(test.path, test.entityName))
		})
	}
}
