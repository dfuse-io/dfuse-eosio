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
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

func PreprocessBlock(rawBlk *bstream.Block) (interface{}, error) {
	if rawBlk.Num()%600 == 0 {
		zlog.Info("pre-processing block (printed each 600 blocks)", zap.Stringer("block", rawBlk))
	}

	blockID, err := hex.DecodeString(rawBlk.ID())
	if err != nil {
		return nil, fmt.Errorf("unable to decode block %q: %w", rawBlk, err)
	}

	blk := rawBlk.ToNative().(*pbcodec.Block)

	lastDbOpForRowPath := map[string]*pbcodec.DBOp{}
	firstDbOpWasInsert := map[string]bool{}

	lastTabletRowMap := map[string]TabletRow{}

	req := &WriteRequest{
		BlockNum: uint32(rawBlk.Num()),
		BlockID:  blockID,
	}

	for _, trx := range blk.TransactionTraces() {
		for _, dbOp := range trx.DbOps {
			// There is no change in this row, not sure how it got here, discarding it anyway
			if dbOp.Operation == pbcodec.DBOp_OPERATION_UPDATE && bytes.Equal(dbOp.OldData, dbOp.NewData) && dbOp.OldPayer == dbOp.NewPayer {
				continue
			}

			path := tableDataRowPath(dbOp)

			lastOp := lastDbOpForRowPath[path]
			if lastOp == nil && dbOp.Operation == pbcodec.DBOp_OPERATION_INSERT {
				firstDbOpWasInsert[path] = true
			}

			if dbOp.Operation == pbcodec.DBOp_OPERATION_REMOVE && firstDbOpWasInsert[path] {
				delete(firstDbOpWasInsert, path)
				delete(lastDbOpForRowPath, path)
			} else {
				lastDbOpForRowPath[path] = dbOp
			}
		}

		for _, permOp := range trx.PermOps {
			permRow, err := NewAccountPermissionsRow(req.BlockNum, permOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create contract table scope row for table op: %w", err)
			}

			lastTabletRowMap[permRow.Key()] = permRow

			rows, err := permOpToKeyAccountRows(req.BlockNum, permOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create key account rows for perm op: %w", err)
			}

			for _, row := range rows {
				lastTabletRowMap[row.Key()] = row
			}
		}

		for _, tableOp := range trx.TableOps {
			row, err := NewContractTableScopeRow(req.BlockNum, tableOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create contract table scope row for table op: %w", err)
			}

			lastTabletRowMap[row.Key()] = row
		}

		for _, act := range trx.ActionTraces {
			switch act.FullName() {
			case "eosio:eosio:newaccount":
				accountsRow, err := NewAccountsRow(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract accounts row: %w", err)
				}

				lastTabletRowMap[accountsRow.Key()] = accountsRow

			case "eosio:eosio:setcode":
				contractRow, err := NewContractRow(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract contract row: %w", err)
				}

				codeEntry, err := NewContractCodeEntry(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract contract entry: %w", err)
				}

				req.AppendSingletEntry(codeEntry)
				lastTabletRowMap[contractRow.Key()] = contractRow

			case "eosio:eosio:setabi":
				abiEntry, err := NewContractABIEntry(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract abi entry: %w", err)
				}

				req.AppendSingletEntry(abiEntry)

			case "eosio:eosio:linkauth":
				authLinkRow, err := NewInsertAuthLinkRow(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract link auth: %w", err)
				}

				req.AppendTabletRow(authLinkRow)

			case "eosio:eosio:unlinkauth":
				authLinkRow, err := NewDeleteAuthLinkRow(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract unlink auth: %w", err)
				}

				req.AppendTabletRow(authLinkRow)
			}
		}
	}

	if err := addDBOpsToWriteRequest(req, lastDbOpForRowPath); err != nil {
		return nil, fmt.Errorf("unable to add db ops to request: %w", err)
	}

	addTabletRowsToRequest(req, lastTabletRowMap)

	return req, nil
}

func addDBOpsToWriteRequest(request *WriteRequest, latestDbOps map[string]*pbcodec.DBOp) error {
	blockNum := request.BlockNum
	for _, op := range latestDbOps {
		row, err := NewContractStateRow(blockNum, op)
		if err != nil {
			return fmt.Errorf("unable to create row for db op: %w", err)
		}

		request.AppendTabletRow(row)
	}

	return nil
}

func addTabletRowsToRequest(request *WriteRequest, tabletRowsMap map[string]TabletRow) {
	for _, row := range tabletRowsMap {
		request.AppendTabletRow(row)
	}
}

func permOpToKeyAccountRows(blockNum uint32, permOp *pbcodec.PermOp) ([]*KeyAccountRow, error) {
	switch permOp.Operation {
	case pbcodec.PermOp_OPERATION_INSERT:
		return permToKeyAccountRows(blockNum, permOp.NewPerm, false)
	case pbcodec.PermOp_OPERATION_UPDATE:
		var rows []*KeyAccountRow
		deletedRows, err := permToKeyAccountRows(blockNum, permOp.OldPerm, true)
		if err != nil {
			return nil, fmt.Errorf("unable to get key accounts from old perm: %w", err)
		}

		insertedRows, err := permToKeyAccountRows(blockNum, permOp.NewPerm, false)
		if err != nil {
			return nil, fmt.Errorf("unable to get key accounts from new perm: %w", err)
		}

		rows = append(rows, deletedRows...)
		rows = append(rows, insertedRows...)

		return rows, nil
	case pbcodec.PermOp_OPERATION_REMOVE:
		return permToKeyAccountRows(blockNum, permOp.OldPerm, true)
	}

	panic(fmt.Errorf("unknown perm op %s", permOp.Operation))
}

func permToKeyAccountRows(blockNum uint32, perm *pbcodec.PermissionObject, isDeletion bool) (rows []*KeyAccountRow, err error) {
	if perm.Authority == nil || len(perm.Authority.Keys) == 0 {
		return nil, nil
	}

	rows = make([]*KeyAccountRow, len(perm.Authority.Keys))
	for i, key := range perm.Authority.Keys {
		rows[i], err = NewKeyAccountRow(blockNum, key.PublicKey, perm.Owner, perm.Name, isDeletion)
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to create key account row for permission object: %w", err)
			}
		}
	}

	return
}

func tableDataRowPath(op *pbcodec.DBOp) string {
	return op.Code + "/" + op.Scope + "/" + op.TableName + "/" + op.PrimaryKey
}
