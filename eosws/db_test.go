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

package eosws

// FIXME: FIX THOSE TESTS IN HERE, they were moved from `bigtable`,
// and it's the only place where we test
// `ListMostRecentTransactions`.. and those tests, we don't know if
// they broke since we brought them here.  Although they're only used
// in `eosq`, so still low prio.
// func TestListMostRecentTransactions(t *testing.T) {

// 	db, cleanup := newServer(t)
// 	defer cleanup()

// 	tests := []struct {
// 		name               string
// 		startKey           string
// 		limit              int
// 		chainDiscriminator func(blockID string) bool
// 		expectedErr        error
// 		expectedKeys       []string
// 		expectedCursor     string
// 	}{
// 		{
// 			name:         "Empty",
// 			startKey:     "00000000:0000",
// 			limit:        1,
// 			expectedKeys: nil,
// 		},
// 		{
// 			name:           "Empty with empty limit",
// 			startKey:       "00000000:0000",
// 			limit:          0,
// 			expectedKeys:   nil,
// 			expectedCursor: "00000000:0000",
// 		},
// 		{
// 			name:           "Single but first block",
// 			startKey:       "00000001:0000",
// 			limit:          1,
// 			expectedKeys:   []string{"01:00000001deadbeef"},
// 			expectedCursor: "",
// 		},
// 		{
// 			name:           "Single but midway in blocks list",
// 			startKey:       "00000003:0000",
// 			limit:          1,
// 			expectedKeys:   []string{"03:00000003deadbeef"},
// 			expectedCursor: "00000002beefdead:0000",
// 		},
// 		{
// 			name:         "MultipleSamePrefix_WithEndCursor",
// 			startKey:     "00000002deadbeef:0001",
// 			limit:        3,
// 			expectedKeys: []string{"02bb:00000002deadbeef", "02aa:00000002deadbeef", "01:00000001deadbeef"},
// 		},
// 		{
// 			name:         "MultipleSamePrefix_WithCursor",
// 			startKey:     "00000002deadbeef:0000",
// 			limit:        2,
// 			expectedKeys: []string{"02aa:00000002deadbeef", "01:00000001deadbeef"},
// 		},
// 		{
// 			name:           "MultipleSamePrefix_WithStartKey",
// 			startKey:       "00000002deadbeef:0001",
// 			limit:          1,
// 			expectedKeys:   []string{"02bb:00000002deadbeef"},
// 			expectedCursor: "00000002deadbeef:0000",
// 		},
// 		{
// 			name:           "MultipleSamePrefix_WithStartKey_MidBlock",
// 			startKey:       "00000002deadbeef:0002",
// 			limit:          2,
// 			expectedKeys:   []string{"02cc:00000002deadbeef", "02bb:00000002deadbeef"},
// 			expectedCursor: "00000002deadbeef:0000",
// 		},
// 		{
// 			name:               "MultipleAll",
// 			startKey:           "",
// 			limit:              100,
// 			chainDiscriminator: func(blockID string) bool { return strings.Contains(blockID, "deadbeef") },
// 			expectedKeys:       []string{"05:00000005deadbeef", "04:00000004deadbeef", "03:00000003deadbeef", "02cc:00000002deadbeef", "02bb:00000002deadbeef", "02aa:00000002deadbeef", "01:00000001deadbeef"},
// 			expectedCursor:     "",
// 		},
// 	}

// 	populateBigtableWithRows(t, db,
// 		blockInserter("00000001deadbeef", true, nil, []string{"01"}),
// 		blockInserter("00000002deadbeef", true, nil, []string{"02aa", "02bb", "02cc"}),
// 		blockInserter("00000002beefdead", false, nil, []string{"01"}),
// 		blockInserter("00000003deadbeef", true, nil, []string{"03"}),
// 		blockInserter("00000004deadbeef", true, nil, []string{"04"}),
// 		blockInserter("00000005deadbeef", false, nil, []string{"05"}),
// 		blockInserter("00000005beefdead", false, nil, []string{"05"}),

// 		executeTransaction("01", "00000001deadbeef", true),
// 		executeTransaction("01", "00000002beefdead", false),
// 		executeTransaction("02aa", "00000002deadbeef", true),
// 		executeTransaction("02bb", "00000002deadbeef", true),
// 		executeTransaction("02cc", "00000002deadbeef", true),
// 		executeTransaction("03", "00000003deadbeef", true),
// 		executeTransaction("04", "00000004deadbeef", true),
// 		executeTransaction("05", "00000005deadbeef", false),
// 		executeTransaction("05", "00000005beefdead", false),
// 	)

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			chainDiscriminator := alwaysInChain
// 			if test.chainDiscriminator != nil {
// 				chainDiscriminator = test.chainDiscriminator
// 			}

// 			lst, err := db.ListMostRecentTransactions(context.Background(), test.startKey, test.limit, chainDiscriminator)

// 			if test.expectedErr != nil {
// 				require.Error(t, err)
// 			} else {
// 				require.NoError(t, err)
// 				var keys []string
// 				for _, tx := range lst.Transactions {
// 					lifecycle, _ := pbcodec.MergeTransactionEvents(tx, chainDiscriminator)
// 					keys = append(keys, transactionLifecycleKey(lifecycle))
// 				}
// 				assert.Equal(t, test.expectedKeys, keys)
// 				assert.Equal(t, test.expectedCursor, lst.NextCursor)
// 			}

// 		})
// 	}
// }
