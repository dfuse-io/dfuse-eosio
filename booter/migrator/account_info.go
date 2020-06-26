package migrator

import pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

// account.json
/*
{
	permissions: [
		{ name: "owner", owner: "", authoriy: obcode.Authority }
	],
}
*/

type linkAuth struct {
	Permission string `json:"permission"`
	Contract   string `json:"contract"`
	Action     string `json:"action"`
}

type accountInfo struct {
	Permissions []pbcodec.PermissionObject `json:"permissions"`
	LinkAuths   []*linkAuth                `json:"link_auths"`
}

func (a *accountInfo) sortPermissions() (out []pbcodec.PermissionObject) {
	perms := map[string][]pbcodec.PermissionObject{}
	for _, perm := range a.Permissions {
		// TODO: can we have 2 permissions without an Owner?
		if perm.Owner == "" {
			// the permission which doesn't have an
			//owner (a.k.a parent) is the EOS owner permissions
			out = append(out, perm)
			continue
		}
		if _, found := perms[perm.Owner]; found {
			perms[perm.Owner] = append(perms[perm.Owner], perm)
			continue
		}
		perms[perm.Owner] = []pbcodec.PermissionObject{perm}
	}

	keys := []string{"owner"}
	for len(keys) > 0 {
		for _, key := range keys {
			for _, permission := range perms[key] {
				out = append(out, permission)
				keys = append(keys, permission.Name)
			}
			keys = keys[1:]
		}
	}
	return out
}
