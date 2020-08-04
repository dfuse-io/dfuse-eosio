// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var b = decodeHex
var str = fmt.Sprintf

type exceptLogger zap.SugaredLogger

func (l *exceptLogger) Logf(fmt string, args ...interface{}) {
	(*zap.SugaredLogger)(l).Debugf(fmt, args...)
}

func readABI(t *testing.T, abiFile string) (out *eos.ABI) {
	path := path.Join("testdata", abiFile)
	abiJSON, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	out = new(eos.ABI)
	err = json.Unmarshal(abiJSON, out)
	require.NoError(t, err)

	return
}

func jsonValueEqual(t *testing.T, tag string, expected string, actual *httpexpect.Value) {
	out, err := json.Marshal(actual.Raw())
	require.NoError(t, err)

	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		require.Fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()))
	}

	if err := json.Unmarshal(out, &actualJSONAsInterface); err != nil {
		require.Fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", string(out), err.Error()))
	}

	if !assert.ObjectsAreEqual(expectedJSONAsInterface, actualJSONAsInterface) {
		assert.Fail(t, fmt.Sprintf("JSON Not Equal\n%s", unifiedDiff(t, tag, expectedJSONAsInterface, actualJSONAsInterface)))
	}
}

func unifiedDiff(t *testing.T, tag string, expected, actual interface{}) string {
	expectedFile := fmt.Sprintf("/tmp/gotests-statedb-%s-expected", tag)
	actualFile := fmt.Sprintf("/tmp/gotests-statedb-%s-actual", tag)

	defer func() {
		os.Remove(expectedFile)
		os.Remove(actualFile)
	}()

	expectedBytes, err := json.MarshalIndent(expected, "", "  ")
	require.NoError(t, err)

	actualBytes, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err)

	err = ioutil.WriteFile(expectedFile, expectedBytes, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(actualFile, actualBytes, 0600)
	require.NoError(t, err)

	fmt.Println("Expected", string(expectedBytes))

	cmd := exec.Command("diff", "-u", expectedFile, actualFile)
	out, _ := cmd.Output()

	return string(out)
}

func decodeHex(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return out
}
