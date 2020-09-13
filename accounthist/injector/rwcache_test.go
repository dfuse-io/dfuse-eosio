package injector

import (
	"context"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
)

func TestRWCache(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	rwCache := NewRWCache(kvStore)
	ctx := context.Background()

	accountMama := eos.MustStringToName("mama")
	accountPapa := eos.MustStringToName("papa")
	accountDada := eos.MustStringToName("dada")

	rwCache.Put(ctx, accounthist.AccountKey(accountMama).Row(0, 0), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountMama).Row(0, 2), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountMama).Row(0, 1), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountPapa).Row(0, 24), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountPapa).Row(0, 23), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountDada).Row(0, 25), []byte{0xaa})
	rwCache.Put(ctx, accounthist.AccountKey(accountPapa).Row(0, 25), []byte{0xaa})

	rwCache.BatchDelete(ctx, [][]byte{
		accounthist.AccountKey(accountMama).Row(0, 1),
		accounthist.AccountKey(accountPapa).Row(0, 23),
	})

	expectedKeys := [][]byte{
		accounthist.AccountKey(accountMama).Row(0, 0),
		accounthist.AccountKey(accountMama).Row(0, 2),
		accounthist.AccountKey(accountPapa).Row(0, 24),
		accounthist.AccountKey(accountDada).Row(0, 25),
		accounthist.AccountKey(accountPapa).Row(0, 25),
	}
	i := 0
	rwCache.OrderedPuts(func(sKey string, value []byte) error {
		assert.Equal(t, string(expectedKeys[i]), sKey)
		i += 1
		return nil
	})
}
