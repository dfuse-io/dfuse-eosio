package grpc

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (srv *Server) prepareRead(
	ctx context.Context,
	blockNum uint32,
	irreversibleOnly bool,
) (chosenBlockNum uint32, lastWrittenBlockID string, upToBlockID string, speculativeWrites []*fluxdb.WriteRequest, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("performing prepare read operation")

	lastWrittenBlock, err := srv.db.FetchLastWrittenBlock(ctx)
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

	headBlock := srv.fetchHeadBlock(ctx, zlog)
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
	speculativeWrites = srv.db.SpeculativeWritesFetcher(ctx, headBlock.ID(), chosenBlockNum)

	if len(speculativeWrites) >= 1 {
		upToBlockID = hex.EncodeToString(speculativeWrites[len(speculativeWrites)-1].BlockID)
		zlog.Debug("speculative writes present",
			zap.Int("speculative_write_count", len(speculativeWrites)),
			zap.String("up_to_block_id", upToBlockID),
		)
	}

	return
}

func (srv *Server) readContractStateTable(
	ctx context.Context,
	tablet fluxdb.ContractStateTablet,
	blockNum uint32,
	keyType string,
	toJSON bool,
	withBlockNum bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableResponse, error) {
	ctx, span := dtracing.StartSpan(ctx, "read contract state table")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("read contract state tablet", zap.Stringer("tablet", tablet))

	tabletRows, err := srv.db.ReadTabletAt(
		ctx,
		blockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		return nil, fmt.Errorf("read tablet at: %w", err)
	}

	zlog.Debug("read tablet rows results", zap.Int("row_count", len(tabletRows)))

	serializationInfo := rowSerializationInfo{
		toJSON: toJSON,
	}

	if toJSON {
		_, contract, _, table := tablet.Explode()
		abiEntry, err := srv.db.ReadSigletEntryAt(ctx, fluxdb.NewContractABISiglet(contract), blockNum, speculativeWrites)
		if err != nil {
			return nil, fmt.Errorf("read abi at: %w", err)
		}

		if abiEntry == nil {
			return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
		}

		serializationInfo.abi, err = abiEntry.(*fluxdb.ContractABIEntry).ABI()
		if err != nil {
			return nil, fmt.Errorf("decode abi: %w", err)
		}

		if serializationInfo.abi == nil {
			return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
		}

		tableDef := serializationInfo.abi.TableForName(eos.TableName(table))
		if tableDef == nil {
			return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table))
		}

		serializationInfo.abiAtBlockNum = abiEntry.BlockNum()
		serializationInfo.tableTypeName = tableDef.Type
	}

	zlog.Debug("post-processing each tablet row (maybe convert to JSON)")
	keyConverter := getKeyConverterForType(keyType)

	out := &readTableResponse{
		Rows: make([]*tableRow, len(tabletRows)),
	}

	for i, tabletRow := range tabletRows {
		row, err := newTableRowResponse(tabletRow.(*fluxdb.ContractStateRow), keyConverter, &serializationInfo, withBlockNum)
		if err != nil {
			return nil, fmt.Errorf("unable to create table row: %w", err)
		}

		out.Rows[i] = row
	}

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(out.Rows))),
	}, "read contract state tablet")

	return out, nil
}

type rowSerializationInfo struct {
	toJSON        bool
	abi           *eos.ABI
	abiAtBlockNum uint32
	tableTypeName string
}

func newTableRowResponse(
	row *fluxdb.ContractStateRow,
	keyConverter KeyConverter,
	serializationInfo *rowSerializationInfo,
	withBlockNum bool,
) (*tableRow, error) {
	var data interface{}
	if serializationInfo.toJSON {
		data = &onTheFlyABISerializer{
			abi:             serializationInfo.abi,
			abiAtBlockNum:   serializationInfo.abiAtBlockNum,
			tableTypeName:   serializationInfo.tableTypeName,
			rowDataToDecode: row.Data(),
		}
	} else {
		data = row.Data()
	}

	var blockNum uint32
	if withBlockNum {
		blockNum = row.BlockNum()
	}

	primaryKey, err := keyConverter.ToString(fluxdb.N(row.PrimaryKey()))
	if err != nil {
		return nil, fmt.Errorf("unable to convert key: %s", err)
	}

	return &tableRow{
		Key:      primaryKey,
		Payer:    row.Payer(),
		Data:     data,
		BlockNum: blockNum,
	}, nil
}

func (srv *Server) readContractStateTableRow(
	ctx context.Context,
	tablet fluxdb.ContractStateTablet,
	blockNum uint32,
	keyType string,
	primaryKey string,
	toJSON bool,
	withBlockNum bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug(
		"reading contract state table row",
		zap.String("table_key", tablet.Key()),
		zap.String("primary_key", primaryKey),
		zap.Uint32("block_nume", blockNum),
	)

	keyConverter := getKeyConverterForType(keyType)

	primaryKeyValue, err := keyConverter.FromString(primaryKey)
	if err != nil {
		return nil, derr.Wrapf(err, "unable to convert key %q to uint64", primaryKey)
	}

	tabletRow, err := srv.db.ReadTabletRowAt(
		ctx,
		blockNum,
		tablet,
		fluxdb.UN(primaryKeyValue),
		speculativeWrites,
	)
	if err != nil {
		return nil, derr.Wrap(err, "unable to retrieve single row from database")
	}

	_, contract, scope, table := tablet.Explode()
	zlog.Debug("read tablet row result",
		zap.String("contract", contract),
		zap.String("table", table),
		zap.String("scope", scope),
		zap.String("scope", primaryKey),
	)

	if tabletRow == nil {
		zlogger.Debug("row deleted or never existed")
		return nil, fluxdb.DataRowNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table), eos.AccountName(scope), primaryKey)
	}

	var abi *eos.ABI
	var abiAtBlockNum uint32
	var tableTypeName string

	if toJSON {

		abiEntry, err := srv.db.ReadSigletEntryAt(ctx, fluxdb.NewContractABISiglet(contract), blockNum, speculativeWrites)
		if err != nil {
			return nil, fmt.Errorf("read abi at: %w", err)
		}

		if abiEntry == nil {
			return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
		}

		abi, err = abiEntry.(*fluxdb.ContractABIEntry).ABI()
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

		abiAtBlockNum = abiEntry.BlockNum()
		tableTypeName = tableDef.Type

	}

	zlog.Debug("post-processing tablet row (maybe convert to JSON)")

	row := tabletRow.(*fluxdb.ContractStateRow)

	var data interface{}
	if toJSON {
		data = &onTheFlyABISerializer{
			abi:             abi,
			abiAtBlockNum:   abiAtBlockNum,
			tableTypeName:   tableTypeName,
			rowDataToDecode: row.Data(),
		}
	} else {
		data = row.Data()
	}

	var rowBlockNum uint32
	if withBlockNum {
		rowBlockNum = row.BlockNum()
	}

	return &readTableRowResponse{
		Row: &tableRow{
			Key:      primaryKey,
			Data:     data,
			Payer:    row.Payer(),
			BlockNum: rowBlockNum,
		},
	}, nil
}

func (srv *Server) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = srv.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.String("head_block_id", headBlock.ID()), zap.Uint64("head_block_num", headBlock.Num()))

	return
}
