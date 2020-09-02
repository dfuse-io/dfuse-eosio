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
	"encoding/json"
	"fmt"
	"sort"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/eoscanada/eos-go"
)

type Producer struct {
	Owner        string  `json:"owner"`
	TotalVotes   float64 `json:"total_votes"`
	ProducerKey  string  `json:"producer_key"`
	IsActive     bool    `json:"is_active"`
	URL          string  `json:"url"`
	UnpaidBlocks int     `json:"unpaid_blocks"`
	Location     int     `json:"location"`
	//LastClaimTime eos.JSONFloat64 `json:"last_claim_time"`
}

type Global struct {
	MaxBlockNetUsage               int       `json:"max_block_net_usage"`
	TargetBlockNetUsagePct         int       `json:"target_block_net_usage_pct"`
	MaxTransactionNetUsage         int       `json:"max_transaction_net_usage"`
	BasePerTransactionNetUsage     int       `json:"base_per_transaction_net_usage"`
	NetUsageLeeway                 int       `json:"net_usage_leeway"`
	ContextFreeDiscountNetUsageNum int       `json:"context_free_discount_net_usage_num"`
	ContextFreeDiscountNetUsageDen int       `json:"context_free_discount_net_usage_den"`
	MaxBlockCPUUsage               int       `json:"max_block_cpu_usage"`
	TargetBlockCPUUsagePct         int       `json:"target_block_cpu_usage_pct"`
	MaxTransactionCPUUsage         int       `json:"max_transaction_cpu_usage"`
	MinTransactionCPUUsage         int       `json:"min_transaction_cpu_usage"`
	MaxTransactionLifetime         int       `json:"max_transaction_lifetime"`
	DeferredTrxExpirationWindow    int       `json:"deferred_trx_expiration_window"`
	MaxTransactionDelay            int       `json:"max_transaction_delay"`
	MaxInlineActionSize            int       `json:"max_inline_action_size"`
	MaxInlineActionDepth           int       `json:"max_inline_action_depth"`
	MaxAuthorityDepth              int       `json:"max_authority_depth"`
	MaxRAMSize                     string    `json:"max_ram_size"`
	TotalRAMBytesReserved          eos.Int64 `json:"total_ram_bytes_reserved"`
	TotalRAMStake                  eos.Int64 `json:"total_ram_stake"`
	LastProducerScheduleUpdate     string    `json:"last_producer_schedule_update"`
	//LastPervoteBucketFill          int64     `json:"last_pervote_bucket_fill,string"`
	PervoteBucket       eos.Int64   `json:"pervote_bucket"`
	PerblockBucket      eos.Int64   `json:"perblock_bucket"`
	TotalUnpaidBlocks   eos.Int64   `json:"total_unpaid_blocks"`
	TotalActivatedStake eos.Float64 `json:"total_activated_stake"`
	//ThreshActivatedStakeTime       int64     `json:"thresh_activated_stake_time,string"`
	LastProducerScheduleSize eos.Int64   `json:"last_producer_schedule_size"`
	TotalProducerVoteWeight  eos.Float64 `json:"total_producer_vote_weight"`
	LastNameClose            string      `json:"last_name_close"`
}

type StateHelper interface {
	QueryTotalActivatedStake(ctx context.Context) (float64, error)
	QueryProducers(ctx context.Context) ([]Producer, float64, error)
}

type DefaultFluxHelper struct {
	client pbstatedb.StateClient
}

func NewDefaultFluxHelper(client pbstatedb.StateClient) *DefaultFluxHelper {
	return &DefaultFluxHelper{
		client: client,
	}
}

func (f *DefaultFluxHelper) QueryTotalActivatedStake(ctx context.Context) (float64, error) {
	response, err := f.client.GetTableRow(ctx, &pbstatedb.GetTableRowRequest{
		BlockNum:   0,
		Contract:   "eosio",
		Table:      "global",
		Scope:      "eosio",
		PrimaryKey: "global",
		KeyType:    "name",
		ToJson:     true,
	})
	if err != nil {
		return 0, fmt.Errorf("statedb read global table: %w", err)
	}

	var global Global
	err = json.Unmarshal([]byte(response.Row.Json), &global)
	if err != nil {
		return 0, fmt.Errorf("umarshalling global chain info: %w", err)
	}

	return float64(global.TotalActivatedStake), nil
}

func (f *DefaultFluxHelper) QueryProducers(ctx context.Context) ([]Producer, float64, error) {
	request := &pbstatedb.StreamTableRowsRequest{
		BlockNum: 0,
		Contract: "eosio",
		Table:    "producers",
		Scope:    "eosio",
		KeyType:  "name",
		ToJson:   true,
	}

	sum := 0.0
	i := 0
	var producers []Producer
	_, err := pbstatedb.ForEachTableRows(ctx, f.client, request, func(response *pbstatedb.TableRowResponse) error {
		var producer Producer
		err := json.Unmarshal([]byte(response.Json), &producer)
		if err != nil {
			return fmt.Errorf("unmarshal producer at index #%d", i)
		}

		sum += producer.TotalVotes
		if producer.IsActive {
			producers = append(producers, producer)
		}

		i++

		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("statedb read producers table: %w", err)
	}

	sort.Slice(producers, func(i, j int) bool {
		return producers[i].TotalVotes > producers[j].TotalVotes
	})

	return producers, sum, nil
}
