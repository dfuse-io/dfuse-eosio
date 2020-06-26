package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

func TestAccountInfo_sortPermissions(t *testing.T) {
	tests := []struct {
		name   string
		in     []pbcodec.PermissionObject
		expect []pbcodec.PermissionObject
	}{
		{
			name:   "empty arrays",
			in:     []pbcodec.PermissionObject{},
			expect: []pbcodec.PermissionObject{},
		},

		{
			name: "sorted owner and active",
			in: []pbcodec.PermissionObject{
				{Owner: "", Name: "owner"},
				{Owner: "owner", Name: "active"},
			},
			expect: []pbcodec.PermissionObject{
				{Owner: "", Name: "owner"},
				{Owner: "owner", Name: "active"},
			},
		},
		{
			name: "un-sorted owner and active",
			in: []pbcodec.PermissionObject{
				{Owner: "owner", Name: "active"},
				{Owner: "", Name: "owner"},
			},
			expect: []pbcodec.PermissionObject{
				{Owner: "", Name: "owner"},
				{Owner: "owner", Name: "active"},
			},
		},
		{
			name: " complex tree",
			in: []pbcodec.PermissionObject{
				{Owner: "day2day", Name: "transfers"},
				{Owner: "", Name: "owner"},
				{Owner: "blacklistops", Name: "purger"},
				{Owner: "purger", Name: "purgees"},
				{Owner: "active", Name: "claimer"},
				{Owner: "owner", Name: "active"},
				{Owner: "active", Name: "blacklistops"},
				{Owner: "active", Name: "day2day"},
			},
			expect: []pbcodec.PermissionObject{
				{Owner: "", Name: "owner"},
				{Owner: "owner", Name: "active"},
				{Owner: "active", Name: "claimer"},
				{Owner: "active", Name: "blacklistops"},
				{Owner: "active", Name: "day2day"},
				{Owner: "blacklistops", Name: "purger"},
				{Owner: "day2day", Name: "transfers"},
				{Owner: "purger", Name: "purgees"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			acc := &accountInfo{
				Permissions: test.in,
			}
			assert.ElementsMatch(t, test.expect, acc.sortPermissions())
		})
	}

}
