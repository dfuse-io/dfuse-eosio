package grpc

import (
	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetTableScopes(request *pbfluxdb.GetTableScopesRequest, stream pbfluxdb.FluxDB_GetTableScopesServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table scopes",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	contract := eos.AccountName(request.Contract)
	table := eos.TableName(request.Table)
	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	scopes, err := s.db.ReadTableScopes(ctx, actualBlockNum, contract, table, speculativeWrites)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to read table scopes from db: %s", err)
	}

	if len(scopes) == 0 {
		logging.Logger(ctx, zlog).Debug("no scopes found for request, checking if we ever see this table")
		seen, err := s.db.HasSeenTableOnce(ctx, contract, table)
		if err != nil {
			return derr.Status(codes.Internal, "unable to know if table was seen once in db")
		}

		if !seen {
			return derr.Status(codes.Internal, "table does not exist in ABI at this block height")
		}
	}

	// TODO: pass block num in header
	for _, scope := range scopes {
		stream.Send(&pbfluxdb.TableScopeResponse{
			BlockNum: uint64(actualBlockNum),
			Scope:    string(scope),
		})
	}

	return nil
}
