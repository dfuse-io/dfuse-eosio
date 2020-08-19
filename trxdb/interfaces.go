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
)

type ChainDiscriminator func(blockID string) bool

type DB interface {
	DBReader
	DBWriter

	Close() error
}

type DBReader interface {
	BlocksReader
	TransactionsReader
	AccountsReader
	TimelineExplorer
}

// This is the main interface, needed by most subsystems.
type BlocksTransactionsReader interface {
	BlocksReader
	TransactionsReader
}

type AccountsReader interface {
	GetAccount(ctx context.Context, accountName string) (*pbcodec.AccountCreationRef, error)
	//TODO: concurrentReadCount is that a property only for Bigtable? should it be configured when creating the driver
	ListAccountNames(ctx context.Context, concurrentReadCount uint32) ([]string, error)
}

type TransactionsReader interface {
	GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error)
	// It's not the job of the Storage layer to discriminate events, just get the data
	// and the caller will discriminate the right block IDs from the wrong.

	// GetTransactionTraces retrieves only the execution traces event, ignoring deferred lifecycle events.
	// It can return a nil list of TransactionEvent with no error, if nothing was found.
	GetTransactionTraces(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error)
	// GetTransactionTracesBatch returns only the execution traces (ignoring deferred licycle events), for each id prefix specified.
	// If some ids are not found, the corresponding index will have a nil list of TransactionEvent.
	GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error)

	// GetTransactionEvents retrieves all the events related to the lifecycle of a transaction, including transaction introduction, deferred creations, cancellations, and traces of execution.
	// It can return a nil list of TransactionEvent with no error, if nothing was found.
	GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error)
	// GetTransactionEventsBatch returns a list of all events for each transaction id prefix.
	// If some ids are not found, the corresponding index will have a nil list of TransactionEvent.
	GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error)
}

type TimelineExplorer interface {
	BlockIDAt(ctx context.Context, start time.Time) (id string, err error)
	BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error)
	BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error)
}

type BlocksReader interface {
	GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error)
	GetLastWrittenBlockID(ctx context.Context) (blockID string, err error)
	GetBlock(ctx context.Context, id string) (*pbcodec.BlockWithRefs, error)
	GetBlockByNum(ctx context.Context, num uint32) ([]*pbcodec.BlockWithRefs, error)
	GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (ref bstream.BlockRef, err error)
	GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (ref bstream.BlockRef, err error)

	// ListBlocks retrieves blocks where `highBlockNum` is the highest
	// returned, and will retrieve a maximum of `limit` rows.  For
	// example, if you pass `highBlockNum = math.MaxUint32` with
	// `limit = 1`, it will retrieve the last written block.
	//
	// FIXME: this one should be `lowBlockNum` and `highBlockNum`, the
	// thing is you might not have the expected block range if there
	// are forked blocks provided.
	ListBlocks(ctx context.Context, highBlockNum uint32, limit int) ([]*pbcodec.BlockWithRefs, error)
	ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbcodec.BlockWithRefs, error)
}

type DBWriter interface {
	// this is used to bootstrap the trxdb-loader pipeline
	GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error)

	SetWriterChainID(chainID []byte)
	PutBlock(ctx context.Context, blk *pbcodec.Block) error
	UpdateNowIrreversibleBlock(ctx context.Context, blk *pbcodec.Block) error
	// Flush MUST be called or you WILL lose data
	Flush(context.Context) error
}

type Debugeable interface {
	Dump()
}
