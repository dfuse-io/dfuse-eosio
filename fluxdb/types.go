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
	"crypto/md5"
	"encoding/binary"
	"fmt"
)

type ReadTableRequest struct {
	Account, Scope, Table uint64
	Key                   *uint64
	BlockNum              uint32
	Offset, Limit         *uint32
	SpeculativeWrites     []*WriteRequest
}

func (r *ReadTableRequest) tableKey() string {
	return fmt.Sprintf("td:%016x:%016x:%016x", r.Account, r.Table, r.Scope)
}

type ReadTableRowRequest struct {
	ReadTableRequest
	PrimaryKey uint64
}

func (r *ReadTableRowRequest) primaryKeyString() string {
	return fmt.Sprintf("%016x", r.PrimaryKey)
}

type ReadTableResponse struct {
	ABI  *ABIRow
	Rows []*TableRow
}

type ReadTableRowResponse struct {
	ABI *ABIRow
	Row *TableRow
}

type TableRow struct {
	Key      uint64
	Payer    uint64
	Data     []byte
	BlockNum uint32
}

type LinkedPermission struct {
	Contract       string `json:"contract"`
	Action         string `json:"action"`
	PermissionName string `json:"permission_name"`
}

type WriteRequest struct {
	ABIs []*ABIRow

	AuthLinks   []*AuthLinkRow
	KeyAccounts []*KeyAccountRow
	TableDatas  []*TableDataRow
	TableScopes []*TableScopeRow

	BlockNum uint32
	BlockID  []byte
}

func (req *WriteRequest) purgeShardedRows(shardIdx, shardCount uint32) {
	include := func(row writableRow) bool {
		h := md5.New()
		_, _ = h.Write([]byte(row.tableKey()))
		md5Hash := h.Sum(nil)

		bigInt := binary.LittleEndian.Uint32(md5Hash)
		elementShard := bigInt % shardCount
		inCurrentShard := shardIdx == elementShard

		return inCurrentShard
	}

	var newAuthLinks []*AuthLinkRow
	for _, el := range req.AuthLinks {
		if include(el) {
			newAuthLinks = append(newAuthLinks, el)
		}
	}
	req.AuthLinks = newAuthLinks

	var newKeyAccounts []*KeyAccountRow
	for _, el := range req.KeyAccounts {
		if include(el) {
			newKeyAccounts = append(newKeyAccounts, el)
		}
	}
	req.KeyAccounts = newKeyAccounts

	var newTableDatas []*TableDataRow
	for _, el := range req.TableDatas {
		if include(el) {
			newTableDatas = append(newTableDatas, el)
		}
	}
	req.TableDatas = newTableDatas

	var newTableScopes []*TableScopeRow
	for _, el := range req.TableScopes {
		if include(el) {
			newTableScopes = append(newTableScopes, el)
		}
	}
	req.TableScopes = newTableScopes

	if shardIdx != 0 {
		req.ABIs = nil
	}
}

func (req *WriteRequest) appendRow(row interface{}) {
	switch obj := row.(type) {
	case *AuthLinkRow:
		req.AuthLinks = append(req.AuthLinks, obj)
	case *KeyAccountRow:
		req.KeyAccounts = append(req.KeyAccounts, obj)
	case *TableDataRow:
		req.TableDatas = append(req.TableDatas, obj)
	case *TableScopeRow:
		req.TableScopes = append(req.TableScopes, obj)
	default:
		panic(fmt.Sprintf("unsupported writable row: %T", row))
	}
}

func (req *WriteRequest) AllWritableRows() (out []writableRow) {
	for _, el := range req.AuthLinks {
		out = append(out, el)
	}

	for _, el := range req.KeyAccounts {
		out = append(out, el)
	}

	for _, el := range req.TableDatas {
		out = append(out, el)
	}

	for _, el := range req.TableScopes {
		out = append(out, el)
	}

	return
}

var emptyRowData = []byte{1}

type writableRow interface {
	tableKey() string
	rowKey(blockNum uint32) string
	primKey() string

	isDeletion() bool
	// buildData *MUST* always be a non-zero amount of bytes, to distinguish from deletion, even if it's only padding.
	buildData() []byte
}

type AuthLinkRow struct {
	Deletion bool

	Account  uint64
	Contract uint64
	Action   uint64

	PermissionName uint64
}

func (r *AuthLinkRow) tableKey() string {
	return fmt.Sprintf("al:%016x", r.Account)
}

func (r *AuthLinkRow) rowKey(blockNum uint32) string {
	return fmt.Sprintf("%s:%08x:%s", r.tableKey(), blockNum, r.primKey())
}

func (r *AuthLinkRow) primKey() string {
	return fmt.Sprintf("%016x:%016x", r.Contract, r.Action)
}

func (r *AuthLinkRow) isDeletion() bool {
	return r.Deletion
}

func (r *AuthLinkRow) buildData() []byte {
	value := make([]byte, 8)
	big.PutUint64(value, r.PermissionName)
	return value
}

type KeyAccountRow struct {
	PublicKey  string
	Account    uint64
	Permission uint64
	Deletion   bool
}

func (r *KeyAccountRow) tableKey() string {
	return "ka2:" + r.PublicKey
}

func (r *KeyAccountRow) rowKey(blockNum uint32) string {
	return fmt.Sprintf("%s:%08x:%s", r.tableKey(), blockNum, r.primKey())
}

func (r *KeyAccountRow) primKey() string {
	return fmt.Sprintf("%016x:%016x", r.Account, r.Permission)
}

func (r *KeyAccountRow) isDeletion() bool {
	return r.Deletion
}

func (r *KeyAccountRow) buildData() []byte {
	return emptyRowData
}

type TableDataRow struct {
	Account, Scope, Table, PrimKey uint64
	Payer                          uint64
	Deletion                       bool
	Data                           []byte
}

func (t *TableDataRow) tableKey() string {
	return fmt.Sprintf("td:%016x:%016x:%016x", t.Account, t.Table, t.Scope)
}

func (t *TableDataRow) rowKey(blockNum uint32) string {
	return fmt.Sprintf("%s:%08x:%s", t.tableKey(), blockNum, t.primKey())
}

func (t *TableDataRow) primKey() string {
	return fmt.Sprintf("%016x", t.PrimKey)
}

func (t *TableDataRow) isDeletion() bool {
	return t.Deletion
}

func (t *TableDataRow) buildData() []byte {
	value := make([]byte, len(t.Data)+8)
	big.PutUint64(value, t.Payer)
	copy(value[8:], t.Data)
	return value
}

type TableScopeRow struct {
	Account, Scope, Table uint64
	Deletion              bool
	Payer                 uint64
}

func (t *TableScopeRow) tableKey() string {
	return fmt.Sprintf("ts:%016x:%016x", t.Account, t.Table)
}

func (t *TableScopeRow) rowKey(blockNum uint32) string {
	return fmt.Sprintf("%s:%08x:%s", t.tableKey(), blockNum, t.primKey())
}

func (t *TableScopeRow) primKey() string {
	return fmt.Sprintf("%016x", t.Scope)
}

func (t *TableScopeRow) isDeletion() bool {
	return t.Deletion
}

func (t *TableScopeRow) buildData() []byte {
	value := make([]byte, 8)
	big.PutUint64(value, t.Payer)

	return value
}

type ABIRow struct {
	Account   uint64
	BlockNum  uint32 // in Read operation only
	PackedABI []byte
}
