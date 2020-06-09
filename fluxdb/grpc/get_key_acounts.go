package grpc

import (
	"context"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

func (s *Server) GetKeyAccounts(ctx context.Context, request *pbfluxdb.GetKeyAccountsRequest) (*pbfluxdb.GetKeyAccountsResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get key accounts",
		zap.String("public_key", request.PublicKey),
		zap.Uint64("block_num", request.BlockNum),
	)

	blockNum := uint32(request.BlockNum)

	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	accountNames, err := s.db.ReadKeyAccounts(ctx, uint32(actualBlockNum), request.PublicKey, speculativeWrites)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to read key accounts from db: %s", err)
	}

	if len(accountNames) == 0 {
		seen, err := s.db.HasSeenPublicKeyOnce(ctx, request.PublicKey)
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "unable to know if public key was seen once in db: %s", err)
		}

		if !seen {
			return nil, derr.Status(codes.Internal, "This public key does not exist at this block height")
		}
	}

	resp := &pbfluxdb.GetKeyAccountsResponse{
		BlockNum: uint64(actualBlockNum),
		Accounts: make([]string, len(accountNames)),
	}
	for itr, acc := range accountNames {
		resp.Accounts[itr] = string(acc)
	}

	return resp, nil

}
