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

package bigt

import (
	"testing"
	"time"

	"cloud.google.com/go/bigtable/bttest"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	"github.com/dfuse-io/dfuse-eosio/eosdb/eosdbtest"
	"github.com/dfuse-io/dgrpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestAll(t *testing.T) {
	eosdbtest.TestAll(t, "bigt", newTestBigtFactory(t))
}

func newTestBigtFactory(t *testing.T) eosdbtest.DriverFactory {
	return func() (eosdb.Driver, eosdbtest.DriverCleanupFunc) {
		return newTestBigtDriver(t)
	}
}

func newTestBigtDriver(t *testing.T) (eosdb.Driver, func()) {
	srv, err := bttest.NewServer("localhost:0")
	require.NoError(t, err)
	conn, err := dgrpc.NewInternalClient(srv.Addr)
	require.NoError(t, err)
	db, err := NewDriver("test", "dev", "dev", true, time.Second, 10, option.WithGRPCConn(conn))
	require.NoError(t, err)
	return db, func() {
		db.Close()
		srv.Close()
	}
}
