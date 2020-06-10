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
	"math"
	"strconv"
	"strings"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

var collections = map[string]bool{}

// Tablet is a block-aware virtual table containing all the rows at any given
// block height for a given element. Let's assume we want to store the
// balance over a fixed set of accounts at any block height. In this case, one
// tablet would represent a single account, each actual row being the balance
// of the account at all block height.
//
// A Tablet always contain 0 to N rows, we maintain the state of each row
// independently.
type Tablet interface {
	NewRowFromKV(key string, value []byte) (TabletRow, error)

	Key() string
	KeyAt(blockNum uint32) string
	KeyForRowAt(blockNum uint32, primaryKey string) string

	IndexableTablet

	String() string
}

type IndexableTablet interface {
	PrimaryKeyByteCount() int
	EncodePrimaryKey(buffer []byte, primaryKey string) error
	DecodePrimaryKey(buffer []byte) (primaryKey string, err error)
}

func ExplodeTabletKey(key string) (collection, tablet string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	err = fmt.Errorf("tablet key should have 2 segments separated by '/' (`<collection/tablet>`), got %d segments", len(parts))
	return
}

type TabletFactory = func(row *pbfluxdb.TabletRow) Tablet

var tabletFactories = map[string]TabletFactory{}

func RegisterTabletFactory(collection string, factory TabletFactory) {
	if collections[collection] {
		panic(fmt.Errorf("collections %q is already registered, they all must be unique among registered ones", collection))
	}

	tabletFactories[collection] = factory
}

type TabletRow interface {
	Key() string
	Value() []byte

	Tablet() Tablet
	BlockNum() uint32
	PrimaryKey() string
}

func ExplodeTabletRowKey(key string) (collection, tablet, blockNum, primaryKey string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) == 4 {
		return parts[0], parts[1], parts[2], parts[3], nil
	}

	err = fmt.Errorf("row key should have 4 segments separated by '/' (`<collection/tablet/blockNum/primaryKey>`), got %d segments", len(parts))
	return
}

type BaseTabletRow struct {
	pbfluxdb.TabletRow
}

func (r *BaseTabletRow) BlockNum() uint32 {
	value, err := strconv.ParseUint(r.BlockNumKey, 16, 32)
	if err != nil {
		panic(fmt.Errorf("value %q is not a valid block num uint32 value: %w", r.BlockNumKey, err))
	}

	return uint32(value)
}

func (r *BaseTabletRow) Key() string {
	return r.Collection + "/" + r.TabletKey + "/" + r.BlockNumKey + "/" + r.PrimKey
}

func (r *BaseTabletRow) PrimaryKey() string {
	return r.PrimKey
}

func (r *BaseTabletRow) Tablet() Tablet {
	factory := tabletFactories[r.Collection]
	if factory == nil {
		panic(fmt.Errorf(`no know tablet factory for collection %s, register factories through a 'RegisterTabletFactory("prefix", func (...) { ... })' call`, r.Collection))
	}

	return factory(&r.TabletRow)
}

func (r *BaseTabletRow) Value() []byte {
	return r.Payload
}

func isDeletionRow(row TabletRow) bool {
	return len(row.Value()) == 0
}

// Siglet is a block-aware container for a single piece of information, for
// example an account's balance.
//
// A Siglet always contain a single row key but stored at any block height.
type Siglet interface {
	Key() string
	KeyAt(blockNum uint32) string

	NewEntryFromKV(entryKey string, value []byte) (SigletEntry, error)

	String() string
}

func ExplodeSigletKey(key string) (collection, siglet string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	err = fmt.Errorf("siglet key should have 2 segments separated by '/' (`<collection/siglet>`), got %d segments", len(parts))
	return
}

type SigletEntry interface {
	Key() string
	Value() []byte

	Siglet() Siglet
	BlockNum() uint32
}

func ExplodeSigletEntryKey(key string) (collection, tablet, blockNum string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], nil
	}

	err = fmt.Errorf("siglet entry key should have 3 segments separated by '/' (`<collection/siglet/blockNum>`), got %d segments", len(parts))
	return
}

type SigletFactory = func(row *pbfluxdb.TabletRow) Siglet

var sigletFactories = map[string]SigletFactory{}

func RegisterSigletFactory(collection string, factory SigletFactory) {
	if collections[collection] {
		panic(fmt.Errorf("collection %q is already registered, they all must be unique among registered ones", collection))
	}

	sigletFactories[collection] = factory
}

type BaseSigletEntry struct {
	pbfluxdb.TabletRow
}

func (r *BaseSigletEntry) BlockNum() uint32 {
	value, err := strconv.ParseUint(r.BlockNumKey, 16, 32)
	if err != nil {
		panic(fmt.Errorf("value %q is not a valid block num uint32 value: %w", r.BlockNumKey, err))
	}

	return math.MaxUint32 - uint32(value)
}

func (r *BaseSigletEntry) Key() string {
	return r.Collection + "/" + r.TabletKey + "/" + r.BlockNumKey
}

func (r *BaseSigletEntry) Siglet() Siglet {
	factory := sigletFactories[r.Collection]
	if factory == nil {
		panic(fmt.Errorf(`no know siglet factory for collection %s, register factories through a 'RegisterSigletFactory("prefix", func (...) { ... })' call`, r.Collection))
	}

	return factory(&r.TabletRow)
}

func (r *BaseSigletEntry) Value() []byte {
	return r.Payload
}

func isDeletionEntry(entry SigletEntry) bool {
	return len(entry.Value()) == 0
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
	ABI  *ContractABIEntry
	Rows []*TableRow
}

type ReadTableRowResponse struct {
	ABI *ContractABIEntry
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
	SigletEntries []SigletEntry
	TabletRows    []TabletRow

	BlockNum uint32
	BlockID  []byte
}

func (r *WriteRequest) AppendSigletEntry(entry SigletEntry) {
	r.SigletEntries = append(r.SigletEntries, entry)
}

func (r *WriteRequest) AppendTabletRow(row TabletRow) {
	r.TabletRows = append(r.TabletRows, row)
}
