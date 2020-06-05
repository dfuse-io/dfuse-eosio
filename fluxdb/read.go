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

package fluxdb

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

// func (fdb *FluxDB) GetABI(ctx context.Context, blockNum uint32, account uint64, speculativeWrites []*WriteRequest) (out *ABIRow, err error) {
// 	ctx, span := dtracing.StartSpan(ctx, "get abi", "account", eos.NameToString(account), "block_num", blockNum)
// 	defer span.End()

// 	zlog := logging.Logger(ctx, zlog)
// 	zlog.Debug("fetching ABI", zap.Uint64("account", account), zap.Uint32("block_num", blockNum))

// 	out = &ABIRow{
// 		Account: account,
// 	}

// 	prefixKey := HexName(account) + ":"
// 	firstKey := prefixKey + HexRevBlockNum(blockNum)
// 	lastKey := prefixKey + HexRevBlockNum(0)

// 	zlog.Debug("reading ABI rows", zap.String("first_key", firstKey), zap.String("last_key", lastKey))
// 	rowKey, rawABI, err := fdb.store.FetchABI(ctx, prefixKey, firstKey, lastKey)
// 	if err != nil && err != store.ErrNotFound {
// 		return nil, err
// 	}

// 	if err != store.ErrNotFound {
// 		abiBlockNum, err := chunkKeyRevBlockNum(rowKey, prefixKey)
// 		if err != nil {
// 			return nil, fmt.Errorf("couldn't infer block num in table ABI's row key: %w", err)
// 		}

// 		out.BlockNum = abiBlockNum
// 		out.PackedABI = rawABI
// 	}

// 	zlog.Debug("handling speculative writes", zap.Int("write_count", len(speculativeWrites)))
// 	for _, blockWrite := range speculativeWrites {
// 		for _, speculativeABI := range blockWrite.ABIs {
// 			if speculativeABI.Account == account {
// 				zlog.Debug("updating ABI", zap.Uint32("block_num", blockWrite.BlockNum))
// 				out = speculativeABI
// 			}
// 		}
// 	}

// 	if len(out.PackedABI) == 0 {
// 		return nil, DataABINotFoundError(ctx, eos.NameToString(account), blockNum)
// 	}

// 	return
// }

func (fdb *FluxDB) ReadTable(ctx context.Context, r *ReadTableRequest) (resp *ReadTableResponse, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading state table", zap.Reflect("request", r))

	rowData := make(map[string]*TableRow)
	rowUpdated := func(blockNum uint32, primaryKey string, value []byte) error {
		if len(value) < 8 {
			return errors.New("table data index mappings should contain at least the payer")
		}

		payer := big.Uint64(value)
		tableDataPrimaryKey, err := strconv.ParseUint(primaryKey, 16, 64)
		if err != nil {
			return fmt.Errorf("unable to transform table data primary key to uint64: %w", err)
		}

		rowData[primaryKey] = &TableRow{tableDataPrimaryKey, payer, value[8:], blockNum}

		return nil
	}

	rowDeleted := func(_ uint32, primaryKey string) error {
		delete(rowData, primaryKey)
		return nil
	}

	tableKey := r.tableKey()
	err = fdb.read(ctx, tableKey, r.BlockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read rows for table key %q: %w", tableKey, err)
	}

	// abi, err := fdb.GetABI(ctx, r.BlockNum, r.Account, r.SpeculativeWrites)
	// if err != nil {
	// 	return nil, err
	// }

	// zlog.Debug("handling speculative writes", zap.Int("write_count", len(r.SpeculativeWrites)))
	// for _, blockWrite := range r.SpeculativeWrites {
	// 	for _, row := range blockWrite.FluxRows {
	// 		if r.Account != row.Account || r.Scope != row.Scope || r.Table != row.Table {
	// 			continue
	// 		}

	// 		stringPrimaryKey := fmt.Sprintf("%016x", row.PrimKey)

	// 		if row.Deletion {
	// 			delete(rowData, stringPrimaryKey)
	// 		} else {
	// 			rowData[stringPrimaryKey] = &TableRow{
	// 				Key:      row.PrimKey,
	// 				Payer:    row.Payer,
	// 				Data:     row.Data,
	// 				BlockNum: blockWrite.BlockNum,
	// 			}
	// 		}
	// 	}
	// }

	zlog.Debug("post-processing table rows", zap.Int("row_count", len(rowData)))
	var rows []*TableRow
	for _, row := range rowData {
		rows = append(rows, row)
	}

	zlog.Debug("sorting table rows")
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })

	return &ReadTableResponse{
		// ABI:  abi,
		Rows: rows,
	}, nil
}

