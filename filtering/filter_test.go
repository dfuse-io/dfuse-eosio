package filtering

import (
	"fmt"
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockFilter(t *testing.T) {
	filterMatched := true
	filterDidNotMatch := false

	tests := []struct {
		name           string
		exprs          filters
		trace          *pbcodec.TransactionTrace
		expectedPass   bool
		expectedSystem bool
	}{
		{
			"filter nothing",
			filters{"", "", ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "whatever:action")),
			filterMatched, false,
		},
		{
			"filter nothing, with default programs",
			filters{"true", "false", ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "whatever:action")),
			filterMatched, false,
		},
		{
			"blacklist things FROM badguy",
			filters{`true`, `account == "eosio.token" && data.from == "badguy"`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:transfer", ct.ActionData(`{"from":"goodguy","to":"badguy"}`))),
			filterMatched, false,
		},
		{
			"blacklist things TO badguy",
			filters{`true`, "account == 'eosio.token' && data.to == 'badguy'", ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:transfer", ct.ActionData(`{"from":"goodguy","to":"badguy"}`))),
			filterDidNotMatch, false,
		},
		{
			"blacklist transfers to eidosonecoin",
			filters{
				"*",
				`account == 'eidosonecoin' || receiver == 'eidosonecoin' || (account == 'eosio.token' && (data.to == 'eidosonecoin' || data.from == 'eidosonecoin'))`,
				"",
			},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:transfer", ct.ActionData(`{"from":"goodguy","to":"eidosonecoin"}`))),
			filterDidNotMatch, false,
		},
		{
			"non-matching identifier in exclude-filter program doesn't blacklist",
			filters{"", `account == 'eosio.token' && data.from == 'broken'`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:issue", ct.ActionData(`{"to":"winner"}`))),
			filterMatched, false,
		},
		{
			"a key not found error in include-filter still includes transaction",
			filters{`account == 'eosio.token' && data.bob == 'broken'`, "", ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio.token:issue", ct.ActionData(`{"to":"winner"}`))),
			filterMatched, false,
		},
		{
			"both whitelist and blacklist fail",
			filters{`data.bob == 'broken'`, `data.rita == 'rebroken'`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "any:any", ct.ActionData(`{"denise":"winner"}`))),
			filterMatched, false,
		},
		{
			"whitelisted but blacklist cleans out",
			filters{`data.bob == '1'`, `data.rita == '2'`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "any:any", ct.ActionData(`{"bob":"1","rita":"2"}`))),
			false, false,
		},
		{
			"whitelisted but blacklist broken so doesn't clean out",
			filters{`data.bob == '1'`, `data.broken == 'really'`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "any:any", ct.ActionData(`{"bob":"1"}`))),
			filterMatched, false,
		},

		{
			"block receiver",
			filters{"", `receiver == "badguy"`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "badguy:any:any", ct.ActionData(`{}`))),
			filterDidNotMatch, false,
		},
		{
			"prevent a failure on evaluation, so matches because blacklist fails",
			filters{"", `account == "badacct" && has(data.from) && data.from != "badguy"`, ""},
			ct.TrxTrace(t, ct.ActionTrace(t, "badrecv:badacct:any", ct.ActionData(`{}`))),
			filterMatched, false,
		},

		{
			"system action already included are not flagged as system",
			filters{`action == "setabi"`, ``, `action == "setabi"`},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionData(`{}`))),
			filterMatched, false,
		},
		{
			"system action are included even when not included",
			filters{`action == "anythingelse"`, ``, `action == "setabi"`},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionData(`{}`))),
			filterMatched, true,
		},
		{
			"system action are included even when excluded",
			filters{"*", `action == "setabi"`, `action == "setabi"`},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionData(`{}`))),
			filterMatched, true,
		},
		{
			"system action are included even when excluded and not included",
			filters{`action == "anythingelse"`, `action == "setabi"`, `action == "setabi"`},
			ct.TrxTrace(t, ct.ActionTrace(t, "eosio:eosio:setabi", ct.ActionData(`{}`))),
			filterMatched, true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Len(t, test.trace.ActionTraces, 1, "This test accepts a single action trace per transaction trace currently")

			filter, err := NewBlockFilter(test.exprs.include, test.exprs.exclude, test.exprs.system)
			require.NoError(t, err)

			hasPass, isSystem := filter.shouldProcess(test.trace, test.trace.ActionTraces[0], func() []string { return nil })

			if test.expectedPass {
				assert.True(t, hasPass, "Expected action trace to match filter (%s) but it did not", test.exprs)
			} else {
				assert.False(t, hasPass, "Expected action trace to NOT match filter (%s) but it did", test.exprs)
			}

			if test.expectedSystem {
				assert.True(t, isSystem, "Expected action trace to be system included (%s) but it did not", test.exprs)
			} else {
				assert.False(t, isSystem, "Expected action trace to NOT be system included (%s) but it did", test.exprs)
			}
		})
	}
}

type filters struct {
	include string
	exclude string
	system  string
}

func (f *filters) String() string {
	return fmt.Sprintf("include %s, exclude %s, system %s", f.include, f.exclude, f.system)
}
