package grpc

import (
	"context"
	"sort"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetMultiScopesTableRows(request *pbfluxdb.GetMultiScopesTableRowsRequest, stream pbfluxdb.FluxDB_GetMultiScopesTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get multi scope tables rows",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	// Sort by scope so at least, a constant order is kept across calls
	sort.Slice(request.Scopes, func(leftIndex, rightIndex int) bool {
		return request.Scopes[leftIndex] < request.Scopes[rightIndex]
	})

	scopes := make([]interface{}, len(request.Scopes))
	for i, s := range request.Scopes {
		scopes[i] = string(s)
	}

	nailer := dhammer.NewNailer(3, func(ctx context.Context, i interface{}) (interface{}, error) {
		scope := i.(string)

		responseRows, err := s.readContractStateTable(
			ctx,
			fluxdb.NewContractStateTablet(request.Contract, scope, request.Table),
			actualBlockNum,
			"",
			true,
			true,
			speculativeWrites,
		)
		if err != nil {
			return nil, err
		}

		resp := &pbfluxdb.TableRowsScopeResponse{
			Scope: scope,
			Row:   make([]*pbfluxdb.TableRowResponse, len(responseRows.Rows)),
		}

		for itr, row := range responseRows.Rows {
			resp.Row[itr] = processTableRow(&readTableRowResponse{
				ABI: responseRows.ABI,
				Row: row,
			})
		}
		return resp, nil
	})

	nailer.PushAll(ctx, scopes)

	stream.SetTrailer(getMetadata(upToBlockID, lastWrittenBlockID))

	for {
		select {
		case <-ctx.Done():
			return nil
		case next, ok := <-nailer.Out:
			if !ok {
				zlog.Debug("nailer completed")
				return nil
			}
			stream.Send(next.(*pbfluxdb.TableRowsScopeResponse))
		}
	}
	return nil
}