func (fdb *FluxDB) ReadTableRow(ctx context.Context, r *ReadTableRowRequest) (resp *ReadTableRowResponse, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading state table row", zap.Reflect("request", r))

	primaryKeyString := r.primaryKeyString()

	var rowData *TableRow
	rowUpdated := func(blockNum uint32, candidatePrimaryKey string, value []byte) error {
		if len(value) < 8 {
			return errors.New("table data index mappings should contain at least the payer")
		}

		if candidatePrimaryKey != primaryKeyString {
			return errors.New("logic error, should never happen, the read single should yield only results related to primary key")
		}

		payer := big.Uint64(value)
		data := value[8:]

		rowData = &TableRow{r.PrimaryKey, payer, data, blockNum}
		return nil
	}

	rowDeleted := func(_ uint32, candidatePrimaryKey string) error {
		if candidatePrimaryKey != primaryKeyString {
			return errors.New("logic error, should never happen, the read single should yield only results related to primary key")
		}

		rowData = nil
		return nil
	}

	tableKey := r.tableKey()
	err = fdb.readSingle(ctx, tableKey, primaryKeyString, r.BlockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read single row for table key %q and primary key %d: %w", tableKey, r.PrimaryKey, err)
	}

	// zlog.Debug("handling speculative writes", zap.Int("write_count", len(r.SpeculativeWrites)))
	// for _, blockWrite := range r.SpeculativeWrites {
	// 	for _, row := range blockWrite.TableDatas {
	// 		if r.Account != row.Account || r.Scope != row.Scope || r.Table != row.Table || r.PrimaryKey != row.PrimKey {
	// 			continue
	// 		}

	// 		if row.Deletion {
	// 			rowData = nil
	// 		} else {
	// 			rowData = &TableRow{
	// 				Key:      row.PrimKey,
	// 				Payer:    row.Payer,
	// 				Data:     row.Data,
	// 				BlockNum: blockWrite.BlockNum,
	// 			}
	// 		}
	// 	}
	// }

	// This was added when fixing a bug with `/state/table/row` since the old location where it was used
	// was not the right place. But when it was moved, it changed the behavior of the old API causing problem
	// to existing customer. To retain old behavior, we now return an empty row data in all cases when a specific
	// key is not found on a given table.
	// if rowData == nil {
	// 	return nil, DataRowNotFoundError(ctx, eos.AccountName(eos.NameToString(r.Account)), eos.TableName(eos.NameToString(r.Table)), eos.NameToString(r.PrimaryKey))
	// }

	// abi, err := fdb.GetABI(ctx, r.BlockNum, r.Account, r.SpeculativeWrites)
	// if err != nil {
	// 	return nil, err
	// }

	return &ReadTableRowResponse{
		// ABI: abi,
		Row: rowData,
	}, nil
}

func (fdb *FluxDB) HasSeenPublicKeyOnce(
	ctx context.Context,
	publicKey string,
) (exists bool, err error) {
	return fdb.hasRowKeyPrefix(ctx, fmt.Sprintf("ka2:%s", publicKey))
}

