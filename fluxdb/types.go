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
	"fmt"
	"strconv"
	"strings"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

/// New Architecture (Proposal)

type RowKey string
type TabletKey string
type PrimaryKey string

type Row interface {
	Key() RowKey

	Tablet() Tablet
	BlockNum() uint32
	PrimaryKey() PrimaryKey

	Data() []byte
}

func ExplodeRowKey(rowKey string) (collection, tablet, blockNum, primaryKey string, err error) {
	parts := strings.Split(rowKey, "/")
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], parts[3], nil
	}

	err = fmt.Errorf("row key should have 4 segments separated by '/' (`<collection/tablet/blockNum/primaryKey>`), got %d segments", len(parts))
	return
}

func ExplodeSingleRowKey(rowKey string) (collection, tablet, blockNum string, err error) {
	parts := strings.Split(rowKey, "/")
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], nil
	}

	err = fmt.Errorf("single row key should have 3 segments separated by '/' (`<collection/tablet/blockNum>`), got %d segments", len(parts))
	return
}

func NewTabletRow(pbRow pbfluxdb.TabletRow) TabletRow {
	return TabletRow{
		TabletRow: pbRow,
	}
}

type TabletRow struct {
	pbfluxdb.TabletRow
}

func (r *TabletRow) BlockNum() uint32 {
	value, err := strconv.ParseInt(r.BlockNumKey, 10, 32)
	if err != nil {
		panic(fmt.Errorf("value %q is not a valid block num uint32 value", r.BlockNumKey))
	}

	return uint32(value)
}

func (r *TabletRow) Key() RowKey {
	return RowKey(r.Collection + "/" + r.TabletKey + "/" + r.BlockNumKey + "/" + r.PrimKey)
}

func (r *TabletRow) PrimaryKey() PrimaryKey {
	return PrimaryKey(r.PrimKey)
}

func (r *TabletRow) Tablet() Tablet {
	factory := knownTabletFactory[r.Collection]
	if factory == nil {
		panic(fmt.Errorf(`no know tablet factory for collection %s, register factories through a 'RegisterTabletFactory("prefix", func (...) { ... })' call`, r.Collection))
	}

	return factory(&r.TabletRow)
}

func (r *TabletRow) Data() []byte {
	return r.Payload
}

func isDeletionFluxRow(row Row) bool {
	return row.Data() == nil
}

// Tablet is a block-aware virtual table containing all the rows at any given
// block height for a given table key. Let's assume we want to store the
// balance over a fixed set of accounts at any block height. In this case, one
// tablet would represent a single account, each actual row being the balance
// of the account at all block height.
type Tablet interface {
	Key() TabletKey

	RowKeyPrefix(blockNum uint32) string
	RowKey(blockNum uint32, primaryKey PrimaryKey) RowKey

	ReadRow(rowKey string, value []byte) (Row, error)

	String() string
}

type IndexableTablet interface {
	PrimaryKeyByteCount() int
	EncodePrimaryKey(buffer []byte, primaryKey string) error
	DecodePrimaryKey(buffer []byte) (primaryKey string, err error)
}

type SingleRowTablet interface {
	SingleRowOnly() bool
}

type TabletFactory = func(row *pbfluxdb.TabletRow) Tablet

var knownTabletFactory map[string]TabletFactory

func RegisterTabletFactory(collection string, factory TabletFactory) {
	if knownTabletFactory == nil {
		knownTabletFactory = map[string]TabletFactory{}
	}

	if _, exists := knownTabletFactory[collection]; exists {
		panic(fmt.Errorf("tablet prefix %q is already registered, they all must be unique among registered ones"))
	}

	knownTabletFactory[collection] = factory
}

///

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
	FluxRows []Row

	BlockNum uint32
	BlockID  []byte
}

func (r *WriteRequest) AppendFluxRow(row Row) {
	r.FluxRows = append(r.FluxRows, row)
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
