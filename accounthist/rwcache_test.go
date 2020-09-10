package accounthist

import (
	"context"
	"testing"

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

	rwCache.Put(ctx, encodeActionKey(accountMama, 0, 0), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountMama, 0, 2), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountMama, 0, 1), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountPapa, 0, 24), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountPapa, 0, 23), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountDada, 0, 25), []byte{0xaa})
	rwCache.Put(ctx, encodeActionKey(accountPapa, 0, 25), []byte{0xaa})

	rwCache.BatchDelete(ctx, [][]byte{
		encodeActionKey(accountMama, 0, 1),
		encodeActionKey(accountPapa, 0, 23),
	})

	expectedKeys := [][]byte{
		encodeActionKey(accountMama, 0, 0),
		encodeActionKey(accountMama, 0, 2),
		encodeActionKey(accountPapa, 0, 24),
		encodeActionKey(accountDada, 0, 25),
		encodeActionKey(accountPapa, 0, 25),
	}
	i := 0
	rwCache.OrderedPuts(func(sKey string, value []byte) error {
		assert.Equal(t, string(expectedKeys[i]), sKey)
		i += 1
		return nil
	})
}
