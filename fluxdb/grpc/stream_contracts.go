package grpc

import (
	"sort"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) StreamContracts(request *pbfluxdb.StreamContractsRequest, stream pbfluxdb.State_StreamContractsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("stream contracts",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlock, upToBlock, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		fluxdb.NewContractTablet(),
		speculativeWrites,
	)
	if err != nil {
		return err
	}

	logging.Logger(ctx, zlog).Debug("post-processing contracts", zap.Int("contract_count", len(tabletRows)))
	contracts := sortedContracts(tabletRows)

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

	for _, contract := range contracts {
		stream.Send(&pbfluxdb.ContractResponse{
			BlockNum: uint64(actualBlockNum),
			Contract: contract,
		})
	}

	return nil
}

func sortedContracts(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	out = make([]string, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i] = tabletRow.(*fluxdb.ContractRow).Contract()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
