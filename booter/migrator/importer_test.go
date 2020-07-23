package migrator

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go/ecc"

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

func Test_Importer(t *testing.T) {

	tests := []struct {
		fixture string
	}{
		{"migration-data"},
	}
	for _, test := range tests {
		testImporterData(t, test.fixture)
	}

}

func testImporterData(t *testing.T, dataDir string) {
	actions := make(chan interface{})
	receivedActions := []interface{}{}

	nonceActionEntropy = func() string {
		return "aaaaaaaaaaaa"
	}

	impt := &importer{
		common:      common{dataDir: testMigrationDataDirPath(dataDir)},
		opPublicKey: ecc.PublicKey{},
		actionChan:  actions,
		logger:      zap.NewNop(),
	}
	err := impt.init()
	require.NoError(t, err)

	go func() {
		defer close(actions)
		err := impt.inject()
		require.NoError(t, err)
	}()

	for {
		act, ok := <-actions
		if !ok {
			break
		}
		switch act.(type) {
		case bootops.TransactionBoundary:
			receivedActions = append(receivedActions, &TestActionWrapper{
				ActionType: "TransactionBoundary",
				Payload:    act.(bootops.TransactionBoundary),
			})
		case *bootops.TransactionAction:
			receivedActions = append(receivedActions, &TestActionWrapper{
				ActionType: "EOSAction",
				Payload:    act.(*bootops.TransactionAction),
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

	assert.JSONEq(t, expected, string(actual))
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
