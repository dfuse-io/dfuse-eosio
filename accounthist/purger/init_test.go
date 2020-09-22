package purger

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/dfuse-io/logging"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"

	"github.com/dfuse-io/dfuse-eosio/accounthist/injector"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/dfuse-io/kvdb/store"
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

func setupAccountInjector(kvStore store.KVStore, shardNum byte, maxEntries uint64) *injector.Injector {
	i := injector.NewInjector(
		injector.NewRWCache(kvStore),
		nil,
		nil,
		shardNum,
		maxEntries,
		1,
		0,
		0,
		nil)
	i.SetFacetFactory(&accounthist.AccountFactory{})
	return i
}

func insertKeys(ctx context.Context, s *injector.Injector, account uint64, keyCount int, sequenceNumber uint64) [][]byte {
	revOrderInsertKeys := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		acctSeqData := accounthist.SequenceData{CurrentOrdinal: uint64(i + 1), LastGlobalSeq: (sequenceNumber + 1)}
		revOrderInsertKeys[keyCount-1-i] = keyer.EncodeAccountKey(account, s.ShardNum, acctSeqData.CurrentOrdinal)
		s.WriteAction(ctx, accounthist.AccountFacet(account), acctSeqData, []byte{})
	}
	s.ForceFlush(ctx)
	return revOrderInsertKeys
}