func (fdb *FluxDB) ReadKeyAccounts(
	ctx context.Context,
	blockNum uint32,
	publicKey string,
	speculativeWrites []*WriteRequest,
) (accountNames []eos.AccountName, err error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("reading key accounts",
		zap.String("public_key", string(publicKey)),
		zap.Uint32("block_num", blockNum),
	)

	rows := map[string]interface{}{}
	rowUpdated := func(_ uint32, primaryKey string, _ []byte) error {
		zlogger.Debug("row updated", zap.String("primary_key", primaryKey))
		rows[primaryKey] = nil
		return nil
	}

	rowDeleted := func(_ uint32, primaryKey string) error {
		zlogger.Debug("row deleted", zap.String("primary_key", primaryKey))
		delete(rows, primaryKey)
		return nil
	}

	tableKey := fmt.Sprintf("ka2:%s", publicKey)
	err = fdb.read(ctx, tableKey, blockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read rows for table key %q: %w", tableKey, err)
	}

	// zlogger.Debug("handling speculative writes", zap.Int("write_count", len(speculativeWrites)))
	// for _, blockWrite := range speculativeWrites {
	// 	for _, keyAccountRow := range blockWrite.KeyAccounts {
	// 		if keyAccountRow.PublicKey != publicKey {
	// 			continue
	// 		}

	// 		zlogger.Debug("updating key account", zap.Reflect("table_scope_row", keyAccountRow))
	// 		stringPrimaryKey := fmt.Sprintf("%016x:%016x", keyAccountRow.Account, keyAccountRow.Permission)

	// 		if keyAccountRow.Deletion {
	// 			delete(rows, stringPrimaryKey)
	// 		} else {
	// 			rows[stringPrimaryKey] = nil
	// 		}
	// 	}
	// }

	zlogger.Debug("post-processing key accounts", zap.Int("key_account_count", len(rows)))
	buffer := make([]byte, indexPrimaryKeyByteCountByTableKey("ka2:"))

	accountNameSet := map[string]bool{}
	for primaryKey := range rows {
		err := keyAccountIndexPrimaryKeyWriter(primaryKey, buffer)
		if err != nil {
			return nil, fmt.Errorf("unable to transform key account primary key %s: %w", primaryKey, err)
		}

		accountNameSet[eos.NameToString(big.Uint64(buffer))] = true
	}

	for account := range accountNameSet {
		accountNames = append(accountNames, eos.AccountName(account))
	}

	zlogger.Debug("sorting key accounts")
	sort.Slice(accountNames, func(i, j int) bool {
		return accountNames[i] < accountNames[j]
	})

	return accountNames, nil
}

