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
		name      string
		exprs1    filters
		exprs2    filters
		block     *pbcodec.Block
		expected1 *pbcodec.Block
		expected2 *pbcodec.Block
	}{
		{
			"same",
			getFilters("*", `receiver == "spamcoint"`, ""),
			getFilters("*", `receiver == "spamcoint"`, ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint"`,
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint"`,
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
		},
		{
			"different",
			getFilters("*", `receiver == "spamcoint"`, ""),
			getFilters("", `receiver == "spamcoin"`, ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint"`,
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint";;;receiver == "spamcoin"`,
				UnfilteredStats: ct.Counts{2, 2, 2},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
		},
		{
			"no-reapply", // we cheat a bit here, we pretend spamcoint was removed, but we keep it
			// this way we know that it does not try to reapply
			getFilters("*", `receiver == "spamcoint"`, ""),
			getFilters("*", `receiver == "eosio"`, ""),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint"`,
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint"`,
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         "*",
				Exclude:         `receiver == "spamcoint";;;receiver == "eosio"`,
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer", ct.ActionMatched)),
			),
		},
		{
			"include filter works on filtered block",
			getFilters(`action == "transfer"`, "", ""),
			getFilters(`receiver == "spamcoin"`, "", ""),
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer")),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:eosio.token:transfer", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         `action == "transfer"`,
				Exclude:         "",
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:eosio.token:transfer", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         `action == "transfer";;;receiver == "spamcoin"`,
				Exclude:         "",
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{1, 1, 1},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer", ct.ActionMatched)),
			),
		},
		{
			"empty filter works on filtered block",
			getFilters("*", "false", "false"),
			getFilters("", "", ""),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         `action == "transfer"`,
				Exclude:         "",
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:eosio.token:transfer", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         `action == "transfer"`,
				Exclude:         "",
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:eosio.token:transfer", ct.ActionMatched)),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{
				Include:         `action == "transfer"`,
				Exclude:         "",
				UnfilteredStats: ct.Counts{3, 3, 3},
				FilteredStats:   ct.Counts{2, 2, 2},
			},
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoin:spamcoin:transfer", ct.ActionMatched)),
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:eosio.token:transfer", ct.ActionMatched)),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			filter, err := NewBlockFilter(test.exprs1.include, test.exprs1.exclude, test.exprs1.system)
			require.NoError(t, err)

			preprocessor := &FilteringPreprocessor{Filter: filter}
			blk := ct.ToBstreamBlock(t, test.block)

			_, err = preprocessor.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected1, blk.ToNative().(*pbcodec.Block))

			filter2, err := NewBlockFilter(test.exprs2.include, test.exprs2.exclude, test.exprs2.system)
			require.NoError(t, err)

			preprocessor2 := &FilteringPreprocessor{Filter: filter2}

			_, err = preprocessor2.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected2, blk.ToNative().(*pbcodec.Block))
		})
	}
}
