package migrator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_retrieveContractAccounts(t *testing.T) {
	dataDir := "migration-data"
	accounts := []string{}
	i := &importer{
		common: common{dataDir: testMigrationDataDirPath(dataDir)},
	}

	contracts, err := i.retrieveContractAccounts(func(account string) error {
		accounts = append(accounts, account)
		return nil
	})

	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"battlefield1",
		"battlefield2",
		"battlefield3",
		"battlefield4",
		"eosio",
		"eosio.bpay",
		"eosio.msig",
		"eosio.ram",
		"eosio.token",
		"eosio2",
		"eosio3",
		"notified1",
		"notified2",
		"notified3",
		"notified4",
	}, accounts)

	ctrs := []string{}
	for _, contract := range contracts {
		ctrs = append(ctrs, contract.name)
	}
	assert.ElementsMatch(t, []string{
		"battlefield1",
		"battlefield3",
		"eosio",
		"eosio.msig",
		"eosio.token",
		"notified2",
	}, ctrs)

}

func Test_walkScopes(t *testing.T) {
	dataDir := "migration-data"
	scopes := []string{}
	accountPath, err := newAccountPath(testMigrationDataDirPath(dataDir), "eosio.token")
	require.NoError(t, err)

	walkScopes(fmt.Sprintf("%s/tables/accounts", accountPath), func(scope string) error {
		scopes = append(scopes, scope)
		return nil
	})

	assert.ElementsMatch(t, []string{
		"battlefield1",
		"battlefield3",
		"battlefield4",
		"eosio",
		"eosio.ram",
		"eosio.ramfee",
		"eosio.stake",
		"notified1",
		"notified2",
		"notified3",
		"notified4",
	}, scopes)

}
