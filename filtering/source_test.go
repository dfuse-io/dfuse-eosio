package filtering

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteringPreprocessor(t *testing.T) {
	tests := []struct {
		name     string
		exprs    filters
		block    *pbcodec.Block
		expected *pbcodec.Block
	}{
		{
			"standard",
			getFilters("*", `receiver == "spamcoint"`, ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// We assign them manually in the expected block to keep them in sync with ther `include`, `exclude` test parameters
			test.expected.FilteringIncludeFilterExpr = test.exprs.include[0]
			test.expected.FilteringExcludeFilterExpr = test.exprs.exclude[0]

			filter, err := NewBlockFilter(test.exprs.include, test.exprs.exclude, test.exprs.system)
			require.NoError(t, err)

			preprocessor := &FilteringPreprocessor{Filter: filter}
			blk := ct.ToBstreamBlock(t, test.block)

			_, err = preprocessor.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected, blk.ToNative().(*pbcodec.Block))
		})
	}
}

func TestFilteringTwice(t *testing.T) {
	tests := []struct {
		name                      string
		exprs1                    filters
		exprs2                    filters
		block                     *pbcodec.Block
		expected                  *pbcodec.Block
		shouldPanicOnSecondFilter bool
	}{
		{
			"standard",
			getFilters("*", `receiver == "spamcoint"`, ""),
			getFilters("*", `receiver == "spamcoint"`, ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			false,
		},
		{
			"panicky",
			getFilters("*", `receiver == "spamcoint"`, ""),
			getFilters("*", `receiver == "spamcoin"`, ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			defer func() { recover() }()
			// We assign them manually in the expected block to keep them in sync with ther `include`, `exclude` test parameters
			test.expected.FilteringIncludeFilterExpr = test.exprs1.include[0]
			test.expected.FilteringExcludeFilterExpr = test.exprs1.exclude[0]

			filter, err := NewBlockFilter(test.exprs1.include, test.exprs1.exclude, test.exprs1.system)
			require.NoError(t, err)

			preprocessor := &FilteringPreprocessor{Filter: filter}
			blk := ct.ToBstreamBlock(t, test.block)

			_, err = preprocessor.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected, blk.ToNative().(*pbcodec.Block))

			filter2, err := NewBlockFilter(test.exprs2.include, test.exprs2.exclude, test.exprs2.system)
			require.NoError(t, err)

			preprocessor2 := &FilteringPreprocessor{Filter: filter2}

			_, err = preprocessor2.PreprocessBlock(blk)
			if test.shouldPanicOnSecondFilter {
				t.Errorf("%s did not panic", test.name)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, test.expected, blk.ToNative().(*pbcodec.Block))
		})
	}
}
