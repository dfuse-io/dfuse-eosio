package filtering

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteringPreprocessor(t *testing.T) {
	block1 := ct.Block(t, "00000001aa",
		ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount"), ct.ActionTrace(t, "spamcoint:spamcoint:transfer")),
	)

	expected1 := ct.Block(t, "00000001aa",
		ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:newaccount")),
	)

	expected1.FilteringKind = pbcodec.FilteringKind_FILTERINGKIND_FILTERED
	expected1.FilteringMetadata = &pbcodec.FilteringMetadata{
		CelFilterIn:  "",
		CelFilterOut: `receiver == "spamcoint"`,
	}

	tests := []struct {
		name     string
		in       *pbcodec.Block
		expected *pbcodec.Block
	}{
		{"standard", block1, expected1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter, err := NewBlockFilter("", `receiver == "spamcoint"`)
			require.NoError(t, err)

			preprocessor := &FilteringPreprocessor{Filter: filter}
			blk := ct.ToBstreamBlock(t, test.in)

			_, err = preprocessor.PreprocessBlock(blk)
			require.NoError(t, err)

			assert.Equal(t, test.expected, blk.ToNative().(*pbcodec.Block))
		})
	}
}
