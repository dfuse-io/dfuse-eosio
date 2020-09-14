package injector

import (
	"testing"

	ct "github.com/dfuse-io/dfuse-eosio/codec/testing"
	"github.com/stretchr/testify/assert"
)

func Test_AccountContractLiveShardWithTransfers(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := setupAccountContractInjector(NewRWCache(kvStore), 0, 2)

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:battlefieldt:init")),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "04c524a080000000005530ea033482a60000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "some1:eosio.token:transfer", ct.GlobalSequence(1))},
	}, listAccountContractActions(t, s, "some1", "eosio.token", nil))
}

func Test_AccountContractLiveShard_ActionGate(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := setupAccountContractInjector(NewRWCache(kvStore), 0, 2)

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:battlefieldt:init")),
		),
	)

	assert.Equal(t, []*actionResult(nil), listAccountContractActions(t, s, "some1", "battlefieldt", nil))
}

func Test_Test_AccountContractLiveShard_DeleteWindow(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()

	s := setupAccountContractInjector(NewRWCache(kvStore), 0, 2)

	autoGlobalSequence := ct.AutoGlobalSequence()

	streamBlocks(t, s,
		ct.Block(t, "00000001aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:battlefieldt:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some2:eosio.token:transfer")),
		),

		ct.Block(t, "00000002aa", autoGlobalSequence,
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:eosio.token:transfer")),
			ct.TrxTrace(t, ct.ActionTrace(t, "some1:battlefieldt:transfer")),
		),
	)

	assert.Equal(t, []*actionResult{
		{cursor: "04c524a0800000000039b398a96e54539000fffffffffffffffd:00:2", actionTrace: ct.ActionTrace(t, "some1:battlefieldt:transfer", ct.GlobalSequence(7))},
		{cursor: "04c524a0800000000039b398a96e54539000fffffffffffffffe:00:1", actionTrace: ct.ActionTrace(t, "some1:battlefieldt:transfer", ct.GlobalSequence(1))},
	}, listAccountContractActions(t, s, "some1", "battlefieldt", nil))

	assert.Equal(t, []*actionResult{
		{cursor: "04c524a100000000005530ea033482a60000fffffffffffffffb:00:4", actionTrace: ct.ActionTrace(t, "some2:eosio.token:transfer", ct.GlobalSequence(5))},
		{cursor: "04c524a100000000005530ea033482a60000fffffffffffffffc:00:3", actionTrace: ct.ActionTrace(t, "some2:eosio.token:transfer", ct.GlobalSequence(4))},
	}, listAccountContractActions(t, s, "some2", "eosio.token", nil))
}
