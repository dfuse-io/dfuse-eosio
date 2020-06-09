package grpc

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
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

	var abi *eos.ABI
	var abiAtBlockNum uint32
	var tableTypeName string

	if toJSON {
		_, contract, _, table := tablet.Explode()
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

	zlog.Debug("post-processing each tablet row (maybe convert to JSON)")
	keyConverter := getKeyConverterForType(keyType)

	out := &readTableResponse{}
	for _, tabletRow := range tabletRows {
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

		var blockNum uint32
		if withBlockNum {
			blockNum = row.BlockNum()
		}

		primaryKey, err := keyConverter.ToString(fluxdb.N(row.PrimaryKey()))
		if err != nil {
			return nil, fmt.Errorf("unable to convert key: %s", err)
		}

		out.Rows = append(out.Rows, &tableRow{
			Key:      primaryKey,
			Payer:    row.Payer(),
			Data:     data,
			BlockNum: blockNum,
		})
	}

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(out.Rows))),
	}, "read contract state tablet")

	return out, nil
}

func (srv *Server) readTableRow(
	ctx context.Context,
	blockNum uint32,
	account string,
	table string,
	scope string,
	primaryKey string,
	keyType string,
	withABI bool,
	toJSON bool,
	withBlockNum bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableRowResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug(
		"reading table row",
		zap.String("account", account),
		zap.String("table", table),
		zap.String("scope", scope),
		zap.String("primary_key", primaryKey),
	)

	keyConverter := getKeyConverterForType(keyType)

	primaryKeyValue, err := keyConverter.FromString(primaryKey)
	if err != nil {
		return nil, derr.Wrapf(err, "unable to convert key %q to uint64", primaryKey)
	}

	resp, err := srv.db.ReadTableRow(ctx, &fluxdb.ReadTableRowRequest{
		ReadTableRequest: fluxdb.ReadTableRequest{
			Account:           fluxdb.N(account),
			Scope:             fluxdb.EN(scope),
			Table:             fluxdb.N(table),
			BlockNum:          blockNum,
			SpeculativeWrites: speculativeWrites,
		},
		PrimaryKey: primaryKeyValue,
	})

	if err != nil {
		return nil, derr.Wrap(err, "unable to retrieve single row from database")
	}

	var abiObj *eos.ABI
	// if err := eos.UnmarshalBinary(resp.ABI.PackedABI, &abiObj); err != nil {
	// 	return nil, derr.Wrapf(err, "unable to decode packed ABI %q to JSON", resp.ABI.PackedABI)
	// }

	out := &readTableRowResponse{}
	if withABI {
		out.ABI = abiObj
	}

	tableName := eos.TableName(table)
	tableDef := abiObj.TableForName(tableName)
	if tableDef == nil {
		return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(account), tableName)
	}

	if resp.Row == nil {
		zlogger.Debug("row deleted or never existed")
		return out, nil
	}

	rowKey, err := keyConverter.ToString(resp.Row.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to convert key: %s", err)
	}

	zlogger.Debug("post-processing row (maybe convert to JSON)")
	out.Row = &tableRow{
		Key:   rowKey,
		Payer: fluxdb.NameToString(resp.Row.Payer),
		Data:  resp.Row.Data,
	}

	if toJSON {
		out.Row.Data = &onTheFlyABISerializer{
			abi: abiObj,
			// abiRow:     resp.ABI.,
			tableTypeName:   tableDef.Type,
			rowDataToDecode: resp.Row.Data,
		}
	}

	if withBlockNum {
		out.Row.BlockNum = resp.Row.BlockNum
	}

	return out, nil
}

func (srv *Server) readTable(
	ctx context.Context,
	blockNum uint32,
	account string,
	table string,
	scope string,
	keyType string,
	withABI bool,
	toJSON bool,
	withBlockNum bool,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableResponse, error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading rows", zap.String("account", account), zap.String("table", table), zap.String("scope", scope))

	keyConverter := getKeyConverterForType(keyType)

	resp, err := srv.db.ReadTable(ctx, &fluxdb.ReadTableRequest{
		Account:           fluxdb.N(account),
		Scope:             fluxdb.EN(scope),
		Table:             fluxdb.N(table),
		BlockNum:          blockNum,
		SpeculativeWrites: speculativeWrites,
	})

	if err != nil {
		return nil, derr.Wrap(err, "unable to retrieve rows from database")
	}

	zlog.Debug("read rows results", zap.Int("row_count", len(resp.Rows)))

	var abiObj *eos.ABI
	// if err := eos.UnmarshalBinary(resp.ABI.PackedABI, &abiObj); err != nil {
	// 	return nil, derr.Wrapf(err, "unable to decode packed ABI %q to JSON", resp.ABI.PackedABI)
	// }

	out := &readTableResponse{}
	if withABI {
		out.ABI = abiObj
	}

	tableName := eos.TableName(table)
	tableDef := abiObj.TableForName(tableName)
	if tableDef == nil {
		return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(account), tableName)
	}

	zlog.Debug("post-processing each row (maybe convert to JSON)")
	for _, row := range resp.Rows {
		var data interface{}
		if toJSON {
			data = &onTheFlyABISerializer{
				abi: abiObj,
				// abiRow:          resp.ABI,
				tableTypeName:   tableDef.Type,
				rowDataToDecode: row.Data,
			}
		} else {
			data = row.Data
		}

		var blockNum uint32
		if withBlockNum {
			blockNum = row.BlockNum
		}

		rowKey, err := keyConverter.ToString(row.Key)
		if err != nil {
			return nil, fmt.Errorf("unable to convert key: %s", err)
		}

		out.Rows = append(out.Rows, &tableRow{
			Key:      rowKey,
			Payer:    fluxdb.NameToString(row.Payer),
			Data:     data,
			BlockNum: blockNum,
		})
	}

	return out, nil
}

func (srv *Server) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = srv.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.String("head_block_id", headBlock.ID()), zap.Uint64("head_block_num", headBlock.Num()))

	return
}

func (srv *Server) fetchABI(
	ctx context.Context,
	account string,
	blockNum uint32,
) (*fluxdb.ContractABIEntry, error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, fmt.Errorf("speculative writes: %w", err)
	}

	siglet := fluxdb.NewContractABISiglet(account)
	entry, err := srv.db.ReadSigletEntryAt(ctx, siglet, actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	return entry.(*fluxdb.ContractABIEntry), nil
}
