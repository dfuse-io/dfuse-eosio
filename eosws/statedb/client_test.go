package statedb

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalUnmarshall(t *testing.T) {

	j := `{
	"last_name_close": "2020-08-20T09:32:58",
	"total_producer_vote_weight": 546040288249037850000,
	"last_producer_schedule_size": 21,
	"thresh_activated_stake_time": "2020-02-19T19:26:33",
	"total_activated_stake": "11510100000000",
	"total_unpaid_blocks": 4852281,
	"perblock_bucket": "15321432760",
	"pervote_bucket": 700213915,
	"last_pervote_bucket_fill": "2020-09-02T09:05:02.5",
	"last_producer_schedule_update": "2020-09-02T14:38:24",
	"total_ram_stake": 1079833087,
	"total_ram_bytes_reserved": "6697154415",
	"max_ram_size": "68719476736",
	"max_authority_depth": 6,
	"max_inline_action_depth": 32,
	"max_inline_action_size": 524287,
	"max_transaction_delay": 3888000,
	"deferred_trx_expiration_window": 600,
	"max_transaction_lifetime": 3600,
	"min_transaction_cpu_usage": 1,
	"max_transaction_cpu_usage": 150000,
	"target_block_cpu_usage_pct": 10,
	"max_block_cpu_usage": 400000,
	"context_free_discount_net_usage_den": 100,
	"context_free_discount_net_usage_num": 20,
	"net_usage_leeway": 500,
	"base_per_transaction_net_usage": 12,
	"max_transaction_net_usage": 524287,
	"target_block_net_usage_pct": 1000,
	"max_block_net_usage": 524288
}`

	var global Global
	err := json.Unmarshal([]byte(j), &global)
	require.NoError(t, err)
}
