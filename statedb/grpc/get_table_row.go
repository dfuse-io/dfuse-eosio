package grpc

import (
	"context"

	"github.com/streamingfast/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/streamingfast/logging"
	pbbstream "github.com/streamingfast/pbgo/dfuse/bstream/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetTableRow(ctx context.Context, request *pbstatedb.GetTableRowRequest) (*pbstatedb.GetTableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table row",
		zap.Reflect("request", request),
	)

	blockNum := uint64(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	tablet := statedb.NewContractStateTablet(request.Contract, request.Table, request.Scope)
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
		// If not `Unknown` code, return it as-is, it's already a status
		if status.Code(err) != codes.Unknown {
			return nil, err
		}

		return nil, derr.Statusf(codes.Internal, "read tablet %q row failed: %s", tablet, err)
	}

	response, err := toTableRowResponse(row.(*statedb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "creating table row response failed: %s", err)
	}

	return &pbstatedb.GetTableRowResponse{
		UpToBlock:             &pbbstream.BlockRef{Num: upToBlock.Num(), Id: upToBlock.ID()},
		LastIrreversibleBlock: &pbbstream.BlockRef{Num: lastWrittenBlock.Num(), Id: lastWrittenBlock.ID()},
		Row:                   response,
	}, nil
}
