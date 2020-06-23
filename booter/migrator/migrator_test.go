package migrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/dfuse-io/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func Test_Migrator(t *testing.T) {
	tests := []struct {
		fixture string
	}{
		{"migration-data"},
	}
	for _, test := range tests {
		testWalkContracts(t, test.fixture)
		testWalkScope(t, test.fixture)
		//testMigrationData(t, test.fixture)
	}

}
func testWalkContracts(t *testing.T, dataDir string) {
	contracts := []string{}
	walkContracts(testMigrationDataDirPath(dataDir), func(contract string) error {
		contracts = append(contracts, contract)
		return nil
	})

	assert.ElementsMatch(t, []string{
		"battlefield1",
		"battlefield3",
		"eosio",
		"eosio.msig",
		"eosio.token",
		"notified2",
	}, contracts)

}

func testWalkScope(t *testing.T, dataDir string) {
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

//func testMigrationData(t *testing.T, dataDir string) {
//	actions := make(chan interface{})
//	receivedActions := []interface{}{}
//
//	migrator := &Migrator{
//		box:         rice.MustFindBox("./code/build"),
//		contract:    "dfuse.mgrt",
//		opPublicKey: ecc.PublicKey{},
//		actionChan:  actions,
//		dataDir:     testMigrationDataDirPath(dataDir),
//	}
//
//	go func() {
//		defer close(actions)
//		migrator.migrate()
//	}()
//
//	for {
//		act, ok := <-actions
//		if !ok {
//			break
//		}
//		switch act.(type) {
//		case *ops.TransactionBoundary:
//			receivedActions = append(receivedActions, &TestActionWrapper{
//				ActionType: "TransactionBoundary",
//				Payload:    act.(*ops.TransactionBoundary),
//			})
//		case *eos.Action:
//			receivedActions = append(receivedActions, &TestActionWrapper{
//				ActionType: "EOSAction",
//				Payload:    act.(*eos.Action),
//			})
//		}
//	}
//
//	actual, err := json.MarshalIndent(receivedActions, "", "  ")
//	require.NoError(t, err)
//
//	goldenfile := testMigrationDataDirGoldenFile(dataDir)
//
//	if os.Getenv("GOLDEN_UPDATE") != "" {
//		require.NoError(t, ioutil.WriteFile(goldenfile, actual, os.ModePerm))
//	}
//	expected := fromFixture(t, goldenfile)
//
//	assert.JSONEqf(t, expected, string(actual), "Expected:\n%s\n\nActual:\n%s\n", expected, actual)
//}

type TestActionWrapper struct {
	ActionType string      `json:"type"`
	Payload    interface{} `json:"payload"`
}

var migrationDataNormalizeRegexp = regexp.MustCompile("[^a-z0-9_]")

func testMigrationDataDirPath(dirName string) string {
	return filepath.Join("test-data", dirName)
}

func testMigrationDataDirGoldenFile(dirName string) string {
	normalized := migrationDataNormalizeRegexp.ReplaceAllString(dirName, "_")
	return filepath.Join("test-data", normalized+".golden.json")
}

func fromFixture(t *testing.T, path string) string {
	t.Helper()

	cnt, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	return string(cnt)
}
