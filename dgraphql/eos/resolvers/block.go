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

package resolvers

import (
	"context"
	"encoding/hex"
	"strings"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/dgraphql"
	"github.com/dfuse-io/dgraphql/analytics"
	commonTypes "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/logging"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbgraphql "github.com/dfuse-io/pbgo/dfuse/graphql/v1"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"
)

type QueryBlockRequest struct {
	Num *commonTypes.Uint32
	Id  *string
}

func (r *Root) QueryBlock(ctx context.Context, req QueryBlockRequest) (*Block, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("querying block by num/id", zap.Reflect("request", req))

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, One Outbound Documents
	// Additional outbound docs are counted in TransactionTrace
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	// TODO: How could we include the total outbound docs count since it's composed?
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "Block",
		RequestsCount:  1,
		ResponsesCount: 1,
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "QueryBlock", "QueryBlockRequest", req)
	/////////////////////////////////////////////////////////////////////////

	if (req.Id != nil) && (req.Num != nil) {
		return nil, dgraphql.Errorf(ctx, "Invalid request, can only specify block num or ID")
	}

	var block *pbeos.BlockWithRefs
	var err error
	if req.Id != nil {
		blockID := strings.TrimPrefix(*req.Id, "0x")
		if !validateBlockId(blockID) {
			return nil, dgraphql.Errorf(ctx, "Invalid 'id' field %q", *req.Id)
		}

		block, err = r.blocksReader.GetBlock(ctx, blockID)
		if err != nil {
			if err == kvdb.ErrNotFound {
				zlogger.Error("failed to get block", zap.Error(err))
				return nil, dgraphql.Errorf(ctx, "block %q not found", *req.Id)
			}
			return nil, dgraphql.Errorf(ctx, "failed to get block by id: %s", err)
		}
	} else if req.Num != nil {
		blocks, err := r.blocksReader.GetBlockByNum(ctx, uint32(*req.Num))
		if err != nil {
			if err == kvdb.ErrNotFound {
				return nil, dgraphql.Errorf(ctx, "block %d not found", *req.Num)
			}
			zlogger.Error("failed to get block", zap.Error(err))
			return nil, dgraphql.Errorf(ctx, "failed to get block by num: %s", err)
		}
		for _, b := range blocks {
			if b.Irreversible {
				block = b
			}
		}
		if block == nil {
			zlog.Debug("no irreversible block found, checking blockmeta for longest chain block", zap.Uint32("block_num", uint32(*req.Num)), zap.Int("block_count", len(blocks)))
			resp, err := r.blockmetaClient.ChainDiscriminatorClient().GetBlockInLongestChain(ctx, &pbblockmeta.GetBlockInLongestChainRequest{BlockNum: uint64(*req.Num)})
			if err != nil {
				zlogger.Error("failed to get block in longest chain", zap.Error(err))
				return nil, dgraphql.Errorf(ctx, "failed to get block by num: %s", err)
			}
			for _, b := range blocks {
				if b.Id == resp.BlockId {
					block = b
				}
			}
			if block == nil {
				zlogger.Error("failed to find blockmeta's block in kcvd", zap.Uint32("block_num", uint32(*req.Num)), zap.String("block_id", resp.BlockId), zap.Int("block_count", len(blocks)))
				return nil, dgraphql.Errorf(ctx, "failed to get block by num: %s", err)
			}
		}
	} else {
		return nil, dgraphql.Errorf(ctx, "Invalid request, must specify either a block num or ID")

	}

	return newBlock(block, r), nil
}

//---------------------------
// Block
//----------------------------
type Block struct {
	root        *Root
	blkWithRefs *pbeos.BlockWithRefs
}

func newBlock(blkWithRefs *pbeos.BlockWithRefs, root *Root) *Block {
	return &Block{
		root:        root,
		blkWithRefs: blkWithRefs,
	}
}

func (b *Block) ID() string {
	zlog.Info("block ID")
	return b.blkWithRefs.Block.Id
}

func (b *Block) Num() commonTypes.Uint32 {
	return commonTypes.Uint32(b.blkWithRefs.Block.Number)
}

func (b *Block) DposLIBNum() commonTypes.Uint32 {
	return commonTypes.Uint32(b.blkWithRefs.Block.DposIrreversibleBlocknum)
}

func (b *Block) Irreversible() bool {
	return b.blkWithRefs.Irreversible
}

func (b *Block) Header() *BlockHeader {
	return newBlockHeader(b.blkWithRefs.Block.Id, commonTypes.Uint32(b.blkWithRefs.Block.Number), b.blkWithRefs.Block.Header)
}

