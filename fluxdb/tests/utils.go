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
	"path"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/gavv/httpexpect/v2"
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

func jsonValueEqual(t *testing.T, expected string, actual *httpexpect.Value) {
	out, err := json.Marshal(actual.Raw())
	require.NoError(t, err)

	require.JSONEq(t, expected, string(out))
}

func decodeHex(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return out
}
