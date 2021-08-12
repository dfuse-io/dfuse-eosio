package injector

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/streamingfast/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/grpc"
	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/logging"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.TestingOverride()
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
				zlog.Debug("badger key",
					zap.Stringer("key", accounthist.RowKey(it.Item().Key)),
					zap.String("value", hex.EncodeToString(it.Item().Value)),
				)
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

func setupAccountInjector(kvStore store.KVStore, shardNum byte, maxEntries uint64) *Injector {
	i := NewInjector(
		NewRWCache(kvStore),
		nil,
		nil,
		shardNum,
		maxEntries,
		1,
		0,
		0,
		nil)
	i.lastCheckpoint = &pbaccounthist.ShardCheckpoint{}
	i.SetFacetFactory(&accounthist.AccountFactory{})
	return i
}

func setupAccountContractInjector(kvStore store.KVStore, shardNum byte, maxEntries uint64) *Injector {
	i := NewInjector(
		NewRWCache(kvStore),
		nil,
		nil,
		shardNum,
		maxEntries,
		1,
		0,
		0,
		nil)
	i.lastCheckpoint = &pbaccounthist.ShardCheckpoint{}
	i.SetFacetFactory(&accounthist.AccountContractFactory{})
	return i
}

func streamBlocks(t *testing.T, s *Injector, blocks ...*pbcodec.Block) {
	preprocessor := PreprocessingFunc(s.BlockFilter)

	for _, block := range blocks {
		blk := ct.ToBstreamBlock(t, block)
		obj, err := preprocessor(blk)
		require.NoError(t, err)

		s.ProcessBlock(blk, &forkable.ForkableObject{Obj: obj, Step: forkable.StepIrreversible})
	}
}

type actionResult struct {
	cursor      string
	actionTrace *pbcodec.ActionTrace
}

func listAccountActions(t *testing.T, s *Injector, act string, cursor *pbaccounthist.Cursor) (out []*actionResult) {
	ctx := context.Background()
	// TODO: this is kinda ugly maybe should cast the interace

	server := grpc.Server{KVStore: s.KvStore, MaxEntries: s.MaxEntries}
	err := server.StreamAccountActions(ctx, eos.MustStringToName(act), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		cursorStr := fmt.Sprintf("%x:%02x:%d", cursor.Key, byte(cursor.ShardNum), cursor.SequenceNumber)
		out = append(out, &actionResult{cursor: cursorStr, actionTrace: actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}

func listAccountContractActions(t *testing.T, s *Injector, act, ctr string, cursor *pbaccounthist.Cursor) (out []*actionResult) {
	ctx := context.Background()
	// TODO: this is kinda ugly maybe should cast the interace

	server := grpc.Server{KVStore: s.KvStore, MaxEntries: s.MaxEntries}
	err := server.StreamAccountContractActions(ctx, eos.MustStringToName(act), eos.MustStringToName(ctr), 1000, nil, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		cursorStr := fmt.Sprintf("%x:%02x:%d", cursor.Key, byte(cursor.ShardNum), cursor.SequenceNumber)
		out = append(out, &actionResult{cursor: cursorStr, actionTrace: actionTrace})
		return nil
	})
	require.NoError(t, err)

	return out
}

func insertKeys(ctx context.Context, s *Injector, account uint64, keyCount int, sequenceNumber uint64) [][]byte {
	revOrderInsertKeys := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		acctSeqData := accounthist.SequenceData{CurrentOrdinal: uint64(i + 1), LastGlobalSeq: (sequenceNumber + 1)}
		revOrderInsertKeys[keyCount-1-i] = keyer.EncodeAccountKey(account, s.ShardNum, acctSeqData.CurrentOrdinal)
		s.WriteAction(ctx, accounthist.AccountFacet(account), acctSeqData, []byte{})
	}
	s.ForceFlush(ctx)
	return revOrderInsertKeys
}

func runShard(t *testing.T, shardNum byte, maxEntriesPerAccount uint64, kvStore store.KVStore, blocks ...*pbcodec.Block) *Injector {
	s := setupAccountInjector(NewRWCache(kvStore), shardNum, maxEntriesPerAccount)
	streamBlocks(t, s, blocks...)
	return s
}
