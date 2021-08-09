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
	"strings"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/logging"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/streamingfast/dgraphql"
	"github.com/streamingfast/dgraphql/analytics"
	commonTypes "github.com/streamingfast/dgraphql/types"
	"github.com/streamingfast/dmetering"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BlockIDAtAccountCreationArgs struct {
	Account string
}

func (r *Root) QueryBlockIDAtAccountCreation(ctx context.Context, args BlockIDAtAccountCreationArgs) (*BlockIDResponse, error) {
	if err := r.RateLimit(ctx, "blockmeta"); err != nil {
		return nil, err
	}

	zlogger := logging.Logger(ctx, zlog)
	acctResp, err := r.accountsReader.GetAccount(ctx, args.Account)

	if err != nil {
		if err != kvdb.ErrNotFound {
			zlogger.Warn("call to dbReader failed", zap.Error(err))
		} else {
			//////////////////////////////////////////////////////////////////////
			// Billable event on GraphQL Query - One Request, Many Oubound Documents
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "dgraphql",
				Kind:           "GraphQL Query",
				Method:         "BlockIDAtAccountCreation",
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////
		}
		return nil, dgraphql.UnwrapError(ctx, derr.Wrap(err, "failed to retrieve block ID for requested account"))
	}

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "QueryBlockIDAtAccountCreation", "BlockIDAtAccountCreationArgs", args)
	/////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, Many Oubound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "BlockIDAtAccountCreation",
		RequestsCount:  1,
		ResponsesCount: 1,
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	time, err := ptypes.Timestamp(acctResp.BlockTime)
	if err != nil {
		return nil, err
	}

	return &BlockIDResponse{
		blockID:   acctResp.BlockId,
		blockTime: time,
	}, nil
}

type BlockIDByTimeArgs struct {
	Time       graphql.Time
	Comparator string
}

func (r *Root) QueryBlockIDByTime(ctx context.Context, args BlockIDByTimeArgs) (*BlockIDResponse, error) {
	if err := r.RateLimit(ctx, "blockmeta"); err != nil {
		return nil, err
	}
	zlogger := logging.Logger(ctx, zlog)

	t := args.Time.Time

	var btResp *pbblockmeta.BlockResponse
	var err error
	switch strings.ToLower(args.Comparator) {
	case "eq":
		btResp, err = r.blockmetaClient.BlockAt(ctx, t)
	case "gte":
		btResp, err = r.blockmetaClient.BlockAfter(ctx, t, true)
	case "gt":
		btResp, err = r.blockmetaClient.BlockAfter(ctx, t, false)
	case "lte":
		btResp, err = r.blockmetaClient.BlockBefore(ctx, t, true)
	case "lt":
		btResp, err = r.blockmetaClient.BlockBefore(ctx, t, false)
	default:
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Oubound Documents
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "BlockIDByTime",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
		return nil, dgraphql.Status(ctx, codes.InvalidArgument, "'comparator' should be one of 'GT', 'GTE', 'LT', 'LTE' or 'EQ'")
	}
	if err != nil {
		if sErr, ok := status.FromError(err); ok && sErr.Code() != codes.NotFound {
			zlogger.Warn("call to blockmeta failed", zap.Error(err))
		}
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Oubound Documents
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "BlockIDByTime",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
		return nil, dgraphql.UnwrapError(ctx, derr.Wrap(err, "failed to retrieve block ID for requested time"))
	}

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "QueryBlockIDByTime", "BlockIDByTimeArgs", args)
	/////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, Many Oubound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "BlockIDByTime",
		RequestsCount:  1,
		ResponsesCount: 1,
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return &BlockIDResponse{
		blockID:   btResp.Id,
		blockTime: pbblockmeta.Timestamp(btResp.Time),
	}, nil
}

type BlockIDResponse struct {
	blockID   string
	blockTime time.Time
}

func (r *BlockIDResponse) ID() string {
	return r.blockID
}

func (r *BlockIDResponse) Num() commonTypes.Uint32 {
	return commonTypes.Uint32(eos.BlockNum(r.blockID))
}

func (r *BlockIDResponse) Time() graphql.Time {
	return graphql.Time{Time: r.blockTime}
}
