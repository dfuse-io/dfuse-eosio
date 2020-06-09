package grpc

import (
	"context"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

func (s *Server) GetTableRow(ctx context.Context, request *pbfluxdb.GetTableRowRequest) (*pbfluxdb.TableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("get table row",
		zap.Reflect("request", request),
	)

	blockNum := uint32(request.BlockNum)
	actualBlockNum, lastWrittenBlockID, upToBlockID, speculativeWrites, err := s.prepareRead(ctx, blockNum, request.IrreversibleOnly)
	if err != nil {
		return nil, derr.Statusf(codes.Internal, "unable to prepare read: %s", err)
	}

	tableRowResponse, err := s.readTableRow(
		ctx,
		actualBlockNum,
		request.Contract,
		request.Table,
		request.Scope,
		request.PrimaryKey,
		request.KeyType,
		request.WithAbi,
		request.ToJson,
		request.WithBlockNum,
		speculativeWrites,
	)

	if err != nil {
		return nil, derr.Statusf(codes.Internal, "read table row failed: %s", err)
	}

	return processTableRow(tableRowResponse, newReadReference(upToBlockID, lastWrittenBlockID)), nil
}

func processTableRow(tableRow *readTableRowResponse, ref *readReference) *pbfluxdb.TableRowResponse {
	payload := &pbfluxdb.TableRowResponse{
		Key:                      tableRow.Row.Key,
		Payer:                    tableRow.Row.Payer,
		BlockNumber:              uint64(tableRow.Row.BlockNum),
		UpToBlockId:              ref.upToBlockId,
		UpToBlockNum:             ref.upToBlockNum,
		LastIrreversibleBlockId:  ref.lastIrreversibleBlockId,
		LastIrreversibleBlockNum: ref.lastIrreversibleBlockNum,
	}
	switch v := tableRow.Row.Data.(type) {
	case []byte:
		payload.Data = v
	case *onTheFlyABISerializer:
		s := v
		jsonData, err := s.abi.DecodeTableRowTyped(s.structType, s.data)
		if err != nil {
			tableRow.Row.Data = s.data
			zlog.Warn("failed to decode row from ABI",
				zap.Uint32("block_num", s.abiRow.BlockNum),
				zap.String("struct_type", s.structType),
				zap.Error(err),
			)
		} else {
			payload.Json = string(jsonData)
		}
	}
	return payload
}
