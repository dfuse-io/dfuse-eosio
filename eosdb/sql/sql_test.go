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

package sql

import (
	"fmt"
	"os"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/eosdb/eosdbtest"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	//eosdbtest.TestAll(t, "SQL", newTestSqlFactory(t, "/tmp/kvdbtest.db"))

}

func newTestSqlFactory(t *testing.T, testDbFilename string) eosdbtest.DriverFactory {
	return func() (eosdb.Driver, eosdbtest.DriverCleanupFunc) {
		return newTestSqlFileDriver(t, testDbFilename), func() {
			err := os.Remove(testDbFilename)
			require.NoError(t, err)
		}
	}
}

func newTestSqlFileDriver(t *testing.T, testDbFilename string) eosdb.Driver {
	dns := fmt.Sprintf("sqlite3://%s?cache=shared&mode=memory&createTables=true", testDbFilename)
	rawdb, err := New(dns)
	require.NoError(t, err)
	return rawdb
}
