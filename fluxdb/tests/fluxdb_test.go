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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/bigtable/bttest"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/bigt"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/hidalgo"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/logging"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func TestFluxdb_HidalgoStore(t *testing.T) {
	runAll(t, func() (store.KVStore, StoreCleanupFunc) {
		dir, err := ioutil.TempDir("", "fluxdb-hidalgo")
		require.NoError(t, err)

		dsn := fmt.Sprintf("bbolt://%s?createTables=true", path.Join(dir, "db.bbolt"))
		kvStore, err := hidalgo.NewKVStore(context.Background(), dsn)
		require.NoError(t, err)

		return kvStore, func() {
			os.RemoveAll(dir)
			kvStore.Close()
		}
	})
}

//func TestFluxdb_BadgerStore(t *testing.T) {
//	runAll(t, func() (store.KVStore, StoreCleanupFunc) {
//		dir, err := ioutil.TempDir("", "fluxdb-badger")
//		require.NoError(t, err)
//
//		dsn := fmt.Sprintf("badger://%s", path.Join(dir, "flux.db"))
//
//		kvStore, err := badger.NewBadgerStore(context.Background(), dsn)
//		require.NoError(t, err)
//
//		return kvStore, func() {
//			os.RemoveAll(dir)
//			kvStore.Close()
//		}
//	})
//}

func TestFluxdb_Bigtable(t *testing.T) {
	runAll(t, func() (store.KVStore, StoreCleanupFunc) {
		srv, err := bttest.NewServer("localhost:0")
		require.NoError(t, err)
		conn, err := grpc.Dial(srv.Addr, grpc.WithInsecure())
		require.NoError(t, err)

		kvStore, err := bigt.NewKVStore(context.Background(), "bigtable://dev.dev/test?createTables=true", option.WithGRPCConn(conn))
		require.NoError(t, err)

		return kvStore, func() {
			srv.Close()
		}
	})
}

func runAll(t *testing.T, storeFactory StoreFactory) {
	tests := []struct {
		name   string
		tester e2eTester
	}{
		{"state table, single row, head, hex", testStateTableSingleHeadHex},
		{"state table, single row, head, json", testStateTableSingleHeadJSON},
		{"state table, single row, historical, json", testStateTableSingleHistoricalJSON},
		{"state table, multi rows, head, json", testStateTableMultiHeadJSON},
		{"state table, multi rows, historical, json", testStateTableMultiHistoricalJSON},

		{"state table scopes, head", testStateTableScopesHeadJSON},
		{"state table scopes, historical", testStateTableScopesHistoricalJSON},

		{"state tables for accounts, head", testStateTablesForAccountsHeadJSON},
		{"state tables for accounts, historical", testStateTablesForAccountsHistoricalJSON},

		{"state table row, historical", testStateTableRowHeadJSON},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e2eTest(t, storeFactory, test.tester)
		})
	}
}

func testStateTableSingleHeadHex(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "")

	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `[{"key":"eos","payer":"eosio1","hex":"a08601000000000004454f5300000000"}]`, response.Path("$.rows"))
}

func testStateTableSingleHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "json=true")

	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]`, response.Path("$.rows"))
}

func testStateTableSingleHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "json=true&block_num=4")

	assertIrrBlockInfo(response)
	jsonValueEqual(t, `[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]`, response.Path("$.rows"))
}

func testStateTableMultiHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.test/rows2/s", "json=true")

	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `[
		{"key":"b","payer":"s","json":{"to":20}},
		{"key":"c","payer":"s","json":{"to":3}},
		{"key":"d","payer":"s","json":{"to":4}},
		{"key":"e","payer":"s","json":{"to":5}},
		{"key":"f","payer":"s","json":{"to":6}}
	]`, response.Path("$.rows"))
}

func testStateTableMultiHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.test/rows/s", "json=true&block_num=3")

	assertIrrBlockInfo(response)
	jsonValueEqual(t, `[
		{"key":"a","payer":"s","json":{"from":"a"}},
		{"key":"b","payer":"s","json":{"from":"b2"}},
		{"key":"c","payer":"s","json":{"from":"c"}}
	]`, response.Path("$.rows"))
}

func testStateTableScopesHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableScopes(e, "eosio.token/accounts", "")

	response.ValueEqual("block_num", 6)
	jsonValueEqual(t, `["eosio1", "eosio2"]`, response.Path("$.scopes"))
}

func testStateTableScopesHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableScopes(e, "eosio.token/accounts", "block_num=3")

	response.ValueEqual("block_num", 3)
	jsonValueEqual(t, `["eosio1", "eosio2", "eosio3"]`, response.Path("$.scopes"))
}

func testStateTablesForScopesHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForScopes(e, "eosio.token/accounts/eosio1|eosio2|eosio3", "json=true")

	// That is not the correct behavior, there should be only `eosio1` & `eosio3` in the tests
	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `[
		{ "account": "eosio.token","scope": "eosio1", "rows": [{ "key": "eos", "payer": "eosio1", "json": {"balance":"10.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio2", "rows": [{ "key": "eos", "payer": "eosio2", "json": {"balance":"22.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio3", "rows": []}
	]`, response.Path("$.tables"))
}

func testStateTablesForScopesHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForScopes(e, "eosio.token/accounts/eosio1|eosio2|eosio3", "block_num=3&json=true")

	assertIrrBlockInfo(response)
	jsonValueEqual(t, `[
		{ "account": "eosio.token","scope": "eosio1", "rows": [{ "key": "eos", "payer": "eosio1", "json": {"balance":"1.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio2", "rows": [{ "key": "eos", "payer": "eosio2", "json": {"balance":"20.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio3", "rows": [{ "key": "eos", "payer": "eosio3", "json": {"balance":"3.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTablesForAccountsHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForAccounts(e, "eosio.token|eosio.nekot/accounts/eosio1", "json=true")

	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `[
		{"account":"eosio.nekot","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"1.0000 SOE"}}]},
		{"account":"eosio.token","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTablesForAccountsHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForAccounts(e, "eosio.token|eosio.nekot/accounts/eosio1", "block_num=4&json=true")

	assertIrrBlockInfo(response)
	jsonValueEqual(t, `[
		{"account":"eosio.nekot","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"1.0000 SOE"}}]},
		{"account":"eosio.token","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTableRowHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableRow(e, "eosio.nekot/accounts/eosio5/SOE", "json=true&key_type=symbol_code")

	assertHeadBlockInfo(response, "00000006aa")
	jsonValueEqual(t, `{"key":"SOE","payer":"eosio5","json":{"balance":"5.0000 SOE"}}`, response.Path("$.row"))
}

func tableBlocks(t *testing.T) []*pbeos.Block {
	eosioTokenABI1 := readABI(t, "eosio.token.1.abi.json")
	eosioTestABI1 := readABI(t, "eosio.test.1.abi.json")
	eosioTestABI2 := readABI(t, "eosio.test.2.abi.json")
	eosioNekotABI1 := readABI(t, "eosio.nekot.1.abi.json")

	return []*pbeos.Block{
		// Block #2 | Sets ABI on `eosio.token` (v1) and `eosio.test` (v1)
		testBlock(t, "00000002aa", "0000000000000000000000000000000000000000000000000000000000000000",
			trxTrace(t, actionSetABI(t, "eosio.token", eosioTokenABI1)),
			trxTrace(t, actionSetABI(t, "eosio.test", eosioTestABI1)),
		),

		// Block #3
		testBlock(t, "00000003aa", "00000002aa",
			// Creates three balances `eosio1`, `eosio2`, `eosio3` on `eosio.token`
			trxTrace(t,
				tableOp(t, "insert", "eosio.token/accounts/eosio1", "eosio1"),
				dbOp(t, eosioTokenABI1, "insert", "eosio.token/accounts/eosio1/eos", "/eosio1", `/{"balance":"1.0000 EOS"}`),

				tableOp(t, "insert", "eosio.token/accounts/eosio2", "eosio2"),
				dbOp(t, eosioTokenABI1, "insert", "eosio.token/accounts/eosio2/eos", "/eosio2", `/{"balance":"2.0000 EOS"}`),

				tableOp(t, "insert", "eosio.token/accounts/eosio3", "eosio3"),
				dbOp(t, eosioTokenABI1, "insert", "eosio.token/accounts/eosio3/eos", "/eosio3", `/{"balance":"3.0000 EOS"}`),
			),

			// Add three rows (keys `a`, `b` & `c`) to `eosio.test` contract, on table `rows` under scope `s`, then update key `b` within same transaction
			trxTrace(t,
				tableOp(t, "insert", "eosio.test/rows/s", "s"),
				dbOp(t, eosioTestABI1, "insert", "eosio.test/rows/s/a", "/s", `/{"from":"a"}`),
				dbOp(t, eosioTestABI1, "insert", "eosio.test/rows/s/b", "/s", `/{"from":"b"}`),
				dbOp(t, eosioTestABI1, "insert", "eosio.test/rows/s/c", "/s", `/{"from":"c"}`),
				dbOp(t, eosioTestABI1, "update", "eosio.test/rows/s/b", "s/s", `{"from":"b"}/{"from":"b2"}`),
			),

			// Update balance of `eosio2` on `eosio.token` within same block, but in different transaction
			trxTrace(t,
				dbOp(t, eosioTokenABI1, "update", "eosio.token/accounts/eosio2/eos", "eosio2/eosio2", `{"balance":"2.0000 EOS"}/{"balance":"20.0000 EOS"}`),
			),
		),

		// Block #4
		testBlock(t, "00000004aa", "00000003aa",
			// Add a new token contract `eosio.nekot` (to test `/tables/accounts` calls) and populate odd rows from `eosio.token`
			trxTrace(t,
				actionSetABI(t, "eosio.nekot", eosioNekotABI1),

				tableOp(t, "insert", "eosio.nekot/accounts/eosio1", "eosio1"),
				dbOp(t, eosioNekotABI1, "insert", "eosio.nekot/accounts/eosio1/eos", "/eosio1", `/{"balance":"1.0000 SOE"}`),

				tableOp(t, "insert", "eosio.nekot/accounts/eosio3", "eosio3"),
				dbOp(t, eosioNekotABI1, "insert", "eosio.nekot/accounts/eosio3/eos", "/eosio3", `/{"balance":"3.0000 SOE"}`),
			),

			// Modify `eosio.token` `eosio1` balance and delete `eosio3`
			trxTrace(t,
				dbOp(t, eosioTokenABI1, "update", "eosio.token/accounts/eosio1/eos", "eosio1/eosio1", `{"balance":"1.0000 EOS"}/{"balance":"10.0000 EOS"}`),

				dbOp(t, eosioTokenABI1, "remove", "eosio.token/accounts/eosio3/eos", "eosio3/", `{"balance":"3.0000 EOS"}/`),
				tableOp(t, "remove", "eosio.token/accounts/eosio3", "eosio3"),
			),
		),

		// Block #5
		testBlock(t, "00000005aa", "00000004aa",
			// Remove all rows (keys `a`, `b`) of `eosio.test`
			trxTrace(t,
				dbOp(t, eosioTestABI1, "remove", "eosio.test/rows/s/a", "s/", `{"from":"a"}/`),
				dbOp(t, eosioTestABI1, "remove", "eosio.test/rows/s/b", "s/", `{"from":"b2"}/`),
				dbOp(t, eosioTestABI1, "remove", "eosio.test/rows/s/b", "s/", `{"from":"c"}/`),
				tableOp(t, "remove", "eosio.test/rows/s", "s"),
			),

			// Set a new ABI on `eosio.test`
			trxTrace(t, actionSetABI(t, "eosio.test", eosioTestABI2)),

			// Re-add all rows on `eosio.test` using new ABI
			trxTrace(t,
				tableOp(t, "insert", "eosio.test/rows2/s", "s"),
				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/a", "/s", `/{"to":1}`),
				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/b", "/s", `/{"to":2}`),
				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/c", "/s", `/{"to":3}`),
			),

			// Add a new token contract `eosio.nekot` (to test `/tables/accounts` calls) and populate odd rows from `eosio.token`
			trxTrace(t,
				tableOp(t, "insert", "eosio.nekot/accounts/eosio5", "eosio5"),
				dbOp(t, eosioNekotABI1, "insert", "eosio.nekot/accounts/eosio5/........cpbp3", "/eosio5", `/{"balance":"5.0000 SOE"}`),
			),
		),

		// Block #6 | This block will be in the reversible segment, i.e. in the speculative writes
		testBlock(t, "00000006aa", "00000005aa",
			// Update balance of `eosio2` on `eosio.token`
			trxTrace(t,
				dbOp(t, eosioTokenABI1, "update", "eosio.token/accounts/eosio2/eos", "eosio2/eosio2", `{"balance":"20.0000 EOS"}/{"balance":"22.0000 EOS"}`),
			),

			// Delete rows `a` from `eosio.test`, update `b` and add three new rows (`d`, `e` & `f`)
			trxTrace(t,
				dbOp(t, eosioTestABI2, "remove", "eosio.test/rows2/s/a", "s/", `{"to":1}/`),

				dbOp(t, eosioTestABI2, "update", "eosio.test/rows2/s/b", "s/s", `{"to":2}/{"to":20}`),

				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/d", "/s", `/{"to":4}`),
				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/e", "/s", `/{"to":5}`),
				dbOp(t, eosioTestABI2, "insert", "eosio.test/rows2/s/f", "/s", `/{"to":6}`),
			),
		),
	}
}

func okQueryStateTable(e *httpexpect.Expect, table string, extraQuery string) (response *httpexpect.Object) {
	parts := strings.Split(table, "/")

	queryString := fmt.Sprintf("account=%s&table=%s&scope=%s", parts[0], parts[1], parts[2])
	if extraQuery != "" {
		queryString += "&" + extraQuery
	}

	return okQuery(e, "/v0/state/table", queryString)
}

func okQueryStateTableScopes(e *httpexpect.Expect, table string, extraQuery string) (response *httpexpect.Object) {
	parts := strings.Split(table, "/")

	queryString := fmt.Sprintf("account=%s&table=%s", parts[0], parts[1])
	if extraQuery != "" {
		queryString += "&" + extraQuery
	}

	return okQuery(e, "/v0/state/table_scopes", queryString)
}

func okQueryStateTablesForScopes(e *httpexpect.Expect, table string, extraQuery string) (response *httpexpect.Object) {
	parts := strings.Split(table, "/")

	queryString := fmt.Sprintf("account=%s&table=%s&scopes=%s", parts[0], parts[1], parts[2])
	if extraQuery != "" {
		queryString += "&" + extraQuery
	}

	return okQuery(e, "/v0/state/tables/scopes", queryString)
}

func okQueryStateTablesForAccounts(e *httpexpect.Expect, table string, extraQuery string) (response *httpexpect.Object) {
	parts := strings.Split(table, "/")

	queryString := fmt.Sprintf("accounts=%s&table=%s&scope=%s", parts[0], parts[1], parts[2])
	if extraQuery != "" {
		queryString += "&" + extraQuery
	}

	return okQuery(e, "/v0/state/tables/accounts", queryString)
}

func okQueryStateTableRow(e *httpexpect.Expect, table string, extraQuery string) (response *httpexpect.Object) {
	parts := strings.Split(table, "/")

	queryString := fmt.Sprintf("account=%s&table=%s&scope=%s&primary_key=%s", parts[0], parts[1], parts[2], parts[3])
	if extraQuery != "" {
		queryString += "&" + extraQuery
	}

	return okQuery(e, "/v0/state/table/row", queryString)
}

func okQuery(e *httpexpect.Expect, path string, queryString string) (response *httpexpect.Object) {
	return e.GET(path).
		WithQueryString(queryString).
		Expect().
		Status(http.StatusOK).JSON().Object()
}

func assertIrrBlockInfo(response *httpexpect.Object) {
	response.NotContainsKey("up_to_block_id")
	response.NotContainsKey("up_to_block_num")

	// FIXME: When are those supposed to be filled exactly? They are not (in our tests). Tested on `eos-dev1`
	//        for an historical block, same results. Need to dig down, maybe something is broken here.
	response.ValueEqual("last_irreversible_block_id", "")
	response.ValueEqual("last_irreversible_block_num", 0)
}

func assertHeadBlockInfo(response *httpexpect.Object, blockRef string) {
	ref := bstream.BlockRefFromID(blockRef)

	response.ValueEqual("up_to_block_id", ref.ID())
	response.ValueEqual("up_to_block_num", ref.Num())
	response.ValueEqual("last_irreversible_block_id", "")
	response.ValueEqual("last_irreversible_block_num", 0)
}
