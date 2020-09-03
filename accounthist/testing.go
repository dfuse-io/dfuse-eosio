package accounthist

import (
	"context"
	"fmt"
	"testing"

	"github.com/dfuse-io/bstream/forkable"
	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func streamBlocks(t *testing.T, s *Service, blocks ...*pbcodec.Block) {
	preprocessor := preprocessingFunc(s.blockFilter)

	for _, block := range blocks {
		blk := ct.ToBstreamBlock(t, block)
		obj, err := preprocessor(blk)
		require.NoError(t, err)

		s.ProcessBlock(blk, &forkable.ForkableObject{Obj: obj})
	}
}

type actionResult struct {
	cursor      *pbaccounthist.Cursor
	actionTrace *pbcodec.ActionTrace
}

func (r *actionResult) StringCursor() string {
	return fmt.Sprintf("%s:%02x:%d", eos.NameToString(r.cursor.Account), byte(r.cursor.ShardNum), r.cursor.SequenceNumber)
}

func listActions(t *testing.T, s *Service, account string, cursor *pbaccounthist.Cursor) (out []*actionResult) {
	ctx := context.Background()

	err := s.StreamActions(ctx, eos.MustStringToName(account), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		out = append(out, &actionResult{cursor, actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}

type actionBetaResult struct {
	cursor      string
	actionTrace *pbcodec.ActionTrace
}

func listBetaActions(t *testing.T, s *Service, account string, cursor *pbaccounthist.Cursor) (out []*actionBetaResult) {
	ctx := context.Background()

	err := s.StreamActions(ctx, eos.MustStringToName(account), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		cursorStr := fmt.Sprintf("%s:%02x:%d", eos.NameToString(cursor.Account), byte(cursor.ShardNum), cursor.SequenceNumber)
		out = append(out, &actionBetaResult{cursor: cursorStr, actionTrace: actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}
