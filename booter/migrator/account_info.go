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
	idToPerm := make(map[uint64]*pbcodec.PermissionObject, len(permissions))
	for _, perm := range permissions {
		idToPerm[perm.Id] = perm
	}

	return &accountInfo{
		Permissions: permissions,
		LinkAuths:   linkAuths,
		idToPerm:    idToPerm,
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

	var walk func(root *pbcodec.PermissionObject)
	walk = func(root *pbcodec.PermissionObject) {
		if root == nil {
			return
		}

		out = append(out, root)
		for _, child := range parentToChildren[root.Id] {
			walk(child)
		}
	}

	for _, root := range roots {
		walk(root)
	}

	return out
}
