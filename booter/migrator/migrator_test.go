package migrator

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dfuse-io/eosio-boot/ops"

	rice "github.com/GeertJohan/go.rice"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/eoscanada/eos-go"

	"github.com/dfuse-io/logging"
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
		testMigrationData(t, test.fixture)
	}

}

func testMigrationData(t *testing.T, dataDir string) {
	actions := make(chan interface{})
	receivedActions := []interface{}{}

	migrator := &Migrator{
		box:         rice.MustFindBox("./code/build"),
		contract:    "dfuse.mgrt",
		opPublicKey: ecc.PublicKey{},
		actionChan:  actions,
		dataDir:     testMigrationDataDirPath(dataDir),
	}

	go func() {
		defer close(actions)
		migrator.startMigration()
	}()

	for {
		act, ok := <-actions
		if !ok {
			break
		}
		switch act.(type) {
		case *ops.TransactionBoundary:
			receivedActions = append(receivedActions, &TestActionWrapper{
				ActionType: "TransactionBoundary",
				Payload:    act.(*ops.TransactionBoundary),
			})
		case *eos.Action:
			receivedActions = append(receivedActions, &TestActionWrapper{
				ActionType: "EOSAction",
				Payload:    act.(*eos.Action),
			})
		}
	}

	actual, err := json.MarshalIndent(receivedActions, "", "  ")
	require.NoError(t, err)

	goldenfile := testMigrationDataDirGoldenFile(dataDir)

	if os.Getenv("GOLDEN_UPDATE") != "" {
		require.NoError(t, ioutil.WriteFile(goldenfile, actual, os.ModePerm))
	}
	expected := fromFixture(t, goldenfile)

	assert.JSONEqf(t, expected, string(actual), "Expected:\n%s\n\nActual:\n%s\n", expected, actual)
}

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
