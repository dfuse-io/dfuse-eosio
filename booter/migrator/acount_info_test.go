package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountInfo_sortPermissions(t *testing.T) {
	tests := []struct {
		name   string
		in     []*permissionObject
		expect []*permissionObject
	}{
		{
			name: "sorted owner and active",
			in: []*permissionObject{
				{Parent: "", Owner: "battlefield1", Name: "owner"},
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
			},
			expect: []*permissionObject{
				{Parent: "", Owner: "battlefield1", Name: "owner"},
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
			},
		},
		{
			name: "un-sorted owner and active",
			in: []*permissionObject{
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
				{Parent: "", Owner: "battlefield1", Name: "owner"},
			},
			expect: []*permissionObject{
				{Parent: "", Owner: "battlefield1", Name: "owner"},
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
			},
		},
		{
			name: " complex tree",
			in: []*permissionObject{
				{Parent: "day2day", Owner: "battlefield1", Name: "transfers"},
				{Parent: "", Owner: "battlefield1", Name: "owner"},
				{Parent: "blacklistops", Owner: "battlefield1", Name: "purger"},
				{Parent: "purger", Owner: "battlefield1", Name: "purgees"},
				{Parent: "active", Owner: "battlefield1", Name: "claimer"},
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
				{Parent: "active", Owner: "battlefield1", Name: "blacklistops"},
				{Parent: "active", Owner: "battlefield1", Name: "day2day"},
			},
			expect: []*permissionObject{
				{Parent: "", Owner: "battlefield1", Name: "owner"},
				{Parent: "owner", Owner: "battlefield1", Name: "active"},
				{Parent: "active", Owner: "battlefield1", Name: "claimer"},
				{Parent: "active", Owner: "battlefield1", Name: "blacklistops"},
				{Parent: "active", Owner: "battlefield1", Name: "day2day"},
				{Parent: "blacklistops", Owner: "battlefield1", Name: "purger"},
				{Parent: "day2day", Owner: "battlefield1", Name: "transfers"},
				{Parent: "purger", Owner: "battlefield1", Name: "purgees"},
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
