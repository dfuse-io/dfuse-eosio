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

import (
	"context"
	"fmt"
	"strings"

	"github.com/dfuse-io/derr"
)

// Authentication/Authorization Errors

func AuthTokenMissingError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPUnauthorizedError(ctx, nil, derr.C("auth_token_missing_error"),
		"Required authorization token not found.",
	)
}

func AuthInvalidTokenError(ctx context.Context, cause error, token string) *derr.ErrorResponse {
	return derr.HTTPUnauthorizedError(ctx, cause, derr.C("auth_invalid_token_error"),
		"Unable to correctly decode provided token.",
		"token", token,
		"reason", cause.Error(),
	)
}

func AuthInvalidTierError(ctx context.Context, tier string, expectedTier string) *derr.ErrorResponse {
	return derr.HTTPForbiddenError(ctx, nil, derr.C("auth_invalid_tier_error"),
		"Your are not allowed to perform this operation.",
		"tier", tier,
		"required_tier", expectedTier,
	)
}

func AuthInvalidStreamingStartBlockError(
	ctx context.Context,
	actualBlockNum uint32,
	requestedStartBlock uint32,
	authStartBlock uint32,
) *derr.ErrorResponse {
	return derr.HTTPForbiddenError(ctx, nil,
		derr.C("auth_invalid_streaming_start_block_error"),
		streamingInvalidBlockMessage(actualBlockNum, requestedStartBlock, authStartBlock),
	)
}

func streamingInvalidBlockMessage(actualBlockNum uint32, requestedStartBlock uint32, startBlock uint32) string {
	contactSupport := "Contact support@dfuse.io or reach us via our dfuse Telegram channels to request an API key able to retrieve more historical data."
	if startBlock >= 0 {
		return fmt.Sprintf(
			"Your credentials allow you to go back as far as block #%d, but you requested block #%d. %s",
			startBlock,
			requestedStartBlock,
			contactSupport,
		)
	}

	difference := actualBlockNum - requestedStartBlock
	return fmt.Sprintf(
		"Your credentials allow you to go back up to %d blocks, but you requested to go %d blocks back in time. %s",
		-startBlock,
		difference,
		contactSupport,
	)
}

// todo add rate limits as response
func RateLimitTooManyRequests(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPTooManyRequestsError(ctx, nil,
		derr.C("ratelimiter_too_many_requests"),
		"You exceeded your rate limit.",
	)
}

// Database Errors

func DBAccountNotFoundError(ctx context.Context, account string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_account_not_found_error"),
		"The requested account was not found.",
		"account", account,
	)
}

func DBABINotFoundError(ctx context.Context, account string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_abi_not_found_error"),
		"The requested ABI was not .",
		"account", account,
	)
}

func DBBlockNotFoundError(ctx context.Context, identifier string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_block_not_found_error"),
		"The requested block was not found.",
		"block", identifier,
	)
}

func DBForumProposalNotFoundError(ctx context.Context, proposalName string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_forum_proposal_not_found_error"),
		"The requested forum proposal was not found.",
		"proposal_name", proposalName,
	)
}

func DBTrxNotFoundError(ctx context.Context, trxID string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_trx_not_found_error"),
		"The requested transaction was not found.",
		"trx_id", trxID,
	)
}

func DBTrxAppearanceTimeoutError(ctx context.Context, blockID string, trxID string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("data_trx_appearance_timeout_error"),
		"The requested transaction did not appear in a timely matter.",
		"trx_id", trxID,
		"last_seen_block_id", blockID,
	)
}

// Application Errors

func AppHeadInfoNotReadyError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPServiceUnavailableError(ctx, nil, derr.C("app_head_info_not_ready_error"),
		"Head info not ready, please try again later.",
	)
}

func AppPriceNotReadyError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPServiceUnavailableError(ctx, nil, derr.C("app_price_not_ready_error"),
		"Price not ready, please try again later.",
	)
}

