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
	lastKeyAccountForRowKey := map[string]*KeyAccountRow{}
	// lastTableOpForTablePath := map[string]*pbcodec.TableOp{}

	req := &WriteRequest{
		BlockNum: uint32(rawBlk.Num()),
		BlockID:  blockID,
	}

	for _, trx := range blk.TransactionTraces {
		for _, dbOp := range trx.DbOps {
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
			for _, row := range permOpToKeyAccountRows(req.BlockNum, permOp) {
				lastKeyAccountForRowKey[row.Key()] = row
			}
		}

		// for _, tableOp := range trx.TableOps {
		// 	lastTableOpForTablePath[tableRowPath(tableOp)] = tableOp
		// }

		for _, act := range trx.ActionTraces {
			switch act.FullName() {
			case "eosio:eosio:setabi":
				abiEntry, err := NewContractABIEntry(req.BlockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract abi entry: %w", err)
				}

				req.AppendSigletEntry(abiEntry)

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

	// req.TableScopes = tableOpsToWritableRows(lastTableOpForTablePath)

	addDBOpsToWriteRequest(req, lastDbOpForRowPath)
	addKeyAccountOpsToWriteRequest(req, lastKeyAccountForRowKey)

	return req, nil
}

func permOpToKeyAccountRows(blockNum uint32, permOp *pbcodec.PermOp) []*KeyAccountRow {
	switch permOp.Operation {
	case pbcodec.PermOp_OPERATION_INSERT:
		return permOpDataToKeyAccountOps(blockNum, permOp.NewPerm, false)
	case pbcodec.PermOp_OPERATION_UPDATE:
		var ops []*KeyAccountRow

		ops = append(ops, permOpDataToKeyAccountOps(blockNum, permOp.OldPerm, true)...)
		ops = append(ops, permOpDataToKeyAccountOps(blockNum, permOp.NewPerm, false)...)

		return ops
	case pbcodec.PermOp_OPERATION_REMOVE:
		return permOpDataToKeyAccountOps(blockNum, permOp.OldPerm, true)
	}

	panic(fmt.Errorf("unknown perm op %s", permOp.Operation))
}

func permOpDataToKeyAccountOps(blockNum uint32, perm *pbcodec.PermissionObject, isDeletion bool) []*KeyAccountRow {
	if perm.Authority == nil || len(perm.Authority.Keys) == 0 {
		return nil
	}

	rows := make([]*KeyAccountRow, len(perm.Authority.Keys))
	for i, key := range perm.Authority.Keys {
		rows[i] = NewKeyAccountRow(blockNum, key.PublicKey, perm.Owner, perm.Name, isDeletion)
	}

	return rows
}

func addDBOpsToWriteRequest(request *WriteRequest, latestDbOps map[string]*pbcodec.DBOp) {
	blockNum := request.BlockNum
	for _, op := range latestDbOps {
		request.AppendTabletRow(NewContractStateRow(blockNum, op))
	}
}

func addKeyAccountOpsToWriteRequest(request *WriteRequest, lastKeyAccountForRowKey map[string]*KeyAccountRow) {
	for _, row := range lastKeyAccountForRowKey {
		request.AppendTabletRow(row)
	}
}

func tableOpsToWritableRows(latestTableOps map[string]*pbcodec.TableOp) (rows []*TableScopeRow) {
	for _, op := range latestTableOps {
		rows = append(rows, &TableScopeRow{
			Account:  N(op.Code),
			Scope:    N(op.Scope),
			Table:    N(op.TableName),
			Payer:    N(op.Payer),
			Deletion: op.Operation == pbcodec.TableOp_OPERATION_REMOVE,
		})
	}

	return
}

func tableDataRowPath(op *pbcodec.DBOp) string {
	return op.Code + "/" + op.Scope + "/" + op.TableName + "/" + op.PrimaryKey
}

func tableRowPath(op *pbcodec.TableOp) string {
	return op.Code + "/" + op.Scope + "/" + op.TableName
}
