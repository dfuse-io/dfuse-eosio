package migrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eoscanada/eos-go"
	eossnapshot "github.com/eoscanada/eos-go/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//{
//	"____comment": "This file was generated with eosio-abigen. DO NOT EDIT ",
//	"version": "eosio::abi/1.1",
//}

var ABI = []byte{
	0x7B, 0x0A, 0x20, 0x20, 0x20, 0x20, 0x22, 0x5F, 0x5F, 0x5F, 0x5F, 0x63, 0x6F, 0x6D, 0x6D, 0x65, 0x6E, 0x74, 0x22, 0x3A,
	0x20, 0x22, 0x54, 0x68, 0x69, 0x73, 0x20, 0x66, 0x69, 0x6C, 0x65, 0x20, 0x77, 0x61, 0x73, 0x20, 0x67, 0x65, 0x6E, 0x65,
	0x72, 0x61, 0x74, 0x65, 0x64, 0x20, 0x77, 0x69, 0x74, 0x68, 0x20, 0x65, 0x6F, 0x73, 0x69, 0x6F, 0x2D, 0x61, 0x62, 0x69,
	0x67, 0x65, 0x6E, 0x2E, 0x20, 0x44, 0x4F, 0x20, 0x4E, 0x4F, 0x54, 0x20, 0x45, 0x44, 0x49, 0x54, 0x20, 0x22, 0x2C, 0x0A,
	0x20, 0x20, 0x20, 0x20, 0x22, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6F, 0x6E, 0x22, 0x3A, 0x20, 0x22, 0x65, 0x6F, 73, 0x69,
	0x6F, 0x3A, 0x3A, 0x61, 0x62, 0x69, 0x2F, 0x31, 0x2E, 0x31, 0x22, 0x2C, 0x0A, 0x7D,
}

var zlog = zap.NewNop()

func init() {
	if os.Getenv("TRACE") == "true" {
		traceEnable = true
	}
	if os.Getenv("DEBUG") != "" {
		zlog, _ = zap.NewDevelopment()
	}
}

func TestSnapshotRead(t *testing.T) {
	tests := []struct {
		name     string
		testFile string
	}{
		//{name: "eos-jdev", testFile: "eos-jdev_0000000638.bin"},
		//{name: "eos-dev1 snapshot", testFile: "eos-dev1_0004841949.bin"},
		{name: "battlefield snapshot", testFile: "battlefield-snapshot.bin"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputDir := testOutputDir(test.testFile)
			testFile := testData(test.testFile)
			e, err := NewExporter(testFile, outputDir, WithLogger(zlog))
			require.NoError(t, err)
			err = e.Export()
			require.NoError(t, err)
			return
		})
	}
}

func testOutputDir(inFilename string) string {
	chunks := strings.Split(inFilename, ".")
	return filepath.Join("test-data", "snapshot", "out", chunks[0])
}

func testData(filename string) string {
	return filepath.Join("test-data", "snapshot", filename)
}

func TestSnapshot_processAccount(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "golden path",
			input: eossnapshot.AccountObject{
				Name: "battlefield3",
				CreationDate: eos.BlockTimestamp{
					Time: time.Now(),
				},
				RawABI: ABI,
			},
		},
		{
			name:        "invalid object type",
			input:       "not the right type",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := exporter{
				logger:   zlog,
				accounts: map[eos.AccountName]*Account{},
			}
			err := e.processAccountObject(test.input)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})

	}

}
