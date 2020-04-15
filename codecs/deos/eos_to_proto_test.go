// Copyright 2019 dfuse Platform Inc.
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

package deos

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/require"
)

func TestActionToDEOS(t *testing.T) {

	cases := []struct {
		name             string
		json             string
		expectedJSONData string
		expectedRawData  string
	}{
		{
			name:             "with data",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":{"account":"eosio","vmtype":0,"vmversion":0,"code":"00"},"hex_data":"00"}`,
			expectedJSONData: `{"account":"eosio","code":"00","vmtype":0,"vmversion":0}`,
			expectedRawData:  "00",
		},
		{
			name:             "empty string data",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":"","hex_data":""}`,
			expectedJSONData: ``,
			expectedRawData:  "",
		},
		{
			name:             "empty object data",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":{},"hex_data":"00"}`,
			expectedJSONData: `{}`,
			expectedRawData:  "00",
		},
		{
			name:             "no data",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"hex_data":"00"}`,
			expectedJSONData: ``,
			expectedRawData:  "00",
		},
		{
			name:             "json data is pure number",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":1,"hex_data":"01"}`,
			expectedJSONData: `1`,
			expectedRawData:  "01",
		},
		{
			name:             "json data is pure string",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":"caracola","hex_data":"0863617261636f6c61"}`,
			expectedJSONData: `"caracola"`,
			expectedRawData:  "0863617261636f6c61",
		},
		{
			name:             "json data is actually hex",
			json:             `{"account":"eosio","name":"setcode","authorization":[{"actor":"eosio","permission":"active"}],"data":"abde"}`,
			expectedJSONData: ``,
			expectedRawData:  "abde",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := &eos.Action{}
			err := json.Unmarshal([]byte(c.json), &a)
			require.NoError(t, err)

			deosAction := ActionToDEOS(a)
			require.Equal(t, c.expectedJSONData, deosAction.JsonData)
			require.Equal(t, c.expectedRawData, hex.EncodeToString(deosAction.RawData))
		})
	}
}
