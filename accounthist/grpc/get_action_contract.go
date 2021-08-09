package grpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/logging"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/kvdb/store"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetAccountContractActions(req *pbaccounthist.GetTokenActionsRequest, stream pbaccounthist.AccountContractHistory_GetAccountContractActionsServer) error {
	if req.Limit < 0 {
		return status.Error(codes.InvalidArgument, "negative limit is not valid")
	}

	// TODO: triple check that `account` is an EOS Name (encode / decode and check for ==, otherwise, BadRequest), perhaps at the DGraphQL level plz
	account := req.Account
	contract := req.Contract
	limit := uint64(req.Limit)

	err := s.StreamAccountContractActions(stream.Context(), account, contract, limit, req.Cursor, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		if err := stream.Send(&pbaccounthist.ActionResponse{Cursor: cursor, ActionTrace: actionTrace}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return status.Errorf(codes.Unknown, "unable to stream actions: %s", err)
	}

	return nil

}

func (s *Server) StreamAccountContractActions(
	ctx context.Context,
	account uint64,
	contract uint64,
	limit uint64,
	cursor *pbaccounthist.Cursor,
	onAction func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error,
) error {
	logger := logging.Logger(ctx, zlog)

	queryShardNum := byte(0x00)
	querySeqNum := uint64(math.MaxUint64)
	if cursor != nil {
		// TODO: extract these from the key instead
		queryShardNum = byte(cursor.ShardNum)
		querySeqNum = cursor.SequenceNumber - 1
	}

	startKey := keyer.EncodeAccountContractKey(account, contract, queryShardNum, querySeqNum)
	endKey := store.Key(keyer.EncodeAccountContractPrefixKey(account, contract)).PrefixNext()

	if limit == 0 || limit > s.MaxEntries {
		limit = s.MaxEntries
	}

	logger.Debug("scanning actions",
		zap.Stringer("account", EOSName(account)),
		zap.Stringer("contract", EOSName(contract)),
		zap.String("start_key", hex.EncodeToString(startKey)),
		zap.String("end_key", hex.EncodeToString(endKey)),
		zap.Uint64("limit", limit),
	)

	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	it := s.KVStore.Scan(ctx, startKey, endKey, int(limit))
	for it.Next() {
		newact := &pbaccounthist.ActionRow{}
		err := proto.Unmarshal(it.Item().Value, newact)
		if err != nil {
			return fmt.Errorf("unmarshal action: %w", err)
		}

		_, _, shardNo, SeqNum := keyer.DecodeAccountContractKeySeqNum(it.Item().Key)
		if err := onAction(ActionKeyToCursor(it.Item().Key, shardNo, SeqNum), newact.ActionTrace); err != nil {
			return fmt.Errorf("on action: %w", err)
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("fetching actions: %w", err)
	}

	return nil
}
