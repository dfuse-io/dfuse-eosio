package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dfuse-io/validator"

	"go.uber.org/zap"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"google.golang.org/grpc/codes"

	"github.com/dfuse-io/derr"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
)

func (s *Server) GetABI(ctx context.Context, request *pbfluxdb.GetABIRequest) (*pbfluxdb.GetABIResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get abi",
		zap.String("contract", request.Contract),
		zap.Uint64("block_num", request.BlockNum),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, _, _, speculativeWrites, err := s.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read", err)
	}

	abiRow, err := s.db.GetABI(ctx, uint32(actualBlockNum), fluxdb.N(request.Contract), speculativeWrites)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "fetching ABI from db: %w", err)
	}

	if request.ToJson {
		var abiObj *eos.ABI
		if err = eos.UnmarshalBinary(abiRow.PackedABI, &abiObj); err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to decode packed ABI %q to JSON: %w", abiRow.PackedABI, err)
		}

		cnt, err := json.Marshal(abiObj)
		if err != nil {
			return nil, derr.Statusf(codes.Internal, "failed to marshal ABI: %w", err)
		}

		return &pbfluxdb.GetABIResponse{
			BlockNum: uint64(abiRow.BlockNum),
			JsonAbi:  string(cnt),
		}, nil
	}

	return &pbfluxdb.GetABIResponse{
		BlockNum: uint64(abiRow.BlockNum),
		RawAbi:   abiRow.PackedABI,
	}, nil

}

func validateGetABIRequest(request *pbfluxdb.GetABIRequest) error {
	errors := validator.ValidateStruct(request, validator.Rules{
		"contract":  []string{"required", "fluxdb.eos.name"},
		"block_num": []string{"fluxdb.eos.blockNum"},
		"to_json":   []string{"bool"},
	})
	fmt.Println(errors)
	if len(errors) == 0 {
		return nil
	}
	msgs := []string{}
	for _, errs := range errors {
		msgs = append(msgs, errs...)
	}
	return fmt.Errorf("%s", strings.Join(msgs, ", "))
}
