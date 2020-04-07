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

package wsmsg

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/eosws/fluxdb"
)

func init() {
	RegisterIncomingMessage("get_vote_tally", GetVoteTally{})
	RegisterOutgoingMessage("vote_tally", VoteTally{})
}

// OUTGOING MESSAGE

type VoteTally struct {
	CommonOut
	Data struct {
		VoteTally *VoteTallyData `json:"vote_tally"`
	} `json:"data"`
}

func NewVoteTally(data *VoteTallyData) *VoteTally {
	out := &VoteTally{}
	out.Data.VoteTally = data
	return out
}

// INCOMING MESSAGE

type GetVoteTally struct {
	CommonIn
}

func (t *GetVoteTally) Validate(ctx context.Context) error {
	if !t.Listen && !t.Fetch {
		return fmt.Errorf("one of 'listen' or 'fetch' required (both supported)")
	}
	if t.IrreversibleOnly {
		return fmt.Errorf("'irreversible_only' is not supported")
	}
	return nil
}

// other structs

type VoteTallyData struct {
	TotalActivatedStake float64           `json:"total_activated_stake"`
	TotalVotes          float64           `json:"total_votes"`
	DecayWeight         float64           `json:"decay_weight"`
	Producers           []fluxdb.Producer `json:"producers"`
}
