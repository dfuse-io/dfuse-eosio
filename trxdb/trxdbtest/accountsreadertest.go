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

package trxdbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/kvdb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var accountsReaderTest = []struct {
	name string
	test func(t *testing.T, driverFactory DriverFactory)
}{
	{"TestGetAccount", TestGetAccount},
	{"TestListAccountNames", TestListAccountNames},
}

func TestAllAccountsReader(t *testing.T, driverName string, driverFactory DriverFactory) {
	for _, rt := range accountsReaderTest {
		t.Run(driverName+"/"+rt.name, func(t *testing.T) {
			rt.test(t, driverFactory)
		})
	}
}

func TestGetAccount(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name          string
		creator       string
		account       string
		accountLookup string
		expectCreator string
		expectAccount string
		expectErr     error
	}{
		{
			name:          "sunny path",
			account:       "eoscanada2",
			creator:       "eoscanada1",
			accountLookup: "eoscanada2",
			expectCreator: "eoscanada1",
			expectAccount: "eoscanada2",
		},
		{
			name:          "account not found",
			creator:       "eoscanada1",
			account:       "eoscanada2",
			accountLookup: "eoscanada3",
			expectErr:     kvdb.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, clean := driverFactory()
			defer clean()
			putAccount(t, test.creator, test.account, db)

			ref, err := db.GetAccount(context.Background(), test.accountLookup)
			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.creator, ref.Creator)
				assert.Equal(t, test.account, ref.Account)
			}

		})
	}
}

func TestListAccountNames(t *testing.T, driverFactory DriverFactory) {
	tests := []struct {
		name                string
		accounts            []string
		concurrentReadCount uint32
		expectAccounts      []string
		expectErr           error
	}{
		{
			name:                "sunny path",
			concurrentReadCount: 2,
			accounts:            []string{"eoscanada1", "eoscanada2", "eoscanada3"},
			expectAccounts:      []string{"eoscanada1", "eoscanada2", "eoscanada3"},
		},
		{
			name:                "concurrency greater then number of accouns",
			concurrentReadCount: 4,
			accounts:            []string{"eoscanada1", "eoscanada2", "eoscanada3"},
			expectAccounts:      []string{"eoscanada1", "eoscanada2", "eoscanada3"},
		},
		{
			name:                "no accounts",
			concurrentReadCount: 2,
			accounts:            []string{},
			expectAccounts:      []string{},
		},
		{
			name:                "concurrency with 0",
			concurrentReadCount: 0,
			expectErr:           fmt.Errorf("invalid concurrent read"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, clean := driverFactory()
			defer clean()

			for _, acc := range test.accounts {
				putAccount(t, "eoscanada0", acc, db)
			}
			accounts, err := db.ListAccountNames(context.Background(), test.concurrentReadCount)
			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, test.accounts, accounts)
			}

		})
	}
}

func putAccount(t *testing.T, creator, account string, db trxdb.DB) {
	actions := []string{`{"id":"a1","action_traces": [{"receiver":"eosio","action": {"account":"eosio","name":"newaccount"}}]}`}
	blk := TestBlock(t, "00000002aa", "00000001aa", actions...)
	blk.TransactionTraces()[0].ActionTraces[0].Action.JsonData = fmt.Sprintf(`
		{
		   "active": {
			  "accounts": [],
			  "keys": [
				 {
					"key": "EOS5UQzjPekK6g3y1LEdkBY8Seia1iqUhLHAPr55yPPguCN594UfU",
					"weight": 1
				 }
			  ],
			  "threshold": 1,
			  "waits": []
		   },
		   "creator": "%s",
		   "name": "%s",
		   "owner": {
			  "accounts": [],
			  "keys": [
				 {
					"key": "EOS5UQzjPekK6g3y1LEdkBY8Seia1iqUhLHAPr55yPPguCN594UfU",
					"weight": 1
				 }
			  ],
			  "threshold": 1,
			  "waits": []
		   }
		}
	`, creator, account)
	var newAccount *system.NewAccount
	require.NoError(t, json.Unmarshal([]byte(blk.TransactionTraces()[0].ActionTraces[0].Action.JsonData), &newAccount))
	data, err := eos.MarshalBinary(newAccount)
	require.NoError(t, err)
	blk.TransactionTraces()[0].ActionTraces[0].Action.RawData = data
	ctx := context.Background()
	require.NoError(t, db.PutBlock(ctx, blk))
	require.NoError(t, db.UpdateNowIrreversibleBlock(ctx, blk))
	require.NoError(t, db.Flush(ctx))
}
