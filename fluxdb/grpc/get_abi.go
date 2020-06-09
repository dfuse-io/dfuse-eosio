package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/validator"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetABI(ctx context.Context, request *pbfluxdb.GetABIRequest) (*pbfluxdb.GetABIResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get abi",
		zap.String("contract", request.Contract),
		zap.Uint64("block_num", request.BlockNum),
	)

	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, uint32(request.BlockNum), false)
	if err != nil {
		return nil, fmt.Errorf("speculative writes: %w", err)
	}

	entry, err := s.db.ReadSigletEntryAt(ctx, fluxdb.NewContractABISiglet(request.Contract), actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	// FIXME: Is this the semantic we want for not found ABI call?
	if entry == nil {
		return &pbfluxdb.GetABIResponse{}, nil
	}

	abiEntry := entry.(*fluxdb.ContractABIEntry)
	if request.ToJson {
		abi, err := abiEntry.ABI()
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to decode ABI to JSON: %s", err)
		}

		cnt, err := json.Marshal(abi)
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to marshal ABI to JSON: %s", err)
		}

		return &pbfluxdb.GetABIResponse{
			BlockNum: uint64(abiEntry.BlockNum()),
			JsonAbi:  string(cnt),
		}, nil
	}

	return &pbfluxdb.GetABIResponse{
		BlockNum: uint64(abiEntry.BlockNum()),
		RawAbi:   abiEntry.PackedABI(),
	}, nil

}

func validateGetABIRequest(request *pbfluxdb.GetABIRequest) error {
	errors := validator.ValidateStruct(request, validator.Rules{
		"contract":  []string{"required", "fluxdb.eos.name"},
		"block_num": []string{"fluxdb.eos.blockNum"},
		"to_json":   []string{"bool"},
	})
	if len(errors) == 0 {
		return nil
	}

	msgs := []string{}
	for _, errs := range errors {
		msgs = append(msgs, errs...)
	}

	return fmt.Errorf("%s", strings.Join(msgs, ", "))
}
