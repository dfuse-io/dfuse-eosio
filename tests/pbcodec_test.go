package tests

import (
	"fmt"
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
)

// Those tests requires `codec/testing` package which depends on `codec`, so they
// cannot be put inside `codec`. They also cannot be put inside `pb/codec` because that
// causes a cycle because `codec` uses `pb/codec` which would use `codec/testing` which
// uses `codec`.
//
// Easiest solution was to find a "third-party" package to host those tests, here we are.

func TestFilteringActionMatcher(t *testing.T) {
	newAccount := []pbcodec.ActionMatcher{
		func(actTrace *pbcodec.ActionTrace) bool {
			fmt.Printf("%#v\n", actTrace)
			return actTrace.Receiver == "eosio" && actTrace.Action.Account == "eosio" && actTrace.Action.Name == "newaccount"
		},
	}

	tests := []struct {
		name                 string
		block                *pbcodec.Block
		requireSystemActions []pbcodec.ActionMatcher
		expectedMatch        []uint32
		expectedNotMatch     []uint32
	}{
		{
			"none matching",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "match:match:zero"),
				ct.ActionTrace(t, "match:match:first"),
			)),
			nil,
			nil,
			[]uint32{0, 1},
		},
		{
			"some matching",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "match:match:zero", ct.ActionMatched),
				ct.ActionTrace(t, "nonmatch:nonmatch:middle"),
				ct.ActionTrace(t, "match:match:first", ct.ActionMatched),
			)),
			nil,
			[]uint32{0, 2},
			[]uint32{1},
		},
		{
			"all matching",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "match:match:zero", ct.ActionMatched),
				ct.ActionTrace(t, "match:match:first", ct.ActionMatched),
			)),
			nil,
			[]uint32{0, 1},
			nil,
		},

		{
			"block is unfiltered, everything is included",
			ct.Block(t, "00000001aa", ct.TrxTrace(t,
				ct.ActionTrace(t, "match:match:zero"),
				ct.ActionTrace(t, "match:match:first", ct.ActionMatched),
			)),
			nil,
			[]uint32{0, 1},
			nil,
		},

		{
			"is required system actions but was not system included, still match",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched),
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionMatched),
			)),
			newAccount,
			[]uint32{0, 1},
			nil,
		},
		{
			"is required system actions and was system included, still match",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionSystemMatched),
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionSystemMatched),
			)),
			newAccount,
			[]uint32{0, 1},
			nil,
		},
		{
			"was system included but is not a required actions when empty, does not match",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionSystemMatched),
				ct.ActionTrace(t, "eosio:eosio:newaccount", ct.ActionSystemMatched),
			)),
			nil,
			nil,
			[]uint32{0, 1},
		},
		{
			"was system included but is not a required actions not matching, does not match",
			ct.Block(t, "00000001aa", ct.FilteredBlock{}, ct.TrxTrace(t,
				ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionSystemMatched),
				ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionSystemMatched),
			)),
			newAccount,
			nil,
			[]uint32{0, 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher := test.block.FilteringActionMatcher(test.block.TransactionTraces()[0], test.requireSystemActions...)

			for _, expectedMatch := range test.expectedMatch {
				assert.True(t, matcher.Matched(expectedMatch), "Expecting action index %d to be included in matcher, but it was not", expectedMatch)
			}

			for _, expectedNotMatch := range test.expectedNotMatch {
				assert.False(t, matcher.Matched(expectedNotMatch), "Expecting action index %d to be excluded from matcher, but it was not", expectedNotMatch)
			}
		})
	}
}
