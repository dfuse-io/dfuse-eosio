package grpc

import (
	"context"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetTableRow(ctx context.Context, request *pbfluxdb.GetTableRowRequest) (*pbfluxdb.GetTableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table row",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	tablet := fluxdb.NewContractStateTablet(request.Contract, request.Scope, request.Table)
	row, serializationInfo, err := s.readContractStateTableRow(
		ctx,
		tablet,
		request.PrimaryKey,
		actualBlockNum,
		keyConverter,
		request.ToJson,
		speculativeWrites,
	)

	if err != nil {
		return nil, derr.Statusf(codes.Internal, "read tablet %q row failed: %s", tablet, err)
	}

	response, err := toTableRowResponse(row.(*fluxdb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "creating table row response failed: %s", err)
	}

	return &pbfluxdb.GetTableRowResponse{
		UpToBlockId:              upToBlockID,
		UpToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		LastIrreversibleBlockId:  lastWrittenBlockID,
		LastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
		Row:                      response,
	}, nil
}
