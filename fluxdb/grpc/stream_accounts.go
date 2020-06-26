package grpc

import (
	"context"
	"fmt"
	"sort"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) StreamAccounts(request *pbfluxdb.StreamAccountsRequest, stream pbfluxdb.State_StreamAccountsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("stream accounts",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		fluxdb.NewAccountsTablet(),
		speculativeWrites,
	)
	if err != nil {
		return err
	}

	logging.Logger(ctx, zlog).Debug("post-processing accounts", zap.Int("account_count", len(tabletRows)))
	accounts := sortedAccounts(tabletRows)

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

	for _, account := range accounts {
		permissions, err := s.readPermissions(ctx, blockNum, account, speculativeWrites)
		if err != nil {
			return fmt.Errorf("read permissions: %w", err)
		}

		linkedPermissions, err := s.readLinkedPermissions(ctx, blockNum, account, speculativeWrites)
		if err != nil {
			return fmt.Errorf("read linked permissions: %w", err)
		}

		stream.Send(&pbfluxdb.AccountResponse{
			BlockNum:          uint64(actualBlockNum),
			Account:           account,
			Permissions:       permissions,
			LinkedPermissions: linkedPermissions,
		})
	}

	return nil
}

func (s *Server) readPermissions(ctx context.Context, blockNum uint32, account string, speculativeWrites []*fluxdb.WriteRequest) ([]*pbcodec.PermissionObject, error) {
	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		fluxdb.NewAccountPermissionsTablet(account),
		speculativeWrites,
	)
	if err != nil {
		return nil, err
	}

	out := make([]*pbcodec.PermissionObject, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i], err = tabletRow.(*fluxdb.AccountPermissionsRow).PermissionObject()
		if err != nil {
			return nil, fmt.Errorf("row permission object: %w", err)
		}
	}

	return out, nil
}

func (s *Server) readLinkedPermissions(ctx context.Context, blockNum uint32, account string, speculativeWrites []*fluxdb.WriteRequest) ([]*pbfluxdb.LinkedPermission, error) {
	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		fluxdb.NewAuthLinkTablet(account),
		speculativeWrites,
	)
	if err != nil {
		return nil, err
	}

	out := make([]*pbfluxdb.LinkedPermission, len(tabletRows))
	for i, tabletRow := range tabletRows {
		row := tabletRow.(*fluxdb.AuthLinkRow)
		contract, action := row.Explode()

		out[i] = &pbfluxdb.LinkedPermission{
			PermissionName: string(row.Permission()),
			Contract:       contract,
			Action:         action,
		}
	}

	return out, nil
}

func sortedAccounts(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	out = make([]string, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i] = tabletRow.(*fluxdb.AccountsRow).Account()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
