package grpc

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func (s *Server) GetCode(ctx context.Context, request *pbfluxdb.GetCodeRequest) (*pbfluxdb.GetCodeResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get contract",
		zap.String("contract", request.Contract),
		zap.Uint64("block_num", request.BlockNum),
	)

	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, uint32(request.BlockNum), false)
	if err != nil {
		return nil, fmt.Errorf("speculative writes: %w", err)
	}

	entry, err := s.db.ReadSingletEntryAt(ctx, fluxdb.NewContractCodeSinglet(request.Contract), actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	// FIXME: Is this the semantic we want for not found code call?
	if entry == nil {
		return &pbfluxdb.GetCodeResponse{}, nil
	}

	codeEntry := entry.(*fluxdb.ContractCodeEntry)
	return &pbfluxdb.GetCodeResponse{
		BlockNum: uint64(codeEntry.BlockNum()),
		RawCode:  codeEntry.Code(),
	}, nil
}
