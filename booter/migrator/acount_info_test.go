package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

func TestAccountInfo_sortPermissions(t *testing.T) {
	tests := []struct {
		name   string
		in     []*pbcodec.PermissionObject
		expect []*pbcodec.PermissionObject
	}{
		{
			name: "sorted owner and active",
			in: []*pbcodec.PermissionObject{
				{Id: 1, ParentId: 0, Name: "owner"},
				{Id: 2, ParentId: 1, Name: "active"},
			},
			expect: []*pbcodec.PermissionObject{
				{Id: 1, ParentId: 0, Name: "owner"},
				{Id: 2, ParentId: 1, Name: "active"},
			},
		},
		{
			name: "un-sorted owner and active",
			in: []*pbcodec.PermissionObject{
				{Id: 2, ParentId: 1, Name: "active"},
				{Id: 1, ParentId: 0, Name: "owner"},
			},
			expect: []*pbcodec.PermissionObject{
				{Id: 1, ParentId: 0, Name: "owner"},
				{Id: 2, ParentId: 1, Name: "active"},
			},
		},
		{
			name: " complex tree",
			in: []*pbcodec.PermissionObject{
				{Id: 21, ParentId: 12, Name: "transfers"},
				{Id: 1, ParentId: 0, Name: "owner"},
				{Id: 20, ParentId: 11, Name: "purger"},
				{Id: 30, ParentId: 20, Name: "purgees"},
				{Id: 10, ParentId: 2, Name: "claimer"},
				{Id: 2, ParentId: 1, Name: "active"},
				{Id: 11, ParentId: 2, Name: "blacklistops"},
				{Id: 12, ParentId: 2, Name: "day2day"},
			},
			expect: []*pbcodec.PermissionObject{
				{Id: 1, ParentId: 0, Name: "owner"},
				{Id: 2, ParentId: 1, Name: "active"},
				{Id: 10, ParentId: 2, Name: "claimer"},
				{Id: 11, ParentId: 2, Name: "blacklistops"},
				{Id: 12, ParentId: 2, Name: "day2day"},
				{Id: 20, ParentId: 11, Name: "purger"},
				{Id: 21, ParentId: 12, Name: "transfers"},
				{Id: 30, ParentId: 20, Name: "purgees"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			acc := newAccountInfo(test.in, nil)
			assert.Equal(t, test.expect, acc.sortPermissions())
		})
	}

}
