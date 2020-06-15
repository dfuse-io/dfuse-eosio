package grpc

import (
	"context"
	"fmt"
	"sort"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetMultiScopesTableRows(request *pbfluxdb.GetMultiScopesTableRowsRequest, stream pbfluxdb.State_GetMultiScopesTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get multi scope tables rows",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	var serializationInfo *rowSerializationInfo
	if request.ToJson {
		serializationInfo, err = s.newRowSerializationInfo(ctx, request.Contract, request.Table, actualBlockNum, speculativeWrites)
		if err != nil {
			return derr.Statusf(codes.Internal, "unable to obtain serialziation info: %s", err)
		}
	}

	if len(request.Scopes) == 1 && request.Scopes[0] == "*" {
		zlog.Debug("fetching all scopes since single scope received is '*'")
		scopes, err := s.fetchScopes(ctx, actualBlockNum, request.Contract, request.Table, speculativeWrites)
		if err != nil {
			return derr.Statusf(codes.Internal, "unable to fetch scopes: %s", err)
		}

		if len(scopes) == 0 {
			stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

			zlog.Debug("contract's table does not contain any scope, nothing to do")
			return nil
		}

		request.Scopes = scopes
	}

	// Sort by scope so at least, a constant order is kept across calls
	sort.Slice(request.Scopes, func(leftIndex, rightIndex int) bool {
		return request.Scopes[leftIndex] < request.Scopes[rightIndex]
	})

	scopes := make([]interface{}, len(request.Scopes))
	for i, s := range request.Scopes {
		scopes[i] = string(s)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	nailer := dhammer.NewNailer(64, func(ctx context.Context, i interface{}) (interface{}, error) {
		scope := i.(string)

		tablet := fluxdb.NewContractStateTablet(request.Contract, scope, request.Table)
		tabletRows, err := s.db.ReadTabletAt(
			ctx,
			blockNum,
			tablet,
			speculativeWrites,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to read tablet %s at %d: %w", tablet, blockNum, err)
		}

		resp := &pbfluxdb.TableRowsScopeResponse{
			Scope: scope,
			Row:   make([]*pbfluxdb.TableRowResponse, len(tabletRows)),
		}

		for i, tabletRow := range tabletRows {
			response, err := toTableRowResponse(tabletRow.(*fluxdb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
			if err != nil {
				return nil, fmt.Errorf("creating table row response failed: %w", err)
			}

			resp.Row[i] = response
		}

		return resp, nil
	})

	nailer.PushAll(ctx, scopes)

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

	for {
		select {
		case <-ctx.Done():
			zlog.Debug("stream terminated prior completion")
			return nil
		case next, ok := <-nailer.Out:
			if !ok {
				zlog.Debug("nailer completed")
				return nil
			}
			stream.Send(next.(*pbfluxdb.TableRowsScopeResponse))
		}
	}
}
