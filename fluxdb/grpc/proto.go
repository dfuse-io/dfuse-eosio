package grpc

import (
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

func linkPermissionToProto(permission *fluxdb.LinkedPermission) *pbfluxdb.LinkedPermission {
	return &pbfluxdb.LinkedPermission{
		Contract:       permission.Contract,
		Action:         permission.Action,
		PermissionName: permission.PermissionName,
	}
}
