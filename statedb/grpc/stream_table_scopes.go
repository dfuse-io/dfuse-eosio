package grpc

import (
	"context"
	"sort"

	"github.com/streamingfast/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/logging"
	"github.com/streamingfast/fluxdb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) StreamTableScopes(request *pbstatedb.StreamTableScopesRequest, stream pbstatedb.State_StreamTableScopesServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table scopes",
		zap.Reflect("request", request),
	)

	blockNum := uint64(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	scopes, err := s.fetchScopes(ctx, actualBlockNum, request.Contract, request.Table, speculativeWrites)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to fetch scopes: %s", err)
	}

	if len(scopes) == 0 {
		zlogger.Debug("no scopes found for request, checking if we ever see this table")
		seen, err := s.db.HasSeenAnyRowForTablet(ctx, statedb.NewContractTableScopeTablet(request.Contract, request.Table))
		if err != nil {
			return derr.Statusf(codes.Internal, "unable to know if table was seen once in db: %s", err)
		}

		if !seen {
			return derr.Statusf(codes.NotFound, "table %s/%s does not exist in ABI at block height %d", request.Contract, request.Table, actualBlockNum)
		}
	}

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

	for _, scope := range scopes {
		stream.Send(&pbstatedb.TableScopeResponse{
			BlockNum: uint64(actualBlockNum),
			Scope:    string(scope),
		})
	}

	return nil
}

func (s *Server) fetchScopes(ctx context.Context, blockNum uint64, contract, table string, speculativeWrites []*fluxdb.WriteRequest) (scopes []string, err error) {
	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		statedb.NewContractTableScopeTablet(contract, table),
		speculativeWrites,
	)
	if err != nil {
		return nil, err
	}

	logging.Logger(ctx, zlog).Debug("post-processing table scopes", zap.Int("table_scope_count", len(tabletRows)))
	return sortedScopes(tabletRows), nil
}

func sortedScopes(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	out = make([]string, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i] = tabletRow.(*statedb.ContractTableScopeRow).Scope()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
