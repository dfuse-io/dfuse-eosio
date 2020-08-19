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
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	_ "github.com/dfuse-io/dfuse-eosio/codec"
	_ "github.com/dfuse-io/kvdb/store/badger"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/fluxdb/store"
	fluxdbKV "github.com/dfuse-io/fluxdb/store/kv"
	"github.com/dfuse-io/logging"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

func TestAll(t *testing.T) {
	runAll(t, getKVTestFactory(t))
}

func getKVTestFactory(t *testing.T) func() (store.KVStore, StoreCleanupFunc) {
	return func() (store.KVStore, StoreCleanupFunc) {
		tmp, err := ioutil.TempDir("", "badger")
		require.NoError(t, err)
		kvStore, err := fluxdbKV.NewStore(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
		require.NoError(t, err)

		closer := func() {
			kvStore.Close()
			os.RemoveAll(tmp)
		}

		return kvStore, closer
	}
}

func runAll(t *testing.T, storeFactory StoreFactory) {
	all := map[string][]e2eTester{
		"table": {
			testStateTableSingleRowHeadHex,
			testStateTableSingleRowHeadJSON,
			testStateTableSingleRowHistoricalJSON,
			testStateTableMultiRowsHeadJSON,
			testStateTableMultiRowsHistoricalJSON,
		},
		"table_scope": {
			testStateTableScopesHeadJSON,
			testStateTableScopesHistoricalJSON,
		},
		"tables_for_accounts": {
			testStateTablesForAccountsHeadJSON,
			testStateTablesForAccountsHistoricalJSON,
		},
		"table_row": {
			testStateTableRowHeadJSON,
		},
	}

	for group, tests := range all {
		for _, test := range tests {
			t.Run(group+"/"+getFunctionName(test), func(t *testing.T) {
				e2eTest(t, storeFactory, test)
			})
		}
	}
}

func testStateTableSingleRowHeadHex(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "")

	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "table-rows", `[{"key":"eos","payer":"eosio1","hex":"a08601000000000004454f5300000000"}]`, response.Path("$.rows"))
}

func testStateTableSingleRowHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "json=true")

	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "table-rows", `[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]`, response.Path("$.rows"))
}

func testStateTableSingleRowHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.token/accounts/eosio1", "json=true&block_num=4")

	assertIrrBlockInfo(response, "00000005aa")
	jsonValueEqual(t, "table-rows", `[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]`, response.Path("$.rows"))
}

func testStateTableMultiRowsHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.test/rows2/s", "json=true")

	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "table-rows", `[
		{"key":"b","payer":"s","json":{"to":20}},
		{"key":"c","payer":"s","json":{"to":3}},
		{"key":"d","payer":"s","json":{"to":4}},
		{"key":"e","payer":"s","json":{"to":5}},
		{"key":"f","payer":"s","json":{"to":6}}
	]`, response.Path("$.rows"))
}

func testStateTableMultiRowsHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTable(e, "eosio.test/rows/s", "json=true&block_num=3")

	assertIrrBlockInfo(response, "00000005aa")
	jsonValueEqual(t, "table-rows", `[
		{"key":"a","payer":"s","json":{"from":"a"}},
		{"key":"b","payer":"s","json":{"from":"b2"}},
		{"key":"c","payer":"s","json":{"from":"c"}}
	]`, response.Path("$.rows"))
}

func testStateTableScopesHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableScopes(e, "eosio.token/accounts", "")

	response.ValueEqual("block_num", 6)
	jsonValueEqual(t, "scopes", `["eosio1", "eosio2"]`, response.Path("$.scopes"))
}

func testStateTableScopesHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableScopes(e, "eosio.token/accounts", "block_num=3")

	response.ValueEqual("block_num", 3)
	jsonValueEqual(t, "scopes", `["eosio1", "eosio2", "eosio3"]`, response.Path("$.scopes"))
}

func testStateTablesForScopesHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForScopes(e, "eosio.token/accounts/eosio1|eosio2|eosio3", "json=true")

	// That is not the correct behavior, there should be only `eosio1` & `eosio3` in the tests
	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "tables", `[
		{ "account": "eosio.token","scope": "eosio1", "rows": [{ "key": "eos", "payer": "eosio1", "json": {"balance":"10.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio2", "rows": [{ "key": "eos", "payer": "eosio2", "json": {"balance":"22.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio3", "rows": []}
	]`, response.Path("$.tables"))
}

func testStateTablesForScopesHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForScopes(e, "eosio.token/accounts/eosio1|eosio2|eosio3", "block_num=3&json=true")

	assertIrrBlockInfo(response, "00000005aa")
	jsonValueEqual(t, "tables", `[
		{ "account": "eosio.token","scope": "eosio1", "rows": [{ "key": "eos", "payer": "eosio1", "json": {"balance":"1.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio2", "rows": [{ "key": "eos", "payer": "eosio2", "json": {"balance":"20.0000 EOS"}}]},
		{ "account": "eosio.token","scope": "eosio3", "rows": [{ "key": "eos", "payer": "eosio3", "json": {"balance":"3.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTablesForAccountsHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForAccounts(e, "eosio.token|eosio.nekot/accounts/eosio1", "json=true")

	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "tables", `[
		{"account":"eosio.nekot","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"1.0000 SOE"}}]},
		{"account":"eosio.token","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTablesForAccountsHistoricalJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTablesForAccounts(e, "eosio.token|eosio.nekot/accounts/eosio1", "block_num=4&json=true")

	assertIrrBlockInfo(response, "00000005aa")
	jsonValueEqual(t, "tables", `[
		{"account":"eosio.nekot","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"1.0000 SOE"}}]},
		{"account":"eosio.token","scope":"eosio1","rows":[{"key":"eos","payer":"eosio1","json":{"balance":"10.0000 EOS"}}]}
	]`, response.Path("$.tables"))
}

func testStateTableRowHeadJSON(ctx context.Context, t *testing.T, feedSourceWithBlocks blocksFeeder, e *httpexpect.Expect) {
	feedSourceWithBlocks(tableBlocks(t)...)

	response := okQueryStateTableRow(e, "eosio.nekot/accounts/eosio5/SOE", "json=true&key_type=symbol_code")

	assertHeadBlockInfo(response, "00000006aa", "00000005aa")
	jsonValueEqual(t, "row", `{"key":"SOE","payer":"eosio5","json":{"balance":"5.0000 SOE"}}`, response.Path("$.row"))
}

func tableBlocks(t *testing.T) []*pbcodec.Block {
	eosioTokenABI1 := readABI(t, "eosio.token.1.abi.json")
	eosioTestABI1 := readABI(t, "eosio.test.1.abi.json")
	eosioTestABI2 := readABI(t, "eosio.test.2.abi.json")
	eosioNekotABI1 := readABI(t, "eosio.nekot.1.abi.json")

	return []*pbcodec.Block{
		// Block #2 | Sets ABI on `eosio.token` (v1) and `eosio.test` (v1)
		testBlock(t, "00000002aa", "00000001aa",
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

func assertIrrBlockInfo(response *httpexpect.Object, libRef string) {
	lRef := bstream.NewBlockRefFromID(libRef)

	response.NotContainsKey("up_to_block_id")
	response.NotContainsKey("up_to_block_num")

	response.ValueEqual("last_irreversible_block_id", lRef.ID())
	response.ValueEqual("last_irreversible_block_num", lRef.Num())
}

func assertHeadBlockInfo(response *httpexpect.Object, blockRef string, libRef string) {
	bRef := bstream.NewBlockRefFromID(blockRef)
	lRef := bstream.NewBlockRefFromID(libRef)

	response.ValueEqual("up_to_block_id", bRef.ID())
	response.ValueEqual("up_to_block_num", bRef.Num())

	response.ValueEqual("last_irreversible_block_id", lRef.ID())
	response.ValueEqual("last_irreversible_block_num", lRef.Num())
}

// getFunctionName reads the program counter adddress and return the function
// name representing this address.
//
// The `FuncForPC` format is in the form of `github.com/.../.../package.func`.
// As such, we use `filepath.Base` to obtain the `package.func` part and then
// split it at the `.` to extract the function name.
func getFunctionName(i interface{}) string {
	pcIdentifier := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	baseName := filepath.Base(pcIdentifier)
	parts := strings.SplitN(baseName, ".", 2)
	if len(parts) <= 1 {
		return parts[0]
	}

	return parts[1]
}
