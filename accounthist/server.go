package accounthist

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (ws *Service) GetActions(req *pbaccounthist.GetActionsRequest, stream pbaccounthist.AccountHistory_GetActionsServer) error {
	ctx := stream.Context()
	account := req.Account
	accountName := eos.NameToString(account)

	// TODO: triple check that `account` is an EOS Name (encode / decode and check for ==, otherwise, BadRequest), perhaps at the DGraphQL level plz

	queryShardNum := byte(255)
	querySeqNum := uint64(math.MaxUint64)
	if req.Cursor != nil {
		// TODO: we could check that the Cursor.ShardNum doesn't go above 255
		queryShardNum = byte(req.Cursor.ShardNum)
		querySeqNum = req.Cursor.SequenceNumber - 1 // FIXME: CHECK BOUNDARIES, this is EXCLUSIVE, so do we -1, +1 ?
	}

	if req.Limit < 0 {
		return fmt.Errorf("negative limit is not valid")
	}

	startKey := make([]byte, actionKeyLen)
	encodeActionKey(startKey, account, queryShardNum, querySeqNum)
	endKey := make([]byte, actionKeyLen)
	encodeActionKey(endKey, account, 0, 0)

	zlog.Info("scanning actions",
		zap.String("account", accountName),
		zap.String("start_key", hex.EncodeToString(startKey)), // TODO: turn into a hex Stringer(), instead of encoding it all the time
		zap.String("end_key", hex.EncodeToString(endKey)),     // TODO: turn into a hex Stringer(), instead of encoding it all the time
	)

	limit := int(ws.maxEntriesPerAccount)
	if req.Limit != 0 && int(req.Limit) < limit {
		limit = int(req.Limit)
	}
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()
	it := ws.kvStore.Scan(ctx, startKey, endKey, limit)
	for it.Next() {
		newact := &pbaccounthist.ActionRow{}
		err := proto.Unmarshal(it.Item().Value, newact)
		if err != nil {
			return err
		}

		newresp := &pbaccounthist.ActionResponse{
			Cursor:      actionKeyToCursor(account, it.Item().Key),
			ActionTrace: newact.ActionTrace,
		}

		if err := stream.Send(newresp); err != nil {
			return err
		}
	}
	if err := it.Err(); err != nil {
		return fmt.Errorf("error while fetching actions from store: %w", err)
	}

	return nil
}
