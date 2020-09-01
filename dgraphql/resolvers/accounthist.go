package resolvers

import (
	"context"
	"fmt"
	"io"

	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/dgraphql"
	"github.com/dfuse-io/dgraphql/analytics"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/opaque"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type GetAccountHistoryActionsArgs struct {
	Account string
	Limit   types.Int64
	Cursor  *string
}

type AccountHistoryActionsConnection struct {
	Edges    []*SimpleActionTraceEdge
	PageInfo PageInfo
}

type SimpleActionTraceEdge struct {
	Cursor string
	Node   *ActionTrace
}

func (r *Root) QueryGetAccountHistoryActions(ctx context.Context, args GetAccountHistoryActionsArgs) (*AccountHistoryActionsConnection, error) {
	// TODO: is that correct?
	if err := r.RateLimit(ctx, "accounthist"); err != nil {
		return nil, err
	}

	accountUint, err := eos.StringToName(args.Account)
	if err != nil {
		return nil, err
	}

	var cursor *pbaccounthist.Cursor
	if args.Cursor != nil {
		cursor = &pbaccounthist.Cursor{}
		rawCursor, err := opaque.FromOpaque(*args.Cursor)
		if err != nil {
			return nil, fmt.Errorf("unpacking cursor: %w", err)
		}
		err = proto.Unmarshal([]byte(rawCursor), cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid or malformed cursor: %w", err)
		}

		if cursor.Magic != 4374 {
			return nil, fmt.Errorf("invalid magic number in cursor, is this a cursor obtained through this same GraphQL Query?")
		}
	}

	res, err := r.getAccountHist(ctx, accountUint, int64(args.Limit), cursor)
	if err != nil {
		return nil, fmt.Errorf("running query: %w", err)
	}

	/////////////////////////////////////////////////////////////////////////
	// DO NOT change this without updating BigQuery analytics
	analytics.TrackUserEvent(ctx, "dgraphql", "GetAccountHistoryActions", "Args", args, "Edges", len(res.Edges))
	/////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, Many Oubound Documents
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "GetAccountHistoryActions",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(res.Edges)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return res, nil
}

func (r *Root) getAccountHist(ctx context.Context, account uint64, limit int64, cursor *pbaccounthist.Cursor) (*AccountHistoryActionsConnection, error) {
	zlogger := logging.Logger(ctx, zlog)

	// TODO: add a log line with what's going on here..

	stream, err := r.accountHistClient.GetActions(ctx, &pbaccounthist.GetActionsRequest{
		Account: account,
		Limit:   uint32(limit + 1),
		Cursor:  cursor,
	})
	if err != nil {
		zlogger.Error("unable to get acount history", zap.Error(err))
		return nil, dgraphql.Errorf(ctx, "accounthist backend error")
	}

	out := &AccountHistoryActionsConnection{
		Edges: []*SimpleActionTraceEdge{},
		PageInfo: PageInfo{
			HasPreviousPage: cursor != nil,
		},
	}
	for {
		if ctx.Err() != nil {
			break
		}

		match, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if len(out.Edges) >= int(limit) {
			out.PageInfo.HasNextPage = true
			continue
		}

		// TODO: if we reached the limit from the backing service, then mark `HasNextPage`

		if err != nil {
			zlogger.Info("error receiving message from search stream client", zap.Error(err))
			return nil, dgraphql.UnwrapError(ctx, err)
		}

		rawCursor, err := proto.Marshal(match.Cursor)
		if err != nil {
			return nil, err
		}
		stringCursor, _ := opaque.ToOpaque(string(rawCursor))

		if out.PageInfo.StartCursor == "" {
			out.PageInfo.StartCursor = stringCursor
		}
		out.PageInfo.EndCursor = stringCursor

		newEdge := &SimpleActionTraceEdge{
			Cursor: stringCursor,
			Node:   newActionTrace(match.ActionTrace, nil, r.abiCodecClient),
		}

		out.Edges = append(out.Edges, newEdge)
	}

	return out, nil
}
