// Copyright 2019 dfuse Platform Inc.
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

package trxdb

import (
	"context"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

type TestTransactionsReader struct {
	content map[string][]*pbcodec.TransactionEvent
}

func NewTestTransactionsReader(content map[string][]*pbcodec.TransactionEvent) *TestTransactionsReader {
	return &TestTransactionsReader{content: content}
}

func (r *TestTransactionsReader) GetTransactionTraces(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	return r.content[idPrefix], nil
}

func (r *TestTransactionsReader) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbcodec.TransactionEvent, err error) {
	for _, prefix := range idPrefixes {
		out = append(out, r.content[prefix])
	}
	return
}

func (r *TestTransactionsReader) GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	panic("not implemented")
}

func (r *TestTransactionsReader) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error) {
	panic("not implemented")
}

type testDriver struct {
	dsn           string
	options       []Option
	logger        *zap.Logger
	ttl           uint64
	purgeInterval uint64
}

func (db *testDriver) Close() error { return nil }

func (db *testDriver) SetLogger(logger *zap.Logger) error {
	db.logger = logger
	return nil
}

func (db *testDriver) SetPurgeableStore(ttl, purgeInterval uint64) error {
	db.ttl = ttl
	db.purgeInterval = purgeInterval
	return nil
}

func (db *testDriver) GetAccount(ctx context.Context, accountName string) (*pbcodec.AccountCreationRef, error) {
	panic("test driver, not callable")
}

func (db *testDriver) ListAccountNames(ctx context.Context, concurrentReadCount uint32) ([]string, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetTransactionTraces(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error) {
	panic("test driver, not callable")
}

func (db *testDriver) BlockIDAt(ctx context.Context, start time.Time) (id string, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetLastWrittenBlockID(ctx context.Context) (blockID string, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetBlock(ctx context.Context, id string) (*pbcodec.BlockWithRefs, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetBlockByNum(ctx context.Context, num uint32) ([]*pbcodec.BlockWithRefs, error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (ref bstream.BlockRef, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (ref bstream.BlockRef, err error) {
	panic("test driver, not callable")
}

func (db *testDriver) ListBlocks(ctx context.Context, highBlockNum uint32, limit int) ([]*pbcodec.BlockWithRefs, error) {
	panic("test driver, not callable")
}

func (db *testDriver) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbcodec.BlockWithRefs, error) {
	panic("test driver, not callable")
}

func (db *testDriver) SetWriterChainID(chainID []byte) { panic("test driver, not callable") }

func (db *testDriver) GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error) {
	panic("test driver, not callable")
}
func (db *testDriver) PutBlock(ctx context.Context, blk *pbcodec.Block) error {
	panic("test driver, not callable")
}

func (db *testDriver) UpdateNowIrreversibleBlock(ctx context.Context, blk *pbcodec.Block) error {
	panic("test driver, not callable")
}

func (db *testDriver) Flush(ctx context.Context) error { panic("test driver, not callable") }