func AppTableRowsCannotFetchInFutureError(ctx context.Context, blockNum uint32) *derr.ErrorResponse {
	return derr.HTTPServiceUnavailableError(ctx, nil, derr.C("app_table_rows_cannot_fetch_in_future_error"),
		"It's not valid to try fetching table rows for a block in the future.",
		"start_block", blockNum,
	)
}

func AppUnableToGetIrreversibleBlockIDError(ctx context.Context, identifier string) *derr.ErrorResponse {
	return derr.HTTPInternalServerError(ctx, nil, derr.C("data_unable_to_get_irreversible_block_id_error"),
		"Unable to get irreversible block ID.",
		"block_reference", identifier,
	)
}

func AppVoteTallyNotReadyError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPServiceUnavailableError(ctx, nil, derr.C("app_vote_tally_not_ready_error"),
		"Vote tally not ready, please try again later.",
	)
}

// WebSocket Errors

func WSBinaryMessageUnsupportedError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPLockedError(ctx, nil, derr.C("ws_binary_message_unsupported_error"),
		"Binary messages are not supported.",
	)
}

func WSInvalidJSONMessageError(ctx context.Context, err error) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, err, derr.C("ws_invalid_json_message_error"),
		"The received message is not a valid JSON.",
		"reason", err.Error(),
	)
}

func WSInvalidJSONMessageDataError(ctx context.Context, messageType string, err error) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, err, derr.C("ws_invalid_json_message_data_error"),
		"The received message data is not a valid JSON.",
		"reason", err.Error(),
	)
}

func WSMessageDataValidationError(ctx context.Context, err error) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, err, derr.C("ws_message_data_validation_error"),
		"The received message data is not valid.",
		"reason", err.Error(),
	)
}

func WSAlreadyClosedError(ctx context.Context) *derr.ErrorResponse {
	return derr.HTTPTeapotError(ctx, nil, derr.C("ws_too_much_stream_error"), // error should not go to the client, it means the client already left...
		"Connection already closed",
	)
}

func WSTooMuchStreamError(ctx context.Context, streamCount int, maxStreamCount int) *derr.ErrorResponse {
	return derr.HTTPLockedError(ctx, nil, derr.C("ws_too_much_stream_error"),
		"No more request listener could be created within this connection.",
		"actual_listener_count", streamCount,
		"maximum_listener_count", maxStreamCount,
	)
}

func WSStreamAlreadyExistError(ctx context.Context, requestID string) *derr.ErrorResponse {
	return derr.HTTPConflictError(ctx, nil, derr.C("ws_stream_already_exist_error"),
		"A request listener with this id already exists.",
		"request_id", requestID,
	)
}

func WSStreamNotFoundError(ctx context.Context, requestID string) *derr.ErrorResponse {
	return derr.HTTPConflictError(ctx, nil, derr.C("ws_stream_not_found_error"),
		"A request listener with this id does not exist.",
		"request_id", requestID,
	)
}

func WSUnknownMessageError(ctx context.Context, messageType string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("ws_unknown_message_error"),
		"The received message type is unknown.",
		"type", messageType,
	)
}

func WSUnavailableMessageError(ctx context.Context, messageType string) *derr.ErrorResponse {
	return derr.HTTPBadRequestError(ctx, nil, derr.C("ws_unavailable_message_error"),
		"The received message type is no more available.",
		"type", messageType,
	)
}

func WSUnableToUpgradeConnectionError(ctx context.Context, status int, cause error) *derr.ErrorResponse {
	return derr.HTTPErrorFromStatus(status, ctx, cause, derr.C("ws_upgrade_failed_error"),
		"Unable to upgrade WebSocket connection correctly.",
		"reason", sanitizeWebSocketReasonMessage(cause.Error()),
	)
}

func sanitizeWebSocketReasonMessage(message string) string {
	return strings.TrimPrefix(message, "websocket: ")
}
