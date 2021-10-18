package eosio

import (
	"testing"
	"unicode/utf8"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
)

func TestLimitConsoleLengthConversionOption(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		maxByteCount int
		expected     string
	}{
		{"one extra requires truncation, unicode (1 byte)", "abc", 2, "ab"},
		{"exact flush no truncation, unicode (1 byte)", "abc", 3, "abc"},

		{"one extra requires truncation, unicode (multi-byte)", "我我我", 5, "我"},
		{"exact flush no truncation, unicode (multi-byte)", "我我我", 6, "我我"},

		{"truncate before valid multi-byte utf8, nothing before", "🚀", 4, "🚀"},
		{"truncate at 3 before valid multi-byte utf8, nothing before", "🚀", 3, ""},
		{"truncate at 2 before valid multi-byte utf8, nothing before", "🚀", 2, ""},
		{"truncate at 1 before valid multi-byte utf8, nothing before", "🚀", 1, ""},

		{"truncate before valid multi-byte utf8, something before", "我🚀", 7, "我🚀"},
		{"truncate at 3 before valid multi-byte utf8, something before", "我🚀", 6, "我"},
		{"truncate at 2 before valid multi-byte utf8, something before", "我🚀", 5, "我"},
		{"truncate at 1 before valid multi-byte utf8, something before", "我🚀", 4, "我"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actTrace := &pbcodec.ActionTrace{Console: test.in}

			option := LimitConsoleLengthConversionOption(test.maxByteCount)
			option.(ActionConversionOption).Apply(actTrace)

			assert.Equal(t, test.expected, actTrace.Console)
			assert.True(t, utf8.ValidString(actTrace.Console), "The truncated string is not a fully valid utf-8 sequence")
		})
	}
}
