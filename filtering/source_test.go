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
		name             string
		include, exclude string
		block            *pbcodec.Block
		expected         *pbcodec.Block
	}{
		{
			"standard", "*", `receiver == "spamcoint"`,
			ct.Block(t, "00000001aa",
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
				ct.TrxTrace(t, ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
			),
			ct.Block(t, "00000001aa", ct.FilteredBlock{Stats: ct.UnfilteredStats{2, 2, 2}},
				ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched)),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// We assign them manually in the expected block to keep them in sync with ther `include`, `exclude` test parameters
			test.expected.FilteringIncludeFilterExpr = test.include
			test.expected.FilteringExcludeFilterExpr = test.exclude

			filter, err := NewBlockFilter(test.include, test.exclude)
			require.NoError(t, err)

			preprocessor := &FilteringPreprocessor{Filter: filter}
			blk := ct.ToBstreamBlock(t, test.block)

			_, err = preprocessor.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected, blk.ToNative().(*pbcodec.Block))
		})
	}
}
