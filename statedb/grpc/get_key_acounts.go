package grpc

import (
	"context"
	"sort"

	"github.com/streamingfast/derr"
	"github.com/streamingfast/fluxdb"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
)

func (s *Server) GetKeyAccounts(ctx context.Context, request *pbstatedb.GetKeyAccountsRequest) (*pbstatedb.GetKeyAccountsResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get key accounts",
		zap.String("public_key", request.PublicKey),
		zap.Uint64("block_num", request.BlockNum),
	)

	blockNum := request.BlockNum
	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tablet := statedb.NewKeyAccountTablet(request.PublicKey)
	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		actualBlockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to read tablet at %d: %s", actualBlockNum, err)
	}

	zlogger.Debug("post-processing key accounts", zap.Int("key_account_count", len(tabletRows)))
	accountNames := sortedUniqueKeyAccounts(tabletRows)
	if len(accountNames) == 0 {
		seen, err := s.db.HasSeenAnyRowForTablet(ctx, tablet)
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "unable to know if public key was seen once in db: %s", err)
		}

		if !seen {
			return nil, derr.Status(codes.NotFound, "This public key does not exist at this block height")
		}
	}

	return &pbstatedb.GetKeyAccountsResponse{
		BlockNum: uint64(actualBlockNum),
		Accounts: accountNames,
	}, nil
}

func sortedUniqueKeyAccounts(tabletRows []fluxdb.TabletRow) (out []string) {
	if len(tabletRows) <= 0 {
		return
	}

	accountNameSet := map[string]bool{}
	for _, tabletRow := range tabletRows {
		account, _ := tabletRow.(*statedb.KeyAccountRow).Explode()
		accountNameSet[account] = true
	}

	i := 0
	out = make([]string, len(accountNameSet))
	for account := range accountNameSet {
		out[i] = account
		i++
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return
}
