package grpc

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (s *Server) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = s.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.Stringer("head_block", headBlock))

	return
}

func (s *Server) prepareRead(
	ctx context.Context,
	blockNum uint32,
	irreversibleOnly bool,
) (chosenBlockNum uint32, lastWrittenBlockID string, upToBlockID string, speculativeWrites []*fluxdb.WriteRequest, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("performing prepare read operation")

	lastWrittenBlock, err := s.db.FetchLastWrittenBlock(ctx)
	if err != nil {
		err = derr.Wrap(err, "unable to retrieve last written block id")
		return
	}
	lastWrittenBlockNum := uint32(lastWrittenBlock.Num())

	if irreversibleOnly {
		if blockNum > lastWrittenBlockNum {
			err = fluxdb.AppBlockNumHigherThanLIBError(ctx, blockNum, lastWrittenBlockNum)
			return
		}
		if chosenBlockNum == 0 {
			chosenBlockNum = lastWrittenBlockNum
		}
		return
	}

	headBlock := s.fetchHeadBlock(ctx, zlog)
	headBlockNum := uint32(headBlock.Num())
	chosenBlockNum = blockNum
	if chosenBlockNum == 0 {
		chosenBlockNum = headBlockNum
	}

	if chosenBlockNum > headBlockNum {
		err = fluxdb.AppBlockNumHigherThanHeadBlockError(ctx, chosenBlockNum, headBlockNum, lastWrittenBlockNum)
		return
	}

	// If we're between lastWrittenBlockNum and headBlockNum, we need to apply whatever's between
	zlog.Debug("fetching speculative writes", zap.String("head_block_id", headBlock.ID()), zap.Uint32("chosen_block_num", chosenBlockNum))
	speculativeWrites = s.db.SpeculativeWritesFetcher(ctx, headBlock.ID(), chosenBlockNum)

	if len(speculativeWrites) >= 1 {
		upToBlockID = hex.EncodeToString(speculativeWrites[len(speculativeWrites)-1].BlockID)
		zlog.Debug("speculative writes present",
			zap.Int("speculative_write_count", len(speculativeWrites)),
			zap.String("up_to_block_id", upToBlockID),
		)
	}

	return
}

func (s *Server) readContractStateTable(
	ctx context.Context,
	tablet fluxdb.ContractStateTablet,
	blockNum uint32,
	toJSON bool,
	speculativeWrites []*fluxdb.WriteRequest,
) ([]fluxdb.TabletRow, *rowSerializationInfo, error) {
	ctx, span := dtracing.StartSpan(ctx, "read contract state table")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("read contract state tablet", zap.Stringer("tablet", tablet))

	tabletRows, err := s.db.ReadTabletAt(
		ctx,
		blockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("read tablet at: %w", err)
	}

	zlog.Debug("read tablet rows results", zap.Int("row_count", len(tabletRows)))

	var serializationInfo *rowSerializationInfo
	if toJSON {
		_, contract, _, table := tablet.Explode()
		serializationInfo, err = s.newRowSerializationInfo(ctx, contract, table, blockNum, speculativeWrites)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to obtain serialziation info: %w", err)
		}
	}

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(tabletRows))),
	}, "read contract state tablet")

	return tabletRows, serializationInfo, nil
}

func (s *Server) readContractStateTableRow(
	ctx context.Context,
	tablet fluxdb.ContractStateTablet,
	primaryKey string,
	blockNum uint32,
	keyConverter KeyConverter,
	toJSON bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (fluxdb.TabletRow, *rowSerializationInfo, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug(
		"reading contract state table row",
		zap.String("table_key", tablet.Key()),
		zap.String("primary_key", primaryKey),
		zap.Uint32("block_nume", blockNum),
	)

	primaryKeyValue, err := keyConverter.FromString(primaryKey)
	tabletRow, err := s.db.ReadTabletRowAt(
		ctx,
		blockNum,
		tablet,
		fluxdb.UN(primaryKeyValue),
		speculativeWrites,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read tablet row at: %w", err)
	}

	_, contract, scope, table := tablet.Explode()
	zlog.Debug("read tablet row result",
		zap.String("contract", contract),
		zap.String("table", table),
		zap.String("scope", scope),
		zap.String("primary_key", primaryKey),
	)

	if tabletRow == nil {
		zlogger.Debug("row deleted or never existed")
		return nil, nil, fluxdb.DataRowNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table), eos.AccountName(scope), primaryKey)
	}

	var serializationInfo *rowSerializationInfo
	if toJSON {
		_, contract, _, table := tablet.Explode()
		serializationInfo, err = s.newRowSerializationInfo(ctx, contract, table, blockNum, speculativeWrites)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to obtain serialization info: %w", err)
		}
	}

	return tabletRow, serializationInfo, nil
}

type rowSerializationInfo struct {
	abi           *eos.ABI
	abiAtBlockNum uint32
	tableTypeName string
}

func (s *rowSerializationInfo) Decode(data []byte) ([]byte, error) {
	return s.abi.DecodeTableRowTyped(s.tableTypeName, data)
}

func (s *Server) newRowSerializationInfo(ctx context.Context, contract, table string, blockNum uint32, speculativeWrites []*fluxdb.WriteRequest) (*rowSerializationInfo, error) {
	abiEntry, err := s.db.ReadSigletEntryAt(ctx, fluxdb.NewContractABISiglet(contract), blockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("read abi at %d: %w", blockNum, err)
	}

	if abiEntry == nil {
		return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
	}

	abi, err := abiEntry.(*fluxdb.ContractABIEntry).ABI()
	if err != nil {
		return nil, fmt.Errorf("decode abi: %w", err)
	}

	if abi == nil {
		return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
	}

	tableDef := abi.TableForName(eos.TableName(table))
	if tableDef == nil {
		return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table))
	}

	return &rowSerializationInfo{
		abi:           abi,
		abiAtBlockNum: abiEntry.BlockNum(),
		tableTypeName: tableDef.Type,
	}, nil
}

func toTableRowResponse(row *fluxdb.ContractStateRow, keyConverter KeyConverter, serializationInfo *rowSerializationInfo, withBlockNum bool) (*pbfluxdb.TableRowResponse, error) {
	// FIXME: Improve that, if keyConverter is already converting to "name" type, we can simply return actual row.PrimaryKey() as-is unmodified!
	primaryKey, err := keyConverter.ToString(fluxdb.N(row.PrimaryKey()))
	if err != nil {
		return nil, fmt.Errorf("unable to convert key %s: %w", row.PrimaryKey(), err)
	}

	response := &pbfluxdb.TableRowResponse{
		Key:   primaryKey,
		Payer: row.Payer(),
	}

	if withBlockNum {
		response.BlockNumber = uint64(row.BlockNum())
	}

	response.Data = row.Data()
	if serializationInfo != nil {
		jsonData, err := serializationInfo.Decode(response.Data)
		if err != nil {
			zlog.Warn("failed to decode row from ABI",
				zap.Uint32("block_num", serializationInfo.abiAtBlockNum),
				zap.String("struct_type", serializationInfo.tableTypeName),
				zap.Error(err),
			)
		} else {
			response.Data = nil
			response.Json = string(jsonData)
		}
	}

	return response, nil
}
