package grpc

import (
	"context"
	"fmt"
	"sort"

	"github.com/dfuse-io/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) StreamMultiContractsTableRows(request *pbstatedb.StreamMultiContractsTableRowsRequest, stream pbstatedb.State_StreamMultiContractsTableRowsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get multi accounts tables rows",
		zap.Reflect("request", request),
	)

	blockNum := uint64(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	// Sort by contract so at least, a constant order is kept across calls
	sort.Slice(request.Contracts, func(leftIndex, rightIndex int) bool {
		return request.Contracts[leftIndex] < request.Contracts[rightIndex]
	})

	contracts := make([]interface{}, len(request.Contracts))
	for i, s := range request.Contracts {
		contracts[i] = string(s)
	}

	keyConverter := getKeyConverterForType(request.KeyType)

	nailer := dhammer.NewNailer(64, func(ctx context.Context, i interface{}) (interface{}, error) {
		contract := i.(string)

		tablet := statedb.NewContractStateTablet(contract, request.Table, request.Scope)
		rows, serializationInfo, err := s.readContractStateTable(
			ctx,
			tablet,
			actualBlockNum,
			request.ToJson,
			speculativeWrites,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to read contract state tablet %q: %w", tablet, err)
		}

		resp := &pbstatedb.TableRowsContractResponse{
			Contract: contract,
			Rows:     make([]*pbstatedb.TableRowResponse, len(rows)),
		}

		for i, row := range rows {
			response, err := toTableRowResponse(row.(*statedb.ContractStateRow), keyConverter, serializationInfo, request.WithBlockNum)
			if err != nil {
				return nil, fmt.Errorf("creating table row response failed: %w", err)
			}

			resp.Rows[i] = response
		}

		return resp, nil
	}, zlog)

	nailer.PushAll(ctx, contracts)

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

			stream.Send(next.(*pbstatedb.TableRowsContractResponse))
		}
	}
}
