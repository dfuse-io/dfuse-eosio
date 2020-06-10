package grpc

import (
	"context"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

func (s *Server) GetTableRow(ctx context.Context, request *pbfluxdb.GetTableRowRequest) (*pbfluxdb.GetTableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table row",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tableRowResponse, err := s.readContractStateTableRow(
		ctx,
		fluxdb.
			NewContractStateTablet(request.Contract, request.Scope, request.Table),
		actualBlockNum,
		request.KeyType,
		request.PrimaryKey,
		request.ToJson,
		request.WithBlockNum,
		speculativeWrites,
	)

	if err != nil {
		return nil, derr.Statusf(codes.Internal, "read table row failed: %s", err)
	}

	return &pbfluxdb.GetTableRowResponse{
		UpToBlockId:              upToBlockID,
		UpToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		LastIrreversibleBlockId:  lastWrittenBlockID,
		LastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
		Row:                      processTableRow(tableRowResponse),
	}, nil
}

func processTableRow(tableRow *readTableRowResponse) *pbfluxdb.TableRowResponse {
	payload := &pbfluxdb.TableRowResponse{
		Key:         tableRow.Row.Key,
		Payer:       tableRow.Row.Payer,
		BlockNumber: uint64(tableRow.Row.BlockNum),
	}
	switch v := tableRow.Row.Data.(type) {
	case []byte:
		payload.Data = v
	case *onTheFlyABISerializer:
		s := v
		jsonData, err := s.abi.DecodeTableRowTyped(s.tableTypeName, s.rowDataToDecode)
		if err != nil {
			tableRow.Row.Data = s.rowDataToDecode
			zlog.Warn("failed to decode row from ABI",
				zap.Uint32("block_num", s.abiAtBlockNum),
				zap.String("struct_type", s.tableTypeName),
				zap.Error(err),
			)
		} else {
			payload.Json = string(jsonData)
		}
	}
	return payload
}
