package grpc

import (
	"context"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"

	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
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
		return nil, derr.Statusf(codes.Internal, "fetching ABI from db: %s", err)
	}

	linkedPermissions, err := s.db.ReadLinkedPermissions(ctx, actualBlockNum, eos.AccountName(request.Account), speculativeWrites)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "reading linked permissions failed: %s", err)
	}

	resp := &pbfluxdb.GetPermissionLinksResponse{
		UpToBlockId:              upToBlockID,
		UpToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		LastIrreversibleBlockId:  lastWrittenBlockID,
		LastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
		Permissions:              make([]*pbfluxdb.LinkedPermission, len(linkedPermissions)),
	}

	for i, permission := range linkedPermissions {
		resp.Permissions[i] = linkPermissionToProto(permission)
	}

	return resp, nil
}
