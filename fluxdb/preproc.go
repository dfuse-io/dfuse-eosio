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
	"github.com/dfuse-io/derr"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

func PreprocessBlock(rawBlk *bstream.Block) (interface{}, error) {
	if rawBlk.Num()%120 == 0 {
		zlog.Info("pre-processing block (1/120)", zap.Stringer("block_id", rawBlk))
	}

	blockID, err := hex.DecodeString(rawBlk.ID())
	if err != nil {
		return nil, derr.Wrapf(err, "unable to decode block ID %q", rawBlk.ID())
	}

	blk := rawBlk.ToNative().(*pbeos.Block)

	lastDbOpForRowPath := map[string]*pbeos.DBOp{}
	firstDbOpWasInsert := map[string]bool{}
	lastKeyAccountOpForRowPath := map[string]*keyAccountOp{}
	lastTableOpForTablePath := map[string]*pbeos.TableOp{}

	req := &WriteRequest{
		BlockNum: uint32(rawBlk.Num()),
		BlockID:  blockID,
	}

	for _, trx := range blk.TransactionTraces {
		for _, dbOp := range trx.DbOps {
			if dbOp.Operation == pbeos.DBOp_OPERATION_UPDATE && bytes.Equal(dbOp.OldData, dbOp.NewData) && dbOp.OldPayer == dbOp.NewPayer {
				continue
			}

			path := tableDataRowPath(dbOp)

			lastOp := lastDbOpForRowPath[path]
			if lastOp == nil && dbOp.Operation == pbeos.DBOp_OPERATION_INSERT {
				firstDbOpWasInsert[path] = true
			}

			if dbOp.Operation == pbeos.DBOp_OPERATION_REMOVE && firstDbOpWasInsert[path] {
				delete(firstDbOpWasInsert, path)
				delete(lastDbOpForRowPath, path)
			} else {
				lastDbOpForRowPath[path] = dbOp
			}
		}

		for _, permOp := range trx.PermOps {
			for _, keyAccountOp := range permOpToKeyAccountOps(permOp) {
				lastKeyAccountOpForRowPath[keyAccountOp.rowPath] = keyAccountOp
			}
		}

		for _, tableOp := range trx.TableOps {
			lastTableOpForTablePath[tableRowPath(tableOp)] = tableOp
		}

		for _, act := range trx.ActionTraces {
			switch act.FullName() {
			case "eosio:eosio:setabi":
				abi, err := extractABIRow(uint32(rawBlk.Num()), act.Action)
				if err != nil {
					return nil, derr.Wrap(err, "extract abi: %s")
				}

				req.ABIs = append(req.ABIs, abi)

			case "eosio:eosio:linkauth":
				linkStruct, err := extractLinkAuthLinkRow(act.Action)
				if err != nil {
					return nil, derr.Wrap(err, "extract link auth")
				}

				req.AuthLinks = append(req.AuthLinks, linkStruct)

			case "eosio:eosio:unlinkauth":
				linkStruct, err := extractUnlinkAuthLinkRow(act.Action)
				if err != nil {
					return nil, derr.Wrap(err, "extract unlink auth")
				}

				req.AuthLinks = append(req.AuthLinks, linkStruct)
			}
		}
	}

	req.KeyAccounts = keyAccountOpsToWritableRows(lastKeyAccountOpForRowPath)
	req.TableScopes = tableOpsToWritableRows(lastTableOpForTablePath)

	req.TableDatas, err = dbOpsToWritableRows(lastDbOpForRowPath)
	if err != nil {
		return nil, derr.Wrap(err, "unable to convert db ops to table data row")
	}

	return req, nil
}

func permOpToKeyAccountOps(permOp *pbeos.PermOp) []*keyAccountOp {
	switch permOp.Operation {
	case pbeos.PermOp_OPERATION_INSERT:
		return permOpDataToKeyAccountOps(keyAccountOperationInsert, permOp.NewPerm)
	case pbeos.PermOp_OPERATION_UPDATE:
		var ops []*keyAccountOp

		ops = append(ops, permOpDataToKeyAccountOps(keyAccountOperationRemove, permOp.OldPerm)...)
		ops = append(ops, permOpDataToKeyAccountOps(keyAccountOperationInsert, permOp.NewPerm)...)

		return ops
	case pbeos.PermOp_OPERATION_REMOVE:
		return permOpDataToKeyAccountOps(keyAccountOperationRemove, permOp.OldPerm)
	}

	panic(fmt.Errorf("unknown perm op %s", permOp.Operation))
}

