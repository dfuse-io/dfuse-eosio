package resolvers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dfuse-io/derr"
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
	Account  string
	Contract *string
	Limit    types.Int64
	Cursor   *string
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
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("query account history actions", zap.Reflect("request", args))

	// TODO: is that correct?
	if err := r.RateLimit(ctx, "accounthist"); err != nil {
		return nil, err
	}

	if err := r.checkAccounthistServiceAvailability(zlogger, args); err != nil {
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

	timeout := 30 * time.Second
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var res *AccountHistoryActionsConnection
	if args.Contract != nil {
		contractUint, err := eos.StringToName(*args.Contract)
		if err != nil {
			return nil, err
		}
		res, err = r.getAccountHistContract(queryCtx, accountUint, contractUint, int64(args.Limit), cursor)
	} else {
		res, err = r.getAccountHist(queryCtx, accountUint, int64(args.Limit), cursor)
	}

	if err != nil {
		zlogger.Error("unable to complete query", zap.Error(err))
		if derr.Find(err, dgraphql.IsDeadlineExceededError) != nil {
			return nil, dgraphql.Errorf(ctx, "timeout of %s exceeded before completing the request", timeout)
		}

		return nil, dgraphql.Errorf(ctx, "backend error")
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

type ActionReceiver interface {
	Recv() (*pbaccounthist.ActionResponse, error)
}

func (r *Root) getAccountHist(ctx context.Context, account uint64, limit int64, cursor *pbaccounthist.Cursor) (*AccountHistoryActionsConnection, error) {
	stream, err := r.accounthistClients.Account.GetActions(ctx, &pbaccounthist.GetActionsRequest{
		Account: account,
		Limit:   uint32(limit + 1),
		Cursor:  cursor,
	})
	if err != nil {
		return nil, fmt.Errorf("accounthist stream: %w", err)
	}

	return r.handleActionStream(ctx, stream, limit, cursor)
}

func (r *Root) getAccountHistContract(ctx context.Context, account, contract uint64, limit int64, cursor *pbaccounthist.Cursor) (*AccountHistoryActionsConnection, error) {
	stream, err := r.accounthistClients.AccountContract.GetAccountContractActions(ctx, &pbaccounthist.GetTokenActionsRequest{
		Account:  account,
		Contract: contract,
		Limit:    uint32(limit + 1),
		Cursor:   cursor,
	})
	if err != nil {
		return nil, fmt.Errorf("accounthist by contract stream: %w", err)
	}

	return r.handleActionStream(ctx, stream, limit, cursor)
}

func (r *Root) checkAccounthistServiceAvailability(logger *zap.Logger, args GetAccountHistoryActionsArgs) error {
	if r.accounthistClients.AccountContract == nil {
		logger.Info("accounthistClients.AccountContract does not exists")
	} else {
		logger.Info("accounthistClients.AccountContract  exists")
	}

	if r.accounthistClients.Account == nil {
		logger.Info("accounthistClients.Account does not exists")
	} else {
		logger.Info("accounthistClients.Account  exists")
	}

	if args.Account != "" && args.Contract != nil && r.accounthistClients.AccountContract == nil {
		return fmt.Errorf("account history by contract not available")
	}
	if args.Account != "" && args.Contract == nil && r.accounthistClients.Account == nil {
		return fmt.Errorf("account history not available")
	}
	return nil
}

func (r *Root) handleActionStream(ctx context.Context, stream ActionReceiver, limit int64, cursor *pbaccounthist.Cursor) (*AccountHistoryActionsConnection, error) {
	zlogger := logging.Logger(ctx, zlog)

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
