package filtering

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