func permOpDataToKeyAccountOps(operation keyAccountOperation, perm *pbeos.PermissionObject) []*keyAccountOp {
	account := perm.Owner
	permission := perm.Name

	accountName := N(account)
	permissionName := N(permission)

	var ops []*keyAccountOp

	if perm.Authority == nil {
		return ops
	}

	for _, key := range perm.Authority.Keys {
		ops = append(ops, &keyAccountOp{
			operation:  operation,
			publicKey:  key.PublicKey,
			account:    accountName,
			permission: permissionName,
			rowPath:    key.PublicKey + ":" + account + ":" + permission,
		})
	}

	return ops
}

func dbOpsToWritableRows(latestDbOps map[string]*pbeos.DBOp) (rows []*TableDataRow, err error) {
	for _, op := range latestDbOps {
		rows = append(rows, &TableDataRow{
			Account:  N(op.Code),
			Scope:    N(op.Scope),
			Table:    N(op.TableName),
			PrimKey:  N(op.PrimaryKey),
			Payer:    N(op.NewPayer),
			Deletion: op.Operation == pbeos.DBOp_OPERATION_REMOVE,
			Data:     op.NewData,
		})
	}

	return
}

func keyAccountOpsToWritableRows(latestKeyAccountOps map[string]*keyAccountOp) (rows []*KeyAccountRow) {
	for _, op := range latestKeyAccountOps {
		rows = append(rows, &KeyAccountRow{
			PublicKey:  op.publicKey,
			Account:    op.account,
			Permission: op.permission,
			Deletion:   op.operation == keyAccountOperationRemove,
		})
	}

	return
}

func tableOpsToWritableRows(latestTableOps map[string]*pbeos.TableOp) (rows []*TableScopeRow) {
	for _, op := range latestTableOps {
		rows = append(rows, &TableScopeRow{
			Account:  N(op.Code),
			Scope:    N(op.Scope),
			Table:    N(op.TableName),
			Payer:    N(op.Payer),
			Deletion: op.Operation == pbeos.TableOp_OPERATION_REMOVE,
		})
	}

	return
}

func extractABIRow(blockNum uint32, action *pbeos.Action) (*ABIRow, error) {
	var setABI *system.SetABI
	if err := action.UnmarshalData(&setABI); err != nil {
		return nil, err
	}

	return &ABIRow{
		Account:   N(string(setABI.Account)),
		PackedABI: []byte(setABI.ABI),
		BlockNum:  blockNum,
	}, nil
}

func extractLinkAuthLinkRow(action *pbeos.Action) (*AuthLinkRow, error) {
	var linkAuth *system.LinkAuth
	if err := action.UnmarshalData(&linkAuth); err != nil {
		return nil, err
	}

	return &AuthLinkRow{
		Account:        N(string(linkAuth.Account)),
		Contract:       N(string(linkAuth.Code)),
		Action:         N(string(linkAuth.Type)),
		PermissionName: N(string(linkAuth.Requirement)),
	}, nil
}

func extractUnlinkAuthLinkRow(action *pbeos.Action) (*AuthLinkRow, error) {
	var unlinkAuth *system.UnlinkAuth
	if err := action.UnmarshalData(&unlinkAuth); err != nil {
		return nil, err
	}

	return &AuthLinkRow{
		Deletion: true,
		Account:  N(string(unlinkAuth.Account)),
		Contract: N(string(unlinkAuth.Code)),
		Action:   N(string(unlinkAuth.Type)),
	}, nil
}

func tableDataRowPath(op *pbeos.DBOp) string {
	return op.Code + "/" + op.Scope + "/" + op.TableName + "/" + op.PrimaryKey
}

func tableRowPath(op *pbeos.TableOp) string {
	return op.Code + "/" + op.Scope + "/" + op.TableName
}

// Represents a smaller transformation of a `pbeos.PermOp` to an operation
// that added or deleted an account/permission pair for a given public key.
//
// This is done because a single `pbeos.PermOp` can results into multiple
// account/permission pair being added or removed for a given public key.
type keyAccountOperation int

const (
	keyAccountOperationInsert keyAccountOperation = iota
	keyAccountOperationRemove
)

type keyAccountOp struct {
	operation  keyAccountOperation
	publicKey  string
	account    uint64
	permission uint64

	rowPath string
}
