package grpc

import (
	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetTableRows(request *pbfluxdb.GetTableRowsRequest, stream pbfluxdb.FluxDB_GetTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table rows",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read", err)
	}

	responseRows, err := s.readTable(
		ctx,
		actualBlockNum,
		request.Account,
		request.Table,
		request.Scope,
		request.KeyType,
		request.WithAbi,
		request.ToJson,
		request.WithBlockNum,
		speculativeWrites,
	)

	if err != nil {
		return derr.Wrapf(err, "read rows failed: %w", err)
	}

	response := &getTableRowsResponse{
		commonStateResponse: newCommonGetResponse(upToBlockID, lastWrittenBlockID),
		readTableResponse:   responseRows,
	}

	zlog.Debug("streaming response", zap.Int("row_count", len(response.readTableResponse.Rows)), zap.Reflect("common_response", response.commonStateResponse))
	streamResponse(ctx, w, response)

	panic("implement me")
}
