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
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/fluxdb"
	"github.com/dfuse-io/logging"
	eos "github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (srv *EOSServer) prepareRead(
	ctx context.Context,
	blockNum uint64,
	irreversibleOnly bool,
) (chosenBlockNum uint64, lastWrittenBlock bstream.BlockRef, upToBlock bstream.BlockRef, speculativeWrites []*fluxdb.WriteRequest, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("performing prepare read operation")

	_, lastWrittenBlock, err = srv.db.FetchLastWrittenCheckpoint(ctx)
	if err != nil {
		err = fmt.Errorf("unable to retrieve last written block id: %w", err)
		return
	}

	lastWrittenBlockNum := lastWrittenBlock.Num()
	if irreversibleOnly {
		// TODO: Maybe we should insteand honor the request and move it back to last irreversible block...
		if blockNum > lastWrittenBlockNum {
			err = statedb.AppBlockNumHigherThanLIBError(ctx, blockNum, lastWrittenBlockNum)
			return
		}
		if chosenBlockNum == 0 {
			chosenBlockNum = lastWrittenBlockNum
		}
		return
	}

	headBlock := srv.fetchHeadBlock(ctx, zlog)
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
	zlog.Debug("fetching speculative writes", zap.Stringer("head_block", headBlock), zap.Stringer("lib_block", lastWrittenBlock), zap.Uint64("chosen_block_num", chosenBlockNum))
	speculativeWrites = srv.db.SpeculativeWritesFetcher(ctx, headBlock.ID(), chosenBlockNum)

	if len(speculativeWrites) >= 1 {
		lastSpeculativeWrite := speculativeWrites[len(speculativeWrites)-1]
		upToBlock = lastSpeculativeWrite.BlockRef
		zlog.Debug("speculative writes present",
			zap.Int("speculative_write_count", len(speculativeWrites)),
			zap.Stringer("up_to_block", upToBlock),
		)
	}

	return
}

func (srv *EOSServer) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = srv.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.Stringer("head_block", headBlock))

	return
}

func (srv *EOSServer) readContractStateTable(
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

	tabletRows, err := srv.db.ReadTabletAt(
		ctx,
		blockNum,
		tablet,
		speculativeWrites,
	)
	if err != nil {
		if errors.Is(err, fluxdb.ErrTemporarlyExcludedRange) {
			contract, table, scope := tablet.Explode()
			return nil, nil, statedb.HackDataTableExcludedError(ctx, eos.AccountName(contract), eos.TableName(table), eos.AccountName(scope))
		}

		return nil, nil, fmt.Errorf("read tablet at: %w", err)
	}

	zlog.Debug("read tablet rows results", zap.Int("row_count", len(tabletRows)))

	var serializationInfo *rowSerializationInfo
	if toJSON {
		contract, table, _ := tablet.Explode()
		serializationInfo, err = srv.newRowSerializationInfo(ctx, contract, table, blockNum, speculativeWrites)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to obtain serialziation info: %w", err)
		}
	}

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(tabletRows))),
	}, "read contract state tablet")

	return tabletRows, serializationInfo, nil
}

func (s *EOSServer) readContractStateTableRow(
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
		zap.Stringer("table_key", tablet),
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
		return nil, nil, statedb.DataRowNotFoundError(ctx, eos.AccountName(contract), eos.TableName(table), eos.AccountName(scope), primaryKey)
	}

	var serializationInfo *rowSerializationInfo
	if toJSON {
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

func (s *EOSServer) newRowSerializationInfo(ctx context.Context, contract, table string, blockNum uint64, speculativeWrites []*fluxdb.WriteRequest) (*rowSerializationInfo, error) {
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

func (srv *EOSServer) fetchABI(
	ctx context.Context,
	account string,
	blockNum uint64,
) (*statedb.ContractABIEntry, error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare read: %w", err)
	}

	singlet := statedb.NewContractABISinglet(account)
	entry, err := srv.db.ReadSingletEntryAt(ctx, singlet, actualBlockNum, speculativeWrites)
	if err != nil {
		return nil, fmt.Errorf("db read: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	return entry.(*statedb.ContractABIEntry), nil
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
