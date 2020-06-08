package grpc

import (
	"context"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"

	"google.golang.org/grpc/codes"

	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func (s *Server) GetTableRow(ctx context.Context, request *pbfluxdb.GetTableRowRequest) (*pbfluxdb.TableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table row",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read", err)
	}

	tableRowResponse, err := s.readTableRow(
		ctx,
		actualBlockNum,
		request.Account,
		request.Table,
		request.Scope,
		request.PrimaryKey,
		request.KeyType,
		request.WithAbi,
		request.ToJson,
		request.WithBlockNum,
		speculativeWrites,
	)

	if err != nil {
		return nil, derr.Statusf(codes.Internal, "read table row failed: %w", err)
	}

	return &pbfluxdb.TableRowResponse{
		Key:                      "",
		Data:                     tableRowResponse.Row.Data,
		Payer:                    tableRowResponse.Row.Payer,
		BlockNumber:              uint64(actualBlockNum),
		UpToBlockId:              upToBlockID,
		UpToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		LastIrreversibleBlockId:  lastWrittenBlockID,
		LastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
	}, nil
}
