package grpc

import (
	"context"
	"sort"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"

	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetPermissionLinks(ctx context.Context, request *pbfluxdb.GetPermissionLinksRequest) (*pbfluxdb.GetPermissionLinksResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get permission links",
		zap.Uint64("bock_num", request.BlockNum),
		zap.String("account", request.Account),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		actualBlockNum,
		fluxdb.NewAuthLinkTablet(request.Account),
		speculativeWrites,
	)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "uanble to read tablet at %d: %s", blockNum, err)
	}

	resp := &pbfluxdb.GetPermissionLinksResponse{
		UpToBlockId:              upToBlockID,
		UpToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		LastIrreversibleBlockId:  lastWrittenBlockID,
		LastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
		Permissions:              make([]*pbfluxdb.LinkedPermission, len(tabletRows)),
	}

	for i, tabletRow := range tabletRows {
		row := tabletRow.(*fluxdb.AuthLinkRow)
		contract, action := row.Explode()

		resp.Permissions[i] = &pbfluxdb.LinkedPermission{
			Contract:       contract,
			Action:         action,
			PermissionName: string(row.Permission()),
		}
	}

	zlogger.Debug("sorting linked permissions")
	permissions := resp.Permissions
	sort.Slice(permissions, func(i, j int) bool {
		if permissions[i].Contract == permissions[j].Contract {
			return permissions[i].Action < permissions[j].Action
		}

		return permissions[i].Contract < permissions[j].Contract
	})
	resp.Permissions = permissions

	return resp, nil
}
