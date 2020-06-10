package grpc

import (
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetTableRows(request *pbfluxdb.GetTableRowsRequest, stream pbfluxdb.State_GetTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table rows",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	responseRows, err := s.readContractStateTable(
		ctx,
		fluxdb.NewContractStateTablet(request.Contract, request.Scope, request.Table),
		actualBlockNum,
		request.KeyType,
		request.ToJson,
		request.WithBlockNum,
		speculativeWrites,
	)

	if err != nil {
		return derr.Statusf(codes.Internal, "read table rows failed: %s", err)
	}

	stream.SetTrailer(getMetadata(upToBlockID, lastWrittenBlockID))

	for _, row := range responseRows.Rows {
		stream.Send(processTableRow(&readTableRowResponse{
			Row: row,
		}))
	}
	return nil
}
