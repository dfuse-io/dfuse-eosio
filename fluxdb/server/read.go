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

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/derr"
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

func (srv *EOSServer) fetchHeadBlock(ctx context.Context, zlog *zap.Logger) (headBlock bstream.BlockRef) {
	headBlock = srv.db.HeadBlock(ctx)
	zlog.Debug("retrieved head block id", zap.String("head_block_id", headBlock.ID()), zap.Uint64("head_block_num", headBlock.Num()))

	return
}

func (srv *EOSServer) readTable(
	ctx context.Context,
	blockNum uint32,
	account string,
	table string,
	scope string,
	request *readRequestCommon,
	keyConverter KeyConverter,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableResponse, error) {
	ctx, span := dtracing.StartSpan(ctx, "read rows")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading rows", zap.String("account", account), zap.String("table", table), zap.String("scope", scope))

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
	if err := eos.UnmarshalBinary(resp.ABI.PackedABI, &abiObj); err != nil {
		return nil, derr.Wrapf(err, "unable to decode packed ABI %q to JSON", resp.ABI.PackedABI)
	}

	out := &readTableResponse{}
	if request.WithABI {
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
		if request.ToJSON {
			data = &onTheFlyABISerializer{
				abi:        abiObj,
				abiRow:     resp.ABI,
				structType: tableDef.Type,
				data:       row.Data,
			}
		} else {
			data = row.Data
		}

		var blockNum uint32
		if request.WithBlockNum {
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

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("rows", int64(len(out.Rows))),
	}, "read operation")

	return out, nil
}

func (srv *EOSServer) readTableRow(
	ctx context.Context,
	blockNum uint32,
	account string,
	table string,
	scope string,
	primaryKey string,
	request *readRequestCommon,
	keyConverter KeyConverter,
	speculativeWrites []*fluxdb.WriteRequest,
) (*readTableRowResponse, error) {
	ctx, span := dtracing.StartSpan(ctx, "read table row")
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading table row", zap.String("account", account), zap.String("table", table), zap.String("scope", scope), zap.String("primary_key", primaryKey))

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
	if err := eos.UnmarshalBinary(resp.ABI.PackedABI, &abiObj); err != nil {
		return nil, derr.Wrapf(err, "unable to decode packed ABI %q to JSON", resp.ABI.PackedABI)
	}

	out := &readTableRowResponse{}
	if request.WithABI {
		out.ABI = abiObj
	}

	tableName := eos.TableName(table)
	tableDef := abiObj.TableForName(tableName)
	if tableDef == nil {
		return nil, fluxdb.DataTableNotFoundError(ctx, eos.AccountName(account), tableName)
	}

	if resp.Row == nil {
		zlog.Debug("row deleted or never existed")
		return out, nil
	}

	rowKey, err := keyConverter.ToString(resp.Row.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to convert key: %s", err)
	}

	zlog.Debug("post-processing row (maybe convert to JSON)")
	out.Row = &tableRow{
		Key:   rowKey,
		Payer: fluxdb.NameToString(resp.Row.Payer),
		Data:  resp.Row.Data,
	}

	if request.ToJSON {
		out.Row.Data = &onTheFlyABISerializer{
			abi:        abiObj,
			abiRow:     resp.ABI,
			structType: tableDef.Type,
			data:       resp.Row.Data,
		}
	}

	if request.WithBlockNum {
		out.Row.BlockNum = resp.Row.BlockNum
	}

	return out, nil
}

func (srv *EOSServer) listKeyAccounts(
	ctx context.Context,
	publicKey string,
	blockNum uint32,
) (accountNames []eos.AccountName, actualBlockNum uint32, err error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		err = derr.Wrap(err, "unable to prepare read")
		return
	}

	accountNames, err = srv.db.ReadKeyAccounts(ctx, uint32(actualBlockNum), publicKey, speculativeWrites)
	if err != nil {
		err = derr.Wrap(err, "unable to read key accounts from db")
		return
	}

	if len(accountNames) == 0 {
		seen, err := srv.db.HasSeenPublicKeyOnce(ctx, publicKey)
		if err != nil {
			return nil, actualBlockNum, derr.Wrap(err, "unable to know if public key was seen once in db")
		}

		if !seen {
			return nil, actualBlockNum, fluxdb.DataPublicKeyNotFoundError(ctx, publicKey)
		}
	}

	return
}

func (srv *EOSServer) listTableScopes(
	ctx context.Context,
	account eos.AccountName,
	table eos.TableName,
	blockNum uint32,
) (scopes []eos.Name, actualBlockNum uint32, err error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		err = derr.Wrap(err, "unable to prepare read")
		return
	}

	scopes, err = srv.db.ReadTableScopes(ctx, uint32(actualBlockNum), account, table, speculativeWrites)
	if err != nil {
		err = derr.Wrap(err, "unable to read table scopes from db")
		return
	}

	if len(scopes) == 0 {
		logging.Logger(ctx, zlog).Debug("no scopes found for request, checking if we ever see this table")
		seen, err := srv.db.HasSeenTableOnce(ctx, account, table)
		if err != nil {
			return nil, actualBlockNum, derr.Wrap(err, "unable to know if table was seen once in db")
		}

		if !seen {
			return nil, actualBlockNum, fluxdb.DataTableNotFoundError(ctx, account, table)
		}
	}

	return
}

func (srv *EOSServer) fetchABI(
	ctx context.Context,
	account string,
	blockNum uint32,
	toJSON bool,
) (abiRow *fluxdb.ABIRow, abiObj *eos.ABI, err error) {
	actualBlockNum, _, _, speculativeWrites, err := srv.prepareRead(ctx, blockNum, false)
	if err != nil {
		return
	}

	abiRow, err = srv.db.GetABI(ctx, uint32(actualBlockNum), fluxdb.N(account), speculativeWrites)
	if err != nil {
		err = derr.Wrap(err, "fetching ABI from db")
		return
	}

	if toJSON {
		if err = eos.UnmarshalBinary(abiRow.PackedABI, &abiObj); err != nil {
			err = derr.Wrapf(err, "failed to decode packed ABI %q to JSON", abiRow.PackedABI)
			return
		}
	}

	return
}