func (fdb *FluxDB) ReadLinkedPermissions(ctx context.Context, blockNum uint32, account eos.AccountName, speculativeWrites []*WriteRequest) (resp []*LinkedPermission, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading linked permissions", zap.String("account", string(account)), zap.Uint32("block_num", blockNum))

	rowData := make(map[string]*LinkedPermission)
	rowUpdated := func(_ uint32, primaryKey string, value []byte) error {
		primaryKeyBuffer := make([]byte, indexPrimaryKeyByteCountByTableKey("al:"))
		err := authLinkIndexPrimaryKeyWriter(primaryKey, primaryKeyBuffer)
		if err != nil {
			return fmt.Errorf("unable to transform auth link primary key: %w", err)
		}

		contract := big.Uint64(primaryKeyBuffer)
		action := big.Uint64(primaryKeyBuffer[8:])
		permissionName := big.Uint64(value)

		rowData[primaryKey] = &LinkedPermission{
			Contract:       eos.NameToString(contract),
			Action:         eos.NameToString(action),
			PermissionName: eos.NameToString(permissionName),
		}
		return nil
	}

	rowDeleted := func(_ uint32, primaryKey string) error {
		delete(rowData, primaryKey)
		return nil
	}

	tableKey := fmt.Sprintf("al:%016x", N(string(account)))
	err = fdb.read(ctx, tableKey, blockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read rows for table key %q: %w", tableKey, err)
	}

	// zlog.Debug("handling speculative writes", zap.Int("write_count", len(speculativeWrites)))
	// for _, blockWrite := range speculativeWrites {
	// 	for _, row := range blockWrite.AuthLinks {
	// 		if row.Account != N(string(account)) {
	// 			continue
	// 		}

	// 		zlog.Debug("updating auth link", zap.Reflect("auth_link_row", row))
	// 		stringPrimaryKey := fmt.Sprintf("%016x:%016x", row.Contract, row.Action)

	// 		if row.Deletion {
	// 			delete(rowData, stringPrimaryKey)
	// 		} else {
	// 			rowData[stringPrimaryKey] = &LinkedPermission{
	// 				Contract:       eos.NameToString(row.Contract),
	// 				Action:         eos.NameToString(row.Action),
	// 				PermissionName: eos.NameToString(row.PermissionName),
	// 			}
	// 		}
	// 	}
	// }

	zlog.Debug("post-processing linked permissions", zap.Int("link_permission_count", len(rowData)))
	var output []*LinkedPermission
	for _, row := range rowData {
		output = append(output, row)
	}

	zlog.Debug("sorting linked permissions")
	sort.Slice(output, func(i, j int) bool {
		if output[i].Contract == output[j].Contract {
			return output[i].Action < output[j].Action
		}

		return output[i].Contract < output[j].Contract
	})

	return output, nil
}

func (fdb *FluxDB) HasSeenTableOnce(
	ctx context.Context,
	account eos.AccountName,
	table eos.TableName,
) (exists bool, err error) {
	return fdb.hasRowKeyPrefix(ctx, fmt.Sprintf("ts:%016x:%016x", N(string(account)), N(string(table))))
}

func (fdb *FluxDB) ReadTableScopes(
	ctx context.Context,
	blockNum uint32,
	account eos.AccountName,
	table eos.TableName,
	speculativeWrites []*WriteRequest,
) (scopes []eos.Name, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading table scopes",
		zap.String("account", string(account)),
		zap.String("table", string(table)),
		zap.Uint32("block_num", blockNum),
	)

	rows := map[string]interface{}{}
	rowUpdated := func(_ uint32, primaryKey string, _ []byte) error {
		rows[primaryKey] = nil
		return nil
	}

	rowDeleted := func(_ uint32, primaryKey string) error {
		delete(rows, primaryKey)
		return nil
	}

	accountName := N(string(account))
	tableName := N(string(table))

	tableKey := fmt.Sprintf("ts:%016x:%016x", accountName, tableName)
	err = fdb.read(ctx, tableKey, blockNum, rowUpdated, rowDeleted)
	if err != nil {
		return nil, fmt.Errorf("unable to read rows for table key %q: %w", tableKey, err)
	}

	// zlog.Debug("handling speculative writes", zap.Int("write_count", len(speculativeWrites)))
	// for _, blockWrite := range speculativeWrites {
	// 	for _, tableScopeRow := range blockWrite.TableScopes {
	// 		if tableScopeRow.Account != accountName || tableScopeRow.Table != tableName {
	// 			continue
	// 		}

	// 		zlog.Debug("updating table scope", zap.Reflect("table_scope_row", tableScopeRow))
	// 		stringPrimaryKey := fmt.Sprintf("%016x", tableScopeRow.Scope)

	// 		if tableScopeRow.Deletion {
	// 			delete(rows, stringPrimaryKey)
	// 		} else {
	// 			rows[stringPrimaryKey] = nil
	// 		}
	// 	}
	// }

	zlog.Debug("post-processing table scopes", zap.Int("table_scope_count", len(rows)))
	buffer := make([]byte, indexPrimaryKeyByteCountByTableKey("ts:"))

	for primaryKey := range rows {
		err := tableScopeIndexPrimaryKeyWriter(primaryKey, buffer)
		if err != nil {
			return nil, fmt.Errorf("unable to transform table scope primary key: %w", err)
		}

		scopes = append(scopes, eos.Name(eos.NameToString(big.Uint64(buffer))))
	}

	zlog.Debug("sorting table scopes")
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i] < scopes[j]
	})

	return scopes, nil
}

