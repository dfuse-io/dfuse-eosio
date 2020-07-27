package migrator

import (
	"github.com/eoscanada/eos-go"
)

type permissionObject struct {
	// Parent of this permission object
	Parent eos.PermissionName `json:"parent,omitempty"`
	// Owner is the account for which this permission belongs to
	Owner eos.AccountName `json:"owner,omitempty"`
	// Name is the permission's name this permission object is known as (human-readable name for the permission)
	Name eos.PermissionName `json:"name,omitempty"`
	// Authority required to execute this permission
	Authority *eos.Authority `json:"authority,omitempty"`
}

type LinkAuth struct {
	Permission string `json:"permission"`
	Contract   string `json:"contract"`
	Action     string `json:"action"`
}

type AccountInfo struct {
	Permissions []*permissionObject `json:"permissions"`
	LinkAuths   []*LinkAuth         `json:"link_auths"`

	nameToPerm map[eos.PermissionName]*permissionObject
}

func newAccountInfo(permissions []*permissionObject, linkAuths []*LinkAuth) *AccountInfo {
	info := &AccountInfo{
		Permissions: permissions,
		LinkAuths:   linkAuths,
	}
	info.setupIDtoPerm()
	return info
}

func (a *AccountInfo) setupIDtoPerm() {
	a.nameToPerm = make(map[eos.PermissionName]*permissionObject, len(a.Permissions))
	for _, perm := range a.Permissions {
		a.nameToPerm[perm.Name] = perm
	}
}

func (a *AccountInfo) sortPermissions() (out []*permissionObject) {
	var roots []*permissionObject
	parentToChildren := map[eos.PermissionName][]*permissionObject{}
	for _, perm := range a.Permissions {
		if perm.Owner == "" {
			roots = append(roots, perm)
			continue
		}

		parentToChildren[perm.Parent] = append(parentToChildren[perm.Parent], perm)
	}

	var walk func(roots []*permissionObject, index int)
	walk = func(roots []*permissionObject, index int) {
		if index >= len(roots) {
			return
		}
		ele := roots[index]
		out = append(out, ele)

		for _, child := range parentToChildren[ele.Name] {
			roots = append(roots, child)
		}
		index = index + 1
		walk(roots, index)
	}

	walk(roots, 0)

	return out
}
