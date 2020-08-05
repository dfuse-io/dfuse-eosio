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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosws/statedb"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_onGetVoteTally(t *testing.T) {

	subscriptionHub := newTestSubscriptionHub(t, 0, nil)
	stateClient := pbstatedb.NewMockStateClient()
	stateDBHelper := statedb.NewTestStateHelper()

	cases := []struct {
		name                   string
		msg                    string
		totalActivatedStake    float64
		totalActivatedStakeErr error
		producers              []statedb.Producer
		producersTotalVotes    float64
		producersErr           error
		expectedOutput         []string
		expectedError          string
	}{
		{
			name:                "sunny path",
			msg:                 `{"type":"get_vote_tally","req_id":"abc","listen":true,"fetch":true}`,
			expectedOutput:      []string{`{"type":"vote_tally","req_id":"abc","data":{"vote_tally":{"total_activated_stake":999,"total_votes":777,"decay_weight":5.213924440732366e-89,"producers":[{"owner":"","total_votes":123,"producer_key":"producer.key","is_active":false,"url":"","unpaid_blocks":0,"location":0}]}}}`},
			totalActivatedStake: 999,
			producers: []statedb.Producer{
				{TotalVotes: 123, ProducerKey: "producer.key"},
			},
			producersTotalVotes: 777,
		},
	}

	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			stateDBHelper.SetTotalActivatedStakeResponse(c.totalActivatedStake, c.totalActivatedStakeErr)
			stateDBHelper.SetProducersResponse(c.producers, c.producersTotalVotes, c.producersErr)

			NowFunc = func() time.Time {
				t := time.Time{}
				fmt.Println("time : ", t)
				return t
			}

			voteTallyHub := NewVoteTallyHub(stateDBHelper)
			go voteTallyHub.Launch(context.Background())

			handler := NewWebsocketHandler(
				nil,
				nil,
				nil,
				subscriptionHub,
				stateClient,
				voteTallyHub,
				nil,
				nil,
				NewTestIrreversibleFinder("00000002a", nil),
				0,
			)

			conn, closer := newTestConnection(t, handler)
			defer closer()

			conn.WriteMessage(1, []byte(c.msg))

			validateOutput(t, "", c.expectedOutput, conn)

		})
	}
}

func TestVoteTallyHub_FetchVoteTallyData(t *testing.T) {
	cases := []struct {
		name                   string
		totalActivatedStake    float64
		totalActivatedStakeErr error
		producers              []statedb.Producer
		producersTotalVotes    float64
		producersErr           error
		expectedTallyJSON      string
		expectedError          string
	}{
		{
			name:                "sunny path",
			expectedTallyJSON:   `{"total_activated_stake":999,"total_votes":777,"decay_weight":5.213924440732366e-89,"producers":[{"owner":"","total_votes":123,"producer_key":"producer.key","is_active":false,"url":"","unpaid_blocks":0,"location":0}]}`,
			totalActivatedStake: 999,
			producers: []statedb.Producer{
				{TotalVotes: 123, ProducerKey: "producer.key"},
			},
			producersTotalVotes: 777,
		},
		{
			name:                   "total Activated Stake Err",
			totalActivatedStakeErr: fmt.Errorf("error.1"),
			expectedError:          "query total active stake: error.1",
		},
		{
			name:          "producer Err",
			producersErr:  fmt.Errorf("error.1"),
			expectedError: "query producers: error.1",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			stateHelper := statedb.NewTestStateHelper()
			stateHelper.SetTotalActivatedStakeResponse(c.totalActivatedStake, c.totalActivatedStakeErr)
			stateHelper.SetProducersResponse(c.producers, c.producersTotalVotes, c.producersErr)

			hub := NewVoteTallyHub(stateHelper)
			NowFunc = func() time.Time {
				t := time.Time{}
				fmt.Println("time : ", t)
				return t
			}
			voteTally, err := hub.FetchVoteTally()

			if c.expectedError != "" {
				assert.Equal(t, c.expectedError, err.Error())
				return
			}

			require.NoError(t, err)
			voteTallyDataJSON, err := json.Marshal(voteTally.Data.VoteTally)
			require.NoError(t, err)
			assert.Equal(t, c.expectedTallyJSON, string(voteTallyDataJSON))
		})
	}
}

func TestVoteWeightToday(t *testing.T) {
	tests := []struct {
		now    time.Time
		weight float64
	}{
		{
			time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC),
			2.0,
		},
		{
			time.Date(2018, time.June, 11, 0, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 12, 0, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 13, 0, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 14, 0, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 15, 0, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 15, 20, 0, 0, 0, time.UTC),
			370727.6000947326,
		},
		{
			time.Date(2018, time.June, 15, 23, 59, 59, 999999, time.UTC),
			370727.6000947326,
		},
		{ // Weeks turn over on SATURDAY MORNING UTC :)
			time.Date(2018, time.June, 16, 0, 0, 0, 0, time.UTC),
			375702.3903121556,
		},
	}

	for idx, test := range tests {
		res := voteWeightToday(func() time.Time {
			return test.now
		})
		assert.Equal(t, test.weight, res, fmt.Sprintf("idx=%d", idx))
	}
}
