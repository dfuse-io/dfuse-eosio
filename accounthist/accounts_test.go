package accounthist

import (
	"context"
	"fmt"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
)

func Test_scanAccount(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	ctx := context.Background()
	shardZeroService := newTestService(kvStore, 0, 10)

	accountA := eos.MustStringToName("mama")
	accountB := eos.MustStringToName("eoscanadacom")
	accountC := eos.MustStringToName("eoscanadaca")
	accountD := eos.MustStringToName("eosio.token")

	insertKeys(ctx, shardZeroService, accountA, 2, 10)
	insertKeys(ctx, shardZeroService, accountC, 1, 12)

	shard1Service := newTestService(kvStore, 1, 10)
	insertKeys(ctx, shard1Service, accountA, 1, 4)
	insertKeys(ctx, shard1Service, accountB, 1, 5)
	insertKeys(ctx, shard1Service, accountC, 1, 6)
	insertKeys(ctx, shard1Service, accountD, 3, 7)

	shard2Service := newTestService(kvStore, 2, 10)
	insertKeys(ctx, shard2Service, accountA, 1, 1)
	insertKeys(ctx, shard2Service, accountB, 1, 2)
	insertKeys(ctx, shard2Service, accountC, 1, 3)

	expectedAccounts := []struct {
		account    uint64
		shard      byte
		ordinalNum uint64
	}{
		{account: accountC, shard: 0, ordinalNum: 1},
		{account: accountB, shard: 1, ordinalNum: 1},
		{account: accountD, shard: 1, ordinalNum: 3},
		{account: accountA, shard: 0, ordinalNum: 2},
	}
	index := 0
	shardZeroService.ScanAccounts(context.Background(), func(account uint64, shard byte, ordinalNum uint64) error {
		fmt.Println("account: ", eos.NameToString(account))
		assert.Equal(t, expectedAccounts[index], struct {
			account    uint64
			shard      byte
			ordinalNum uint64
		}{account: account, shard: shard, ordinalNum: ordinalNum},
		)
		index++
		return nil
	})
}
