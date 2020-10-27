package filtering

import (
	"fmt"
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlocknumBasedChoose(t *testing.T) {
	tests := []struct {
		name                   string
		blocknumBasedCELFilter blocknumBasedCELFilter
		blocknum               uint64
		expectCode             string
	}{
		{
			"single filter applied every height",
			blocknumBasedCELFilter{
				0: simpleFilter("account=='test'"),
			},
			10,
			"account=='test'",
		},
		{
			"second filter applied",
			blocknumBasedCELFilter{
				0:  simpleFilter("account=='filter1'"),
				50: simpleFilter("account=='filter2'"),
			},
			100,
			"account=='filter2'",
		},
		{
			"second filter not applied before",
			blocknumBasedCELFilter{
				0:  simpleFilter("account=='filter1'"),
				50: simpleFilter("account=='filter2'"),
			},
			49,
			"account=='filter1'",
		},
		{
			"second filter applied on boundary inclusive",
			blocknumBasedCELFilter{
				0:    simpleFilter("account=='filter1'"),
				50:   simpleFilter("account=='filter2'"),
				1000: simpleFilter("account=='filter3'"),
			},
			50,
			"account=='filter2'",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chosen := test.blocknumBasedCELFilter.choose(test.blocknum)

			assert.Equal(t, test.expectCode, chosen.code)
		})
	}

}

func simpleFilter(code string) *CELFilter {
	return &CELFilter{
		code: code,
	}
}

func TestParseBlocknumBasedCode(t *testing.T) {

	tests := []struct {
		name           string
		code           string
		expectErr      bool
		expectCode     string
		expectBlocknum uint64
	}{
		{
			"simple, no blocknum",
			"account == 'blah'",
			false,
			"account == 'blah'",
			0,
		},
		{
			"blocknum 0",
			"#0;account == 'blah'",
			false,
			"account == 'blah'",
			0,
		},
		{
			"blocknum 12323",
			"#12323;account == \"bob\"",
			false,
			"account == \"bob\"",
			12323,
		},
		{
			"blocknum 12323, query trimmed",
			"#12323; account == \"bob\"",
			false,
			"account == \"bob\"",
			12323,
		},
		{
			"blocknum spaced invalid",
			" #12323;account=='bob'",
			true,
			"",
			0,
		},
		{
			"blocknum spaced invalid",
			"#12323 ;account=='bob'",
			true,
			"",
			0,
		},
		{
			"too many ;; is OK",
			"#12323;account=='some;thing;'",
			false,
			"account=='some;thing;'",
			12323,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, blocknum, err := parseBlocknumBasedCode(test.code)

			if test.expectErr {
				require.Error(t, err)
				return
			}
			assert.Equal(t, test.expectCode, code)
			assert.Equal(t, test.expectBlocknum, blocknum)
		})
	}

}

func TestBlocknumBasedFilter(t *testing.T) {
	tests := []struct {
		name                string
		inCodes             []string
		expectedBlockFilter blocknumBasedCELFilter
		expectedErr         bool
	}{
		{
			"default filter",
			[]string{""},
			blocknumBasedCELFilter{
				0: &CELFilter{
					code: "",
				},
			},
			false,
		},
		{
			"simple filter",
			[]string{"account == 'test'"},
			blocknumBasedCELFilter{
				0: &CELFilter{
					code: "account == 'test'",
				},
			},
			false,
		},
		{
			"blocknum-bound filter",
			[]string{"#0;account == 'test'"},
			blocknumBasedCELFilter{
				0: &CELFilter{
					code: "account == 'test'",
				},
			},
			false,
		},
		{
			"blocknum-bound filter adds 0",
			[]string{"#2345;account == 'test'"},
			blocknumBasedCELFilter{
				0: &CELFilter{
					code: "",
				},
				2345: &CELFilter{
					code: "account == 'test'",
				},
			},
			false,
		},
		{
			"blocknum-bound multiple filters",
			[]string{"receiver=='test'", "#2345;account=='test'", "#4567;auth.exists(x,x=='test')"},
			blocknumBasedCELFilter{
				0: &CELFilter{
					code: "receiver=='test'",
				},
				2345: &CELFilter{
					code: "account=='test'",
				},
				4567: &CELFilter{
					code: "auth.exists(x,x=='test')",
				},
			},
			false,
		},
		{
			"invalid filter error bubbled up",
			[]string{"receiver=='"},
			nil,
			true,
		},
		{
			"invalid blocknum syntax error bubbled up",
			[]string{"12345;account=='test'"},
			nil,
			true,
		},
		{
			"too many filters with same blocknum error bubbled up",
			[]string{"account=='test'", "receiver=='test'"},
			nil,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filtersMap, err := newCELFilters("", test.inCodes, []string{""}, false)
			if test.expectedErr {
				fmt.Println(err)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, len(test.expectedBlockFilter), len(filtersMap))
			for k, v := range test.expectedBlockFilter {
				thisFilter, ok := filtersMap[k]
				require.True(t, ok)
				assert.Equal(t, v.code, thisFilter.code)
			}
		})
	}
}

func TestCELActivation(t *testing.T) {
	type activationManifest struct {
		actTrace       *pbcodec.ActionTrace
		isScheduled    bool
		trxActionCount int
	}

	shouldMatch := true
	shouldNotMatch := false

	tests := []struct {
		name          string
		code          string
		activation    activationManifest
		expectedMatch bool
	}{
		{
			"auth match, single, single check",
			`auth.exists(x, x == "eosio")`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active")), false, 0},
			shouldMatch,
		},
		{
			"auth match, multi, single check",
			`auth.exists(x, x == "eosio")`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active", "eosio@owner", "other@active")), false, 0},
			shouldMatch,
		},

		{
			"auth match, single, multi check",
			`auth.exists(x, x in ["bob", "eosio"])`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active")), false, 0},
			shouldMatch,
		},
		{
			"auth match, multi, multi check",
			`auth.exists(x, x in ["bob", "other"])`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active", "eosio@owner", "other@active")), false, 0},
			shouldMatch,
		},

		{
			"auth not match, multi, single check",
			`auth.exists(x, x == "different")`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active", "eosio@owner", "other")), false, 0},
			shouldNotMatch,
		},
		{
			"auth not match, multi, multi check",
			`auth.exists(x, x in ["different", "multi"])`,
			activationManifest{ct.ActionTrace(t, "eosio:eosio:onblock", ct.Authorizations("eosio@active", "eosio@owner", "other@active")), false, 0},
			shouldNotMatch,
		},
	}

	for _, test := range tests {
		celFilter, err := newCELFilter("test", test.code, []string{"false", ""}, false)
		require.NoError(t, err)

		activation := actionTraceActivation{
			trace:          test.activation.actTrace,
			trxScheduled:   test.activation.isScheduled,
			trxActionCount: test.activation.trxActionCount,
		}

		t.Run(test.name, func(t *testing.T) {
			matched := celFilter.match(&activation)

			if test.expectedMatch {
				assert.True(t, matched, "Expected action trace to match CEL filter (%s) but it did not", test.code)
			} else {
				assert.False(t, matched, "Expected action trace to NOT match filter (%s) but it did", test.code)
			}
		})
	}
}