func (fdb *FluxDB) hasRowKeyPrefix(ctx context.Context, keyPrefix string) (exists bool, err error) {
	ctx, span := dtracing.StartSpan(ctx, "has row key prefix", "key_prefix", keyPrefix)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("has row key prefix", zap.String("key_prefix", keyPrefix))

	return fdb.store.HasTabletRow(ctx, keyPrefix)
}

func (fdb *FluxDB) read(
	ctx context.Context,
	tableKey string,
	blockNum uint32,
	rowUpdated func(blockNum uint32, primaryKey string, value []byte) error,
	rowDeleted func(blockNum uint32, primaryKey string) error,
) error {
	ctx, span := dtracing.StartSpan(ctx, "read table", "table_key", tableKey, "block_num", blockNum)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading rows from database", zap.String("table_key", tableKey), zap.Uint32("block_num", blockNum))

	idx, err := fdb.getIndex(ctx, tableKey, blockNum)
	if err != nil {
		return err
	}

	firstRowKey := tableKey + ":00000000"
	lastRowKey := tableKey + ":" + HexBlockNum(blockNum+1)

	if idx != nil {
		zlog.Debug("index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		firstRowKey = tableKey + "/" + HexBlockNum(idx.AtBlockNum+1)

		var keys []string
		for primaryKey, blockNum := range idx.Map {
			keys = append(keys, fmt.Sprintf("%s:%08x:%s", tableKey, blockNum, primaryKey))
		}

		// Fetch all rows in the index.. could be millions
		// We need to batch so that the RowList, when serialized, doesn't blow up 1MB
		// We should batch in 10,000 key reads, we can parallelize those...
		chunkSize := 5000
		chunks := int(math.Ceil(float64(len(keys)) / float64(chunkSize)))

		zlog.Debug("reading index rows chunks", zap.Int("chunk_count", chunks))
		for i := 0; i < chunks; i++ {
			chunkStart := i * chunkSize
			chunkEnd := (i + 1) * chunkSize
			max := len(keys)
			if max < chunkEnd {
				chunkEnd = max
			}

			keysChunk := keys[chunkStart:chunkEnd]

			zlog.Debug("reading index rows chunk", zap.Int("key_count", len(keysChunk)))
			keyRead := false
			err := fdb.store.FetchTabletRows(ctx, keysChunk, func(rowKey string, value []byte) error {
				if len(value) == 0 {
					return fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in an index, row %s", rowKey)
				}

				_, rowBlockNum, primaryKey, err := explodeWritableRowKey(rowKey)
				if err != nil {
					return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
				}

				err = rowUpdated(rowBlockNum, primaryKey, value)
				if err != nil {
					return fmt.Errorf("rowUpdated callback failed for row %q (indexed rows): %w", rowKey, err)
				}

				keyRead = true
				return nil
			})

			if err != nil {
				return fmt.Errorf("reading keys chunks: %w", err)
			}

			if !keyRead {
				return fmt.Errorf("reading a indexed key yielded no row: %s", keysChunk)
			}
		}

		zlog.Debug("finished reconciling index")
	}

	// check for latest index based on r.BlockNum
	// go through keys from last index's `AtBlockNum`, through to `BlockNum`
	// fetch all the keys within the index
	// parse all rows following the index, and keep the latest, so simply override with incoming rows..

	zlog.Debug("reading rows range from database", zap.String("first_row_key", firstRowKey), zap.String("last_row_key", lastRowKey))

	deletedCount := 0
	updatedCount := 0

	err = fdb.store.ScanTabletRows(ctx, firstRowKey, lastRowKey, func(rowKey string, value []byte) error {
		_, rowBlockNum, primaryKey, err := explodeWritableRowKey(rowKey)
		if err != nil {
			return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
		}

		if len(value) == 0 {
			err := rowDeleted(rowBlockNum, primaryKey)
			if err != nil {
				return fmt.Errorf("rowDeleted callback failed for row %q (live rows): %w", rowKey, err)
			}

			deletedCount++
			return nil
		}

		err = rowUpdated(rowBlockNum, primaryKey, value)
		if err != nil {
			return fmt.Errorf("rowUpdated callback failed for row %q (live rows): %w", rowKey, err)
		}

		updatedCount++
		return nil
	})

	if err != nil {
		return err
	}

	zlog.Debug("finished reading rows from database", zap.Int("deleted_count", deletedCount), zap.Int("updated_count", updatedCount))
	return nil
}

func (fdb *FluxDB) Read2(
	ctx context.Context,
	blockNum uint32,
	tablet Tablet,
	speculativeWrites []*WriteRequest,
) ([]Row, error) {
	ctx, span := dtracing.StartSpan(ctx, "read table", "tablet", tablet, "block_num", blockNum)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading rows from database", zap.Stringer("tablet", tablet), zap.Uint32("block_num", blockNum))

	firstRowKey := tablet.RowKeyPrefix(0)
	lastRowKey := tablet.RowKeyPrefix(blockNum + 1)
	rowByPrimaryKey := map[string]Row{}

	var idx *TableIndex
	if _, isIndexableTablet := tablet.(IndexableTablet); isIndexableTablet {
		var err error
		idx, err = fdb.getIndex2(ctx, blockNum, tablet)
		if err != nil {
			return nil, err
		}
	}

	if idx != nil {
		zlog.Debug("index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		firstRowKey = tablet.RowKeyPrefix(idx.AtBlockNum + 1)

		// Let's pre-allocated `rowByPrimaryKey` and `keys`, `rows` is likely to need at least as much rows as in the index itself
		rowByPrimaryKey = make(map[string]Row, len(idx.Map))
		keys := make([]string, len(idx.Map))

		i := 0
		for primaryKey, blockNum := range idx.Map {
			keys[i] = string(tablet.RowKey(blockNum, PrimaryKey(primaryKey)))
			i++
		}

		// Fetch all rows in the index.. could be millions
		// We need to batch so that the RowList, when serialized, doesn't blow up 1MB
		// We should batch in 10,000 key reads, we can parallelize those...
		chunkSize := 5000
		chunks := int(math.Ceil(float64(len(keys)) / float64(chunkSize)))

		zlog.Debug("reading index rows chunks", zap.Int("chunk_count", chunks))
		for i := 0; i < chunks; i++ {
			chunkStart := i * chunkSize
			chunkEnd := (i + 1) * chunkSize
			max := len(keys)
			if max < chunkEnd {
				chunkEnd = max
			}

			keysChunk := keys[chunkStart:chunkEnd]

			zlog.Debug("reading index rows chunk", zap.Int("key_count", len(keysChunk)))

			keyRead := false
			err := fdb.store.FetchTabletRows(ctx, keysChunk, func(rowKey string, value []byte) error {
				if len(value) == 0 {
					return fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in an index, row %s", rowKey)
				}

				fluxRow, err := tablet.ReadRow(rowKey, value)
				if err != nil {
					return fmt.Errorf("failed to create indexed row %s: %w", rowKey, err)
				}

				rowByPrimaryKey[string(fluxRow.PrimaryKey())] = fluxRow

				keyRead = true
				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("reading keys for index chunk %d: %w", i, err)
			}

			if !keyRead {
				return nil, fmt.Errorf("reading a indexed key yielded no row: %s", keysChunk)
			}
		}

		zlog.Debug("finished reconciling index")
	}

	// check for latest index based on r.BlockNum
	// go through keys from last index's `AtBlockNum`, through to `BlockNum`
	// fetch all the keys within the index
	// parse all rows following the index, and keep the latest, so simply override with incoming rows..

	zlog.Debug("reading rows range from database", zap.String("first_row_key", firstRowKey), zap.String("last_row_key", lastRowKey))

	deletedCount := 0
	updatedCount := 0

	err := fdb.store.ScanTabletRows(ctx, firstRowKey, lastRowKey, func(rowKey string, value []byte) error {
		fluxRow, err := tablet.ReadRow(rowKey, value)
		if err != nil {
			return fmt.Errorf("failed to create row %s: %w", rowKey, err)
		}

		if isDeletionFluxRow(fluxRow) {
			deletedCount++
			delete(rowByPrimaryKey, string(fluxRow.PrimaryKey()))

			return nil
		}

		updatedCount++
		rowByPrimaryKey[string(fluxRow.PrimaryKey())] = fluxRow

		return nil
	})

	if err != nil {
		return nil, err
	}

	zlog.Debug("read rows handling speculative writes", zap.Int("write_count", len(speculativeWrites)))
	for _, writeRequest := range speculativeWrites {
		for _, row := range writeRequest.FluxRows {
			if row.Tablet() != tablet {
				continue
			}

			if isDeletionFluxRow(row) {
				delete(rowByPrimaryKey, string(row.PrimaryKey()))
			} else {
				rowByPrimaryKey[string(row.PrimaryKey())] = row
			}
		}
	}

	zlog.Debug("post-processing read rows", zap.Int("row_count", len(rowByPrimaryKey)))

	i := 0
	rows := make([]Row, len(rowByPrimaryKey))
	for _, row := range rowByPrimaryKey {
		rows[i] = row
	}

	zlog.Debug("sorting rows")
	sort.Slice(rows, func(i, j int) bool { return string(rows[i].PrimaryKey()) < string(rows[j].PrimaryKey()) })

	zlog.Info("finished reading rows from database", zap.Int("deleted_count", deletedCount), zap.Int("updated_count", updatedCount))
	return rows, nil
}

func (fdb *FluxDB) readSingle(
	ctx context.Context,
	tableKey string,
	primaryKey string,
	blockNum uint32,
	rowUpdated func(blockNum uint32, primaryKey string, value []byte) error,
	rowDeleted func(blockNum uint32, primaryKey string) error,
) error {
	ctx, span := dtracing.StartSpan(ctx, "read single", "table_key", tableKey, "primary_key", primaryKey, "block_num", blockNum)
	defer span.End()

	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("reading single key from database", zap.String("table_key", tableKey), zap.String("primary_key", primaryKey), zap.Uint32("block_num", blockNum))

	idx, err := fdb.getIndex(ctx, tableKey, blockNum)
	if err != nil {
		return err
	}

	firstRowKey := tableKey + ":00000000"
	lastRowKey := tableKey + ":" + HexBlockNum(blockNum+1)

	if idx != nil && idx.Map[primaryKey] != 0 {
		zlog.Debug("index exists, reconciling it", zap.Int("row_count", len(idx.Map)))
		firstRowKey = tableKey + ":" + HexBlockNum(idx.AtBlockNum+1)

		if idx.Map[primaryKey] == 0 {
			zlog.Debug("index does not contain primary key, probably the key came after taking table index snaphost")
		} else {
			rowKey := fmt.Sprintf("%s:%08x:%s", tableKey, idx.Map[primaryKey], primaryKey)
			err := fdb.store.FetchTabletRow(ctx, rowKey, func(rowKey string, value []byte) error {
				if len(value) == 0 {
					return fmt.Errorf("indexes mappings should not contain empty data, empty rows don't make sense in an index, row %s", rowKey)
				}

				_, rowBlockNum, primaryKey, err := explodeWritableRowKey(rowKey)
				if err != nil {
					return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
				}

				err = rowUpdated(rowBlockNum, primaryKey, value)
				if err != nil {
					return fmt.Errorf("rowUpdated callback failed for row %q (indexed rows): %w", rowKey, err)
				}

				return nil
			})

			if err != nil {
				return err
			}
		}

		zlog.Debug("finished reconciling index")
	}

	zlog.Debug("reading rows range from database for single key retrieval", zap.String("first_row_key", firstRowKey), zap.String("last_row_key", lastRowKey))

	deletedCount := 0
	updatedCount := 0

	err = fdb.store.ScanTabletRows(ctx, firstRowKey, lastRowKey, func(rowKey string, value []byte) error {
		_, rowBlockNum, candidatePrimaryKey, err := explodeWritableRowKey(rowKey)
		if err != nil {
			return fmt.Errorf("couldn't parse row key %q: %w", rowKey, err)
		}

		if candidatePrimaryKey != primaryKey {
			return nil
		}

		if len(value) == 0 {
			err := rowDeleted(rowBlockNum, primaryKey)
			if err != nil {
				return fmt.Errorf("rowDeleted callback failed for row %q: %w", rowKey, err)
			}

			deletedCount++
			return nil
		}

		err = rowUpdated(rowBlockNum, primaryKey, value)
		if err != nil {
			return fmt.Errorf("rowUpdated callback failed for row %q: %w", rowKey, err)
		}

		updatedCount++
		return nil
	})

	if err != nil {
		return err
	}

	zlog.Info("finished reading single key from database", zap.Int("deleted_count", deletedCount), zap.Int("updated_count", updatedCount))
	return nil
}

func (fdb *FluxDB) getLastBlock(ctx context.Context) (out bstream.BlockRef, err error) {
	zlogger := logging.Logger(ctx, zlog)

	lastBlockKey := fdb.lastBlockKey()

	zlogger.Debug("fetching last writting block from storage", zap.String("row_key", lastBlockKey))
	out, err = fdb.store.FetchLastWrittenBlock(ctx, lastBlockKey)
	if err == store.ErrNotFound {
		zlogger.Info("last written block empty, returning block ID 0")
		return bstream.BlockRefFromID(strings.Repeat("00", 32)), nil
	}

	if err != nil {
		return out, err
	}

	zlogger.Debug("last written block", zap.Stringer("block", out))
	return
}

func (fdb *FluxDB) CheckCleanDBForSharding() error {
	_, err := fdb.store.FetchLastWrittenBlock(context.Background(), lastBlockRowKey)
	if err == store.ErrNotFound {
		// When there is nothing, it's what we expect, so there is no error
		return nil
	}

	if err != nil {
		return err
	}

	// At this point, the fetch return something viable, this is not correct for sharding reprocessing
	return errors.New("live injector's marker of last written block present, expected no element to exist")
}

func (fdb *FluxDB) lastBlockKey() string {
	if fdb.IsSharding() {
		return fmt.Sprintf("shard-%03d", fdb.shardIndex)
	}
	return lastBlockRowKey
}

func (fdb *FluxDB) isNextBlock(ctx context.Context, writeBlockNum uint32) error {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("checking if is next block", zap.Uint32("block_num", writeBlockNum))

	lastBlock, err := fdb.getLastBlock(ctx)
	if err != nil {
		return err
	}

	lastBlockNum := uint32(lastBlock.Num())
	if lastBlockNum != writeBlockNum-1 && lastBlockNum != 0 && lastBlockNum != 1 {
		return fmt.Errorf("block %d does not follow last block %d in db", writeBlockNum, lastBlockNum)
	}

	return nil
}

func (fdb *FluxDB) FetchLastWrittenBlock(ctx context.Context) (lastWrittenBlock bstream.BlockRef, err error) {
	zlog := logging.Logger(ctx, zlog)
	zlog.Debug("fetching last written block")

	lastWrittenBlock, err = fdb.getLastBlock(ctx)
	if err != nil {
		err = fmt.Errorf("fetching last written block: %w", err)
	}

	return
}
