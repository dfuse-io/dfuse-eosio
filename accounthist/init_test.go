package accounthist

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/dfuse-io/bstream/forkable"
	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"

	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.TestingOverride()

	// 02 0000000080a024c5 ff ffffffffffffffff
	// 02 0000000000a124c5 ff ffffffffffffffff
}

func getKVTestFactory(t *testing.T) (store.KVStore, func()) {
	tmp, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	kvStore, err := store.New(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
	require.NoError(t, err)

	closer := func() {
		if traceEnabled {
			endKey := make([]byte, 512)
			for i := 0; i < len(endKey); i++ {
				endKey[i] = 0xFF
			}

			it := kvStore.Scan(context.Background(), []byte{}, endKey, int(math.MaxInt64))
			for it.Next() {
				zlog.Debug("badger key", zap.Stringer("key", Key(it.Item().Key)), zap.Stringer("value", Key(it.Item().Value)))
			}

			require.NoError(t, it.Err())
		}

		kvStore.Close()
		os.RemoveAll(tmp)
	}

	return kvStore, closer
}

func assertEqualHex(t *testing.T, expected string, actual []byte, msgAndArgs ...interface{}) {
	assert.Equal(t, expected, hex.EncodeToString(actual))
}

func newTestService(kvStore store.KVStore, shardNum byte, maxEntriesPerAccount uint64) *Service {
	return &Service{
		shardNum:             shardNum,
		maxEntriesPerAccount: maxEntriesPerAccount,
		flushBlocksInterval:  1,
		kvStore:              NewRWCache(kvStore),
		historySeqMap:        map[uint64]SequenceData{},
		lastCheckpoint:       &pbaccounthist.ShardCheckpoint{},
	}

}
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
	cursor      string
	actionTrace *pbcodec.ActionTrace
}

func listActions(t *testing.T, s *Service, account string, cursor *pbaccounthist.Cursor) (out []*actionResult) {
	ctx := context.Background()

	err := s.StreamActions(ctx, eos.MustStringToName(account), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		cursorStr := fmt.Sprintf("%s:%02x:%d", eos.NameToString(cursor.Account), byte(cursor.ShardNum), cursor.SequenceNumber)
		out = append(out, &actionResult{cursor: cursorStr, actionTrace: actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}
