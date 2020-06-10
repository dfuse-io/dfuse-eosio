// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/derr"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	eos "github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (srv *EOSServer) prepareRead(
	ctx context.Context,
	blockNum uint32,
	irreversibleOnly bool,
) (chosenBlockNum uint32, lastWrittenBlockID string, upToBlockID string, speculativeWrites []*fluxdb.WriteRequest, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("performing prepare read operation")

	lastWrittenBlock, err := srv.db.FetchLastWrittenBlock(ctx)
	if err != nil {
		err = fmt.Errorf("unable to retrieve last written block id: %w", err)
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

func (srv *EOSServer) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = srv.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.String("head_block_id", headBlock.ID()), zap.Uint64("head_block_num", headBlock.Num()))

	return
}

func (srv *EOSServer) readContractStateTable(
	ctx context.Context,
	blockNum uint32,
	tablet fluxdb.ContractStateTablet,
	request *readRequestCommon,
	keyConverter KeyConverter,
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

	zlog.Debug("read rows results", zap.Int("row_count", len(tabletRows)))

	_, contract, _, table := tablet.Explode()
	abiEntry, err := srv.db.ReadSigletEntryAt(ctx, fluxdb.NewContractABISiglet(contract), blockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("read abi at: %w", err)
	}

	var abi *eos.ABI
	if abiEntry != nil {
		abi, err = abiEntry.(*fluxdb.ContractABIEntry).ABI()
		if err != nil {
			return nil, fmt.Errorf("decode abi: %w", err)
		}
	}

	out := &readTableResponse{}

	if request.WithABI {
		if abi == nil {
			return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
		}

		out.ABI = abi
	}

	tableTypeName := ""
	if request.ToJSON {
		if abi == nil {
			return nil, fluxdb.DataABINotFoundError(ctx, contract, blockNum)
		}

		tableDef := abi.TableForName(eos.TableName(table))
		if tableDef == nil {
			return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table))
		}

		tableTypeName = tableDef.Type
	}

	zlog.Debug("post-processing each table row (maybe convert to JSON)")
	for _, tabletRow := range tabletRows {
		contractStateRow := tabletRow.(*fluxdb.ContractStateRow)

		var data interface{}
		if request.ToJSON {
			data = &onTheFlyABISerializer{
				abi:             abi,
				abiAtBlockNum:   abiEntry.BlockNum(),
				tableTypeName:   tableTypeName,
				rowDataToDecode: contractStateRow.Data(),
			}
		} else {
			data = contractStateRow.Data()
		}

		var blockNum uint32
		if request.WithBlockNum {
			blockNum = contractStateRow.BlockNum()
		}

		rowKey, err := keyConverter.ToString(fluxdb.NA(eos.Name(contractStateRow.PrimaryKey())))
		if err != nil {
			return nil, fmt.Errorf("unable to convert key: %s", err)
		}

		out.Rows = append(out.Rows, &tableRow{
			Key:      rowKey,
			Payer:    contractStateRow.Payer(),
			Data:     data,
			BlockNum: blockNum,
		})
	}

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(out.Rows))),
	}, "read operation")

	return out, nil
}

func (srv *EOSServer) readContractStateTableRow(
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
		zap.Stringer("tablet", tablet),
		zap.String("primary_key", primaryKey),
		zap.Uint32("block_num", blockNum),
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

func (srv *EOSServer) fetchABI(
	ctx context.Context,
	account string,
	blockNum uint32,
) (*fluxdb.ContractABIEntry, error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare read: %w", err)
	}

	siglet := fluxdb.NewContractABISiglet(account)
	entry, err := srv.db.ReadSigletEntryAt(ctx, siglet, actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	return entry.(*fluxdb.ContractABIEntry), nil
}
