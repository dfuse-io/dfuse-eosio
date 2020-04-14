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

package bigt

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	eosgo "github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"google.golang.org/api/option"
)

type EOSDatabase struct {
	*basebigt.Bigtable

	// Required only when writing
	writerChainID eosgo.SHA256Bytes

	Accounts     *AccountsTable
	Transactions *TransactionsTable
	Blocks       *BlocksTable
	BlocksLast   *BlocksTable // Flushed last, w/ "written" marker for write completion.
	Timeline     *TimelineTable

	maxDurationBeforeFlush time.Duration
	maxBlocksBeforeFlush   uint64
	lastFlushTime          time.Time
	blocksSinceFlush       uint64
}

func init() {
	eosdb.Register("bigtable", New)
}

func New(dsnString string, opts ...eosdb.Option) (eosdb.Driver, error) {
	dsn, err := basebigt.ParseDSN(dsnString)
	if err != nil {
		return nil, err
	}

	// var btOpts []option.ClientOption
	// for _, opt := range opts {
	// 	switch btOpt := opt.(type) {
	// 	case eosdb.GRPCConn:
	// 		btOpts = append(btOpts, option.WithGRPCConn(btOpt.ClientConn))
	// 	}
	// }

	return NewDriver(dsn.TablePrefix, dsn.Project, dsn.Instance, dsn.CreateTables, dsn.MaxDurationBeforeFlush, dsn.MaxBlocksBeforeFlush)
}

func NewDriver(tablePrefix, project, instance string, createTables bool, maxDurationBeforeFlush time.Duration, maxBlocksBeforeFlush uint64, opts ...option.ClientOption) (*EOSDatabase, error) {
	ctx := context.Background()
	client, err := bigtable.NewClient(ctx, project, instance, opts...)
	if err != nil {
		return nil, err
	}

	bt := NewWithClient(tablePrefix, client)
	if createTables {
		bt.CreateTables(ctx, project, instance, opts...)
	}
	bt.maxDurationBeforeFlush = maxDurationBeforeFlush
	bt.maxBlocksBeforeFlush = maxBlocksBeforeFlush

	return bt, nil
}

func NewWithClient(tablePrefix string, client *bigtable.Client) *EOSDatabase {
	blocksTable := NewBlocksTable(fmt.Sprintf("eos-%s-blocks", tablePrefix), client)
	blocksLastTable := NewBlocksTable(fmt.Sprintf("eos-%s-blocks", tablePrefix), client)
	accountsTable := NewAccountsTable(fmt.Sprintf("eos-%s-accounts", tablePrefix), client)
	transactionsTable := NewTransactionsTable(fmt.Sprintf("eos-%s-trxs", tablePrefix), client)
	timelineTable := NewTimelineTable(fmt.Sprintf("eos-%s-timeline", tablePrefix), client)

	bt := &EOSDatabase{
		Bigtable: basebigt.NewWithClient(tablePrefix, []*basebigt.BaseTable{
			accountsTable.BaseTable,
			transactionsTable.BaseTable,
			timelineTable.BaseTable,
			blocksTable.BaseTable,
			blocksLastTable.BaseTable,
		}, client),

		Blocks:        blocksTable,
		BlocksLast:    blocksLastTable,
		Accounts:      accountsTable,
		Transactions:  transactionsTable,
		Timeline:      timelineTable,
		lastFlushTime: time.Now(),
	}

	return bt
}

func (b *EOSDatabase) SetWriterChainID(chainID []byte) {
	b.writerChainID = eosgo.SHA256Bytes(chainID)
}

func (b *EOSDatabase) StartSpan(ctx context.Context, name string, attributes ...trace.Attribute) (context.Context, *trace.Span) {
	return b.Bigtable.StartSpan(ctx, "eos", name, attributes...)
}
