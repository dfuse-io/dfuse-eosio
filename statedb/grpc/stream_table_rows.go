package grpc

import (
	"github.com/dfuse-io/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetTableRows(request *pbstatedb.StreamTableRowsRequest, stream pbstatedb.State_StreamTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table rows",
		zap.Reflect("request", request),
	)

	blockNum := uint64(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tablet := statedb.NewContractStateTablet(request.Contract, request.Table, request.Scope)
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

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))
	for _, row := range rows {
		response, err := toTableRowResponse(row.(*statedb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
		if err != nil {
			return derr.Statusf(codes.Internal, "creating table row response failed: %s", err)
		}

		stream.Send(response)
	}

	return nil
}
