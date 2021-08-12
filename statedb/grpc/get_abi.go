package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/streamingfast/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/streamingfast/logging"
	"github.com/dfuse-io/validator"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) GetABI(ctx context.Context, request *pbstatedb.GetABIRequest) (*pbstatedb.GetABIResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get abi",
		zap.String("contract", request.Contract),
		zap.Uint64("block_num", request.BlockNum),
	)

	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, request.BlockNum, false)
	if err != nil {
		return nil, fmt.Errorf("speculative writes: %w", err)
	}

	entry, err := s.db.ReadSingletEntryAt(ctx, statedb.NewContractABISinglet(request.Contract), actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	// FIXME: Is this the semantic we want for not found ABI call?
	if entry == nil {
		return &pbstatedb.GetABIResponse{}, nil
	}

	abiEntry := entry.(*statedb.ContractABIEntry)
	if request.ToJson {
		abi, _, err := abiEntry.ABI()
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to decode ABI to JSON: %s", err)
		}

		cnt, err := json.Marshal(abi)
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to marshal ABI to JSON: %s", err)
		}

		return &pbstatedb.GetABIResponse{
			BlockNum: abiEntry.Height(),
			JsonAbi:  string(cnt),
		}, nil
	}

	_, rawABI, err := abiEntry.ABI(statedb.ContractABIPackedOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to unmarshal contract abi value: %s", err)
	}

	return &pbstatedb.GetABIResponse{
		BlockNum: abiEntry.Height(),
		RawAbi:   rawABI,
	}, nil

}

func validateGetABIRequest(request *pbstatedb.GetABIRequest) error {
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
