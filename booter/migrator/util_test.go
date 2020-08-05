package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot_extractIndexNumber(t *testing.T) {
	tests := []struct {
		name        string
		tableName   string
		expectTable string
		expectIndex uint64
		expectPanic bool
	}{
		{
			name:        "table @ index 0",
			tableName:   "sk.multi",
			expectIndex: 0,
			expectTable: "sk.multi",
		},
		{
			name:        "table with index 3",
			tableName:   "sk.multi....3",
			expectIndex: 3,
			expectTable: "sk.multi",
		},
		{
			name:        "table with index 8",
			tableName:   "sk.multi....c",
			expectIndex: 8,
			expectTable: "sk.multi",
		},
		{
			name:        "table with index 13",
			tableName:   "sk.multi....h",
			expectIndex: 13,
			expectTable: "sk.multi",
		},
		{
			name:        "table with index 15",
			tableName:   "sk.multi....j",
			expectIndex: 15,
			expectTable: "sk.multi",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() { recover() }()
			table, index := mustExtractIndexNumber(test.tableName)
			if test.expectPanic {
				t.Errorf("%s did not panic", test.name)
				return
			}
			assert.Equal(t, test.expectTable, table)
			assert.Equal(t, test.expectIndex, index)
		})

	}
}

func TestSnapshot_createIndexTable(t *testing.T) {
	tests := []struct {
		name        string
		tableName   string
		indexId     uint64
		expectTable string
	}{
		{
			name:        "index 0",
			tableName:   "sk.multi",
			indexId:     0,
			expectTable: "sk.multi",
		},
		{
			name:        "index 3",
			tableName:   "sk.multi",
			indexId:     3,
			expectTable: "sk.multi....3",
		},
		{
			name:        "index 8",
			tableName:   "sk.multi",
			indexId:     8,
			expectTable: "sk.multi....c",
		},
		{
			name:        "index 13",
			tableName:   "sk.multi",
			indexId:     13,
			expectTable: "sk.multi....h",
		},
		{
			name:        "index 15",
			tableName:   "sk.multi",
			indexId:     15,
			expectTable: "sk.multi....j",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() { recover() }()
			table := mustCreateIndexTable(test.tableName, test.indexId)
			assert.Equal(t, test.expectTable, table)
		})

	}
}
