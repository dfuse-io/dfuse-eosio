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

	tablet := fluxdb.NewContractStateTablet(request.Contract, request.Scope, request.Table)
	rows, serializationInfo, err := s.readContractStateTable(
		ctx,
		tablet,
		actualBlockNum,
		request.ToJson,
		speculativeWrites,
	)

	if err != nil {
		return derr.Statusf(codes.Internal, "read table rows failed: %s", err)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	stream.SetHeader(newMetadata(upToBlockID, lastWrittenBlockID))
	for _, row := range rows {
		response, err := toTableRowResponse(row.(*fluxdb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
		if err != nil {
			return derr.Statusf(codes.Internal, "creating table row response failed: %s", err)
		}

		stream.Send(response)
	}

	return nil
}
