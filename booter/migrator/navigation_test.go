package migrator

import (
	"fmt"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_retrieveContractAccounts(t *testing.T) {
	dataDir := "migration-data"
	i := &importer{
		common: common{dataDir: testMigrationDataDirPath(dataDir)},
		logger: zap.NewNop(),
	}

	accounts, err := i.retrieveAccounts(func(account *Account) error {
		return nil
	})
	require.NoError(t, err)

	expectedAccounts := map[string]bool{
		"battlefeeld4": false,
		"battlefield":  false,
		"battlefield1": true,
		"battlefield2": false,
		"battlefield3": true,
		"battlefield4": false,
		"battlefield5": false,
		"eosio":        true,
		"eosio.bpay":   false,
		"eosio.msig":   true,
		"eosio.ram":    false,
		"eosio.token":  true,
		"eosio2":       false,
		"eosio3":       false,
		"eosio.names":  false,
		"eosio.ramfee": false,
		"eosio.saving": false,
		"eosio.stake":  false,
		"eosio.vpay":   false,
		"notified1":    false,
		"notified2":    true,
		"notified3":    false,
		"notified4":    false,
		"notified5":    false,
		"zzzzzzzzzzzz": false,
	}

	for _, account := range accounts {
		if _, found := expectedAccounts[account.name]; !found {
			assert.Fail(t, "Unable to find account in expected account list", "Account %q is not in expected account list", account.name)
		}

		assert.Equal(t, expectedAccounts[account.name], account.hasContract)
	}
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
		"battlefeeld4",
		"battlefield1",
		"battlefield3",
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
