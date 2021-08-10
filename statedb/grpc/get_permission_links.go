package grpc

import (
	"context"
	"sort"

	"github.com/streamingfast/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/logging"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetPermissionLinks(ctx context.Context, request *pbstatedb.GetPermissionLinksRequest) (*pbstatedb.GetPermissionLinksResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get permission links",
		zap.Uint64("bock_num", request.BlockNum),
		zap.String("account", request.Account),
	)

	blockNum := uint64(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		actualBlockNum,
		statedb.NewAuthLinkTablet(request.Account),
		speculativeWrites,
	)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to read tablet at %d: %s", blockNum, err)
	}

	resp := &pbstatedb.GetPermissionLinksResponse{
		UpToBlock:             &pbbstream.BlockRef{Num: upToBlock.Num(), Id: upToBlock.ID()},
		LastIrreversibleBlock: &pbbstream.BlockRef{Num: lastWrittenBlock.Num(), Id: lastWrittenBlock.ID()},
		Permissions:           make([]*pbstatedb.LinkedPermission, len(tabletRows)),
	}

	for i, tabletRow := range tabletRows {
		row := tabletRow.(*statedb.AuthLinkRow)
		contract, action := row.Explode()
		permission, err := row.Permission()
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "unable to read tablet row %q value at %d: %s", row, blockNum, err)
		}

		resp.Permissions[i] = &pbstatedb.LinkedPermission{
			Contract:       contract,
			Action:         action,
			PermissionName: string(permission),
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
