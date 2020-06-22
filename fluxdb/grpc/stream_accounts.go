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

func (s *Server) StreamAccounts(request *pbfluxdb.StreamAccountsRequest, stream pbfluxdb.State_StreamAccountsServer) error {
	ctx := stream.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("stream accounts",
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
		fluxdb.NewAccountsTablet(),
		speculativeWrites,
	)
	if err != nil {
		return err
	}

	logging.Logger(ctx, zlog).Debug("post-processing accounts", zap.Int("account_count", len(tabletRows)))
	accounts := sortedAccounts(tabletRows)

	stream.SetHeader(newMetadata(upToBlock, lastWrittenBlock))

	for _, account := range accounts {
		stream.Send(&pbfluxdb.AccountResponse{
			BlockNum: uint64(actualBlockNum),
			Account:  account,
		})
	}

	return nil
}

func sortedAccounts(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	out = make([]string, len(tabletRows))
	for i, tabletRow := range tabletRows {
		out[i] = tabletRow.(*fluxdb.AccountsRow).Account()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
