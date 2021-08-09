package grpc

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/fluxdb"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = s.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.Stringer("head_block", headBlock))

	return
}

func (s *Server) prepareRead(
	ctx context.Context,
	blockNum uint64,
	irreversibleOnly bool,
) (chosenBlockNum uint64, lastWrittenBlock bstream.BlockRef, upToBlock bstream.BlockRef, speculativeWrites []*fluxdb.WriteRequest, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("performing prepare read operation")

	_, lastWrittenBlock, err = s.db.FetchLastWrittenCheckpoint(ctx)
	if err != nil {
		err = derr.Wrap(err, "unable to retrieve last written block id")
		return
	}

	lastWrittenBlockNum := lastWrittenBlock.Num()
	if irreversibleOnly {
		if blockNum > lastWrittenBlockNum {
			err = statedb.AppBlockNumHigherThanLIBError(ctx, blockNum, lastWrittenBlockNum)
			return
		}
		if chosenBlockNum == 0 {
			chosenBlockNum = lastWrittenBlockNum
		}
		return
	}

	headBlock := s.fetchHeadBlock(ctx, zlog)
	if bstream.EqualsBlockRefs(headBlock, bstream.BlockRefEmpty) {
		err = statedb.AppNotReadyError(ctx)
		return
	}

	headBlockNum := headBlock.Num()
	chosenBlockNum = blockNum
	if chosenBlockNum == 0 {
		chosenBlockNum = headBlockNum
	}

	if chosenBlockNum > headBlockNum {
		err = statedb.AppBlockNumHigherThanHeadBlockError(ctx, chosenBlockNum, headBlockNum, lastWrittenBlockNum)
		return
	}

	// If we're between lastWrittenBlockNum and headBlockNum, we need to apply whatever's between
	zlog.Debug("fetching speculative writes", zap.Stringer("head_block", headBlock), zap.Uint64("chosen_block_num", chosenBlockNum))
	speculativeWrites = s.db.SpeculativeWritesFetcher(ctx, headBlock.ID(), chosenBlockNum)

	if len(speculativeWrites) >= 1 {
		lastSpeculativeWrite := speculativeWrites[len(speculativeWrites)-1]
		upToBlock = lastSpeculativeWrite.BlockRef
		zlog.Debug("speculative writes present",
			zap.Int("speculative_write_count", len(speculativeWrites)),
			zap.Stringer("up_to_block", upToBlock),
		)
	} else {
		zlog.Info("no speculative writes available, up to block cannot be determined, using empty value")
		upToBlock = bstream.BlockRefEmpty
	}

	return
}

func (s *Server) readContractStateTable(
	ctx context.Context,
	tablet statedb.ContractStateTablet,
	blockNum uint64,
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
		contract, table, _ := tablet.Explode()
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
	tablet statedb.ContractStateTablet,
	primaryKey string,
	blockNum uint64,
	keyConverter KeyConverter,
	toJSON bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (fluxdb.TabletRow, *rowSerializationInfo, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug(
		"reading contract state table row",
		zap.Stringer("tablet", tablet),
		zap.String("primary_key", primaryKey),
		zap.Uint64("block_num", blockNum),
	)

	primaryKeyBytes, err := toContractStatePrimaryKey(primaryKey, keyConverter)
	if err != nil {
		return nil, nil, fmt.Errorf("primary key conversion: %w", err)
	}

	tabletRow, err := s.db.ReadTabletRowAt(
		ctx,
		blockNum,
		tablet,
		primaryKeyBytes,
		speculativeWrites,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read tablet row at: %w", err)
	}

	contract, table, scope := tablet.Explode()
	zlog.Debug("read tablet row result",
		zap.String("contract", contract),
		zap.String("table", table),
		zap.String("scope", scope),
		zap.String("primary_key", primaryKey),
	)

	if tabletRow == nil {
		zlogger.Debug("row deleted or never existed")
		return nil, nil, derr.Status(codes.NotFound, fmt.Sprintf(`table row primary %q on "%s:%s:%s" deleted or never existed`, primaryKey, contract, table, scope))
	}

	var serializationInfo *rowSerializationInfo
	if toJSON {
		contract, table, _ := tablet.Explode()
		serializationInfo, err = s.newRowSerializationInfo(ctx, contract, table, blockNum, speculativeWrites)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to obtain serialization info: %w", err)
		}
	}

	return tabletRow, serializationInfo, nil
}

type rowSerializationInfo struct {
	abi           *eos.ABI
	abiAtBlockNum uint64
	tableTypeName string
}

func (s *rowSerializationInfo) Decode(data []byte) ([]byte, error) {
	return s.abi.DecodeTableRowTyped(s.tableTypeName, data)
}

func (s *Server) newRowSerializationInfo(ctx context.Context, contract, table string, blockNum uint64, speculativeWrites []*fluxdb.WriteRequest) (*rowSerializationInfo, error) {
	abiEntry, err := s.db.ReadSingletEntryAt(ctx, statedb.NewContractABISinglet(contract), blockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("read abi at %d: %w", blockNum, err)
	}

	if abiEntry == nil {
		return nil, statedb.DataABINotFoundError(ctx, contract, blockNum)
	}

	abi, _, err := abiEntry.(*statedb.ContractABIEntry).ABI()
	if err != nil {
		return nil, fmt.Errorf("decode abi: %w", err)
	}

	if abi == nil {
		return nil, statedb.DataABINotFoundError(ctx, contract, blockNum)
	}

	tableDef := abi.TableForName(eos.TableName(table))
	if tableDef == nil {
		return nil, statedb.DataTableNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table))
	}

	return &rowSerializationInfo{
		abi:           abi,
		abiAtBlockNum: abiEntry.Height(),
		tableTypeName: tableDef.Type,
	}, nil
}

func toTableRowResponse(row *statedb.ContractStateRow, keyConverter KeyConverter, serializationInfo *rowSerializationInfo, withBlockNum bool) (*pbstatedb.TableRowResponse, error) {
	primaryKey := statedb.ContractStatePrimaryKey(row.PrimaryKey())
	payer, data, err := row.Info()
	if err != nil {
		return nil, fmt.Errorf("unable to read contract state row %q value: %w", primaryKey, err)
	}

	primaryKeyString, err := convertKey(row.PrimaryKey(), keyConverter)
	if err != nil {
		return nil, fmt.Errorf("unable to convert key %s: %w", row.PrimaryKey(), err)
	}

	response := &pbstatedb.TableRowResponse{
		Key:   primaryKeyString,
		Payer: payer,
	}

	if withBlockNum {
		response.BlockNumber = row.Height()
	}

	response.Data = data
	if serializationInfo != nil {
		jsonData, err := serializationInfo.Decode(response.Data)
		if err != nil {
			zlog.Warn("failed to decode row from ABI",
				zap.Uint64("block_num", serializationInfo.abiAtBlockNum),
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

func convertKey(key []byte, keyConverter KeyConverter) (string, error) {
	if _, ok := keyConverter.(*NameKeyConverter); ok {
		return bytesToName(key), nil
	}

	return keyConverter.ToString(binary.BigEndian.Uint64(key))
}

func bytesToName(bytes []byte) string {
	return eos.NameToString(binary.BigEndian.Uint64(bytes))
}

func toContractStatePrimaryKey(in string, converter KeyConverter) (out statedb.ContractStatePrimaryKey, err error) {
	value, err := converter.FromString(in)
	if err != nil {
		return nil, fmt.Errorf("unable to convert key: %w", err)
	}

	out = make([]byte, 8)
	binary.BigEndian.PutUint64(out, value)
	return
}
