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

package statedb

import (
	"bytes"
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/fluxdb"
)

type BlockMapper struct {
}

func (m *BlockMapper) Map(rawBlk *bstream.Block) (*fluxdb.WriteRequest, error) {
	blk := rawBlk.ToNative().(*pbcodec.Block)

	lastSingletEntryMap := map[string]fluxdb.SingletEntry{}
	lastTabletRowMap := map[string]fluxdb.TabletRow{}

	firstDbOpWasInsert := map[string]bool{}

	req := &fluxdb.WriteRequest{
		Height:   rawBlk.Num(),
		BlockRef: rawBlk.AsRef(),
	}

	blockNum := req.BlockRef.Num()
	for _, trx := range blk.TransactionTraces() {
		for _, dbOp := range trx.DbOps {
			// There is no change in this row, not sure how it got here, discarding it anyway
			if dbOp.Operation == pbcodec.DBOp_OPERATION_UPDATE && bytes.Equal(dbOp.OldData, dbOp.NewData) && dbOp.OldPayer == dbOp.NewPayer {
				continue
			}

			row, err := dbOpToContractStateRow(blockNum, dbOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create contract state row for db op: %w", err)
			}

			rowKey := keyForRow(row)
			lastOp := lastTabletRowMap[rowKey]
			if lastOp == nil && dbOp.Operation == pbcodec.DBOp_OPERATION_INSERT {
				firstDbOpWasInsert[rowKey] = true
			}

			if dbOp.Operation == pbcodec.DBOp_OPERATION_REMOVE && firstDbOpWasInsert[rowKey] {
				delete(firstDbOpWasInsert, rowKey)
				delete(lastTabletRowMap, rowKey)
			} else {
				lastTabletRowMap[rowKey] = row
			}
		}

		for _, permOp := range trx.PermOps {
			rows, err := permOpToKeyAccountRows(blockNum, permOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create key account rows for perm op: %w", err)
			}

			for _, row := range rows {
				lastTabletRowMap[keyForRow(row)] = row
			}
		}

		for _, tableOp := range trx.TableOps {
			row, err := NewContractTableScopeRow(blockNum, tableOp)
			if err != nil {
				return nil, fmt.Errorf("unable to create contract table scope row for table op: %w", err)
			}

			lastTabletRowMap[keyForRow(row)] = row
		}

		for _, act := range trx.ActionTraces {
			switch act.FullName() {
			case "eosio:eosio:setabi":
				abiEntry, err := NewContractABIEntry(req.BlockRef.Num(), act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract abi entry: %w", err)
				}

				lastSingletEntryMap[keyForEntry(abiEntry)] = abiEntry

			case "eosio:eosio:linkauth":
				authLinkRow, err := NewInsertAuthLinkRow(blockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract link auth: %w", err)
				}

				lastTabletRowMap[keyForRow(authLinkRow)] = authLinkRow

			case "eosio:eosio:unlinkauth":
				authLinkRow, err := NewDeleteAuthLinkRow(blockNum, act)
				if err != nil {
					return nil, fmt.Errorf("unable to extract unlink auth: %w", err)
				}

				lastTabletRowMap[keyForRow(authLinkRow)] = authLinkRow
			}
		}
	}

	addSingletEntriesToRequest(req, lastSingletEntryMap)
	addTabletRowsToRequest(req, lastTabletRowMap)

	return req, nil
}

func addSingletEntriesToRequest(request *fluxdb.WriteRequest, singleEntriesMap map[string]fluxdb.SingletEntry) {
	for _, entry := range singleEntriesMap {
		request.AppendSingletEntry(entry)
	}
}

func addTabletRowsToRequest(request *fluxdb.WriteRequest, tabletRowsMap map[string]fluxdb.TabletRow) {
	for _, row := range tabletRowsMap {
		request.AppendTabletRow(row)
	}
}

func addDBOpsToWriteRequest(request *fluxdb.WriteRequest, latestDbOps map[string]*pbcodec.DBOp) error {
	blockNum := request.BlockRef.Num()
	for _, op := range latestDbOps {
		row, err := NewContractStateRow(blockNum, op)
		if err != nil {
			return fmt.Errorf("unable to create row for db op: %w", err)
		}

		request.AppendTabletRow(row)
	}

	return nil
}

func dbOpToContractStateRow(blockNum uint64, op *pbcodec.DBOp) (*ContractStateRow, error) {
	row, err := NewContractStateRow(blockNum, op)
	if err != nil {
		return nil, err
	}

	return row, nil
}

func permOpToKeyAccountRows(blockNum uint64, permOp *pbcodec.PermOp) ([]*KeyAccountRow, error) {
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

func permToKeyAccountRows(blockNum uint64, perm *pbcodec.PermissionObject, isDeletion bool) (rows []*KeyAccountRow, err error) {
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

func keyForEntry(entry fluxdb.SingletEntry) string {
	return string(fluxdb.KeyForSingletEntry(entry))
}

func keyForRow(row fluxdb.TabletRow) string {
	return string(fluxdb.KeyForTabletRow(row))
}
