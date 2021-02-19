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

package statedb

import (
	"context"

	"github.com/dfuse-io/derr"
	eos "github.com/eoscanada/eos-go"
)

// App Errors

func AppBlockNumHigherThanHeadBlockError(ctx context.Context, chosenBlockNum, headBlockNum, lastWrittenBlockNum uint64) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("app_block_num_higher_than_head_block_error"), "The requested block num is higher than head block.",
		"request_block_num", chosenBlockNum,
		"head_block_num", headBlockNum,
		"last_written_block_num", lastWrittenBlockNum,
	)
}

func AppBlockNumHigherThanLIBError(ctx context.Context, chosenBlockNum, lastWrittenBlockNum uint64) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("app_block_num_higher_than_lib_error"), "The requested block num is higher than the last irreversible block and 'irreversible_only' was set.",
		"request_block_num", chosenBlockNum,
		"last_written_block_num", lastWrittenBlockNum,
	)
}

func AppNotReadyError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPServiceUnavailableError(ctx, nil, derr.C("app_not_ready_error"), "The service is not ready to accept requests yet, retry later.")
}

// Data Errors

func DataABINotFoundError(ctx context.Context, account string, blockNum uint64) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_abi_not_found_error"), "Cannot find an ABI for request account at this block height.",
		"account", account,
		"block_num", blockNum,
	)
}

func DataDecodingRowError(ctx context.Context, hexData string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_decoding_table_row_error"), "Unable to decode row against ABI.",
		"data", hexData,
	)
}

func DataPublicKeyNotFoundError(ctx context.Context, publicKey string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_public_key_not_found_error"), "This public key does not exist at this block height.",
		"public_key", publicKey,
	)
}

func DataTableNotFoundError(ctx context.Context, account eos.AccountName, table eos.TableName) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_table_not_found_error"), "Table does not exist in ABI at this block height.",
		"account", account,
		"table", table,
	)
}

func DataRowNotFoundError(ctx context.Context, account eos.AccountName, table eos.TableName, scope eos.AccountName, primaryKey string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_row_not_found_error"), "Row does not exist in table at this block height.",
		"account", account,
		"scope", scope,
		"table", table,
		"primary_key", primaryKey,
	)
}

func HackDataTableExcludedError(ctx context.Context, account eos.AccountName, table eos.TableName, scope eos.AccountName) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_table_excluded_error"), "This table has been temporarly excluded from being readable.",
		"account", account,
		"scope", scope,
		"table", table,
	)
}
