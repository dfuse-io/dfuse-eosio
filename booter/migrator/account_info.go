package migrator

import pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"

type linkAuth struct {
	Permission string `json:"permission"`
	Contract   string `json:"contract"`
	Action     string `json:"action"`
}

type accountInfo struct {
	Permissions []*pbcodec.PermissionObject `json:"permissions"`
	LinkAuths   []*linkAuth                 `json:"link_auths"`

	idToPerm map[uint64]*pbcodec.PermissionObject
}

func newAccountInfo(permissions []*pbcodec.PermissionObject, linkAuths []*linkAuth) *accountInfo {
	info := &accountInfo{
		Permissions: permissions,
		LinkAuths:   linkAuths,
	}
	info.setupIDtoPerm()
	return info
}

func (a *accountInfo) setupIDtoPerm() {
	a.idToPerm = make(map[uint64]*pbcodec.PermissionObject, len(a.Permissions))
	for _, perm := range a.Permissions {
		a.idToPerm[perm.Id] = perm
	}
}

func (a *accountInfo) getParent(child *pbcodec.PermissionObject) (parent *pbcodec.PermissionObject) {
	if child.ParentId == 0 {
		return nil
	}

	return a.idToPerm[child.ParentId]
}

func (a *accountInfo) sortPermissions() (out []*pbcodec.PermissionObject) {
	var roots []*pbcodec.PermissionObject
	parentToChildren := map[uint64][]*pbcodec.PermissionObject{}
	for _, perm := range a.Permissions {
		if perm.ParentId == 0 {
			roots = append(roots, perm)
			continue
		}

		parentToChildren[perm.ParentId] = append(parentToChildren[perm.ParentId], perm)
	}

	var walk func(roots []*pbcodec.PermissionObject, index int)
	walk = func(roots []*pbcodec.PermissionObject, index int) {
		if index >= len(roots) {
			return
		}
		ele := roots[index]
		out = append(out, ele)

		for _, child := range parentToChildren[ele.Id] {
			roots = append(roots, child)
		}
		index = index + 1
		walk(roots, index)
	}

	walk(roots, 0)

	return out
}