func (b *Block) ExecutedTransactionCount() commonTypes.Uint32 {
	return commonTypes.Uint32(b.blkWithRefs.Block.TransactionCount)
}

type TransactionTracesReq struct {
	First  *commonTypes.Uint32
	Last   *commonTypes.Uint32
	Before *string
	After  *string
}

func (b *Block) TransactionTraces(ctx context.Context, req *TransactionTracesReq) (*TransactionTraceConnection, error) {
	zlogger := logging.Logger(ctx, zlog)

	allRefs := append(b.blkWithRefs.ImplicitTransactionRefs.Hashes, b.blkWithRefs.TransactionTraceRefs.Hashes...)

	if len(allRefs) == 0 {
		return newEmptyTransactionTraceConnection(), nil
	}

	paginator, err := dgraphql.NewPaginator(req.First, req.Last, req.Before, req.After, 100, func() proto.Message {
		return &pbgraphql.TransactionCursor{}
	})
	if err != nil {
		return nil, dgraphql.Errorf(ctx, "%s", err)
	}

	trxTraceRefs := PagineableTransactionTraceRefs(allRefs)
	trxTraceRefs = paginator.Paginate(&trxTraceRefs).(PagineableTransactionTraceRefs)

	// chainDiscriminator says `true` always.
	trxEvents, err := b.root.trxsReader.GetTransactionTracesBatch(ctx, trxTraceRefs.trxTraceIds())
	if err != nil {
		zlogger.Error("failed to get db reader get transactions", zap.Error(err))
		return nil, dgraphql.Errorf(ctx, "unable to retrieve transaction traces")
	}

	var trxLifecycles []*pbeos.TransactionLifecycle
	for _, events := range trxEvents {
		lifecycle := pbeos.MergeTransactionEvents(events, func(id string) bool {
			return id == b.blkWithRefs.Id
		})
		trxLifecycles = append(trxLifecycles, lifecycle)
	}

	edges := []*TransactionTraceEdge{}
	for _, trx := range trxLifecycles {
		edges = append(edges, &TransactionTraceEdge{
			cursor: dgraphql.MustProtoToOpaqueCursor(&pbgraphql.TransactionCursor{
				Ver:              1,
				TransactionIndex: 0,
				TransactionHash:  trx.Id,
			}, "transaction_cursor"),
			node: newTransactionTrace(trx.ExecutionTrace, b.blkWithRefs.Block.Header, nil, b.root.abiCodecClient),
		})
	}

	pageInfo := &PageInfo{}
	if len(edges) != 0 {
		pageInfo.StartCursor = edges[0].cursor
		pageInfo.EndCursor = edges[len(edges)-1].cursor
	}

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - Request already counted in block,
	// Many Outbound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	// TODO: maybe we should find a way to link this event with the one under trxTrace..
	//       then again, maybe not.
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "TransactionTraces",
		RequestsCount:  0,
		ResponsesCount: int64(len(edges)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return newTransactionTraceConnection(edges, pageInfo), nil
}

type TransactionTraceConnection struct {
	PageInfo *PageInfo
	Edges    []*TransactionTraceEdge
}

func newEmptyTransactionTraceConnection() *TransactionTraceConnection {
	return &TransactionTraceConnection{
		Edges:    []*TransactionTraceEdge{},
		PageInfo: &PageInfo{},
	}
}

func newTransactionTraceConnection(edges []*TransactionTraceEdge, pageInfo *PageInfo) *TransactionTraceConnection {
	return &TransactionTraceConnection{
		Edges:    edges,
		PageInfo: pageInfo,
	}
}

type TransactionTraceEdge struct {
	cursor string
	node   *TransactionTrace
}

func (t *TransactionTraceEdge) Cursor() string          { return t.cursor }
func (t *TransactionTraceEdge) Node() *TransactionTrace { return t.node }

//---------------------------
// Pageneable Resource
//----------------------------
type PagineableTransactionTraceRefs [][]byte

func (p PagineableTransactionTraceRefs) Length() int {
	return len(p)
}

func (p PagineableTransactionTraceRefs) IsEqual(index int, key string) bool {
	return hex.EncodeToString(p[index]) == key
}

func (p PagineableTransactionTraceRefs) trxTraceIds() (out []string) {
	for _, t := range p {
		out = append(out, hex.EncodeToString(t))
	}
	return out
}

func (p PagineableTransactionTraceRefs) Append(slice dgraphql.Pagineable, index int) dgraphql.Pagineable {
	if slice == nil {
		return dgraphql.Pagineable(PagineableTransactionTraceRefs([][]byte{p[index]}))
	} else {
		return dgraphql.Pagineable(append(slice.(PagineableTransactionTraceRefs), p[index]))
	}
}
