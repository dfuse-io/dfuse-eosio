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
	"math"
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/statedb"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"github.com/streamingfast/derr"
	"go.uber.org/zap"
)

var NowFunc func() time.Time

func init() {
	NowFunc = time.Now
}

func (ws *WSConn) onGetVoteTally(ctx context.Context, msg *wsmsg.GetVoteTally) {
	if msg.Listen {
		ws.voteTallyHub.Subscribe(ctx, msg, ws)
		return
	}
	if msg.Fetch {
		out := ws.voteTallyHub.Last()
		if out == nil {
			ws.EmitErrorReply(ctx, msg, AppVoteTallyNotReadyError(ctx))
		}
		metrics.DocumentResponseCounter.Inc()
		ws.EmitReply(ctx, msg, out)
	}
}

type VoteTallyHub struct {
	CommonHub

	stateHelper statedb.StateHelper
}

func NewVoteTallyHub(stateHelper statedb.StateHelper) *VoteTallyHub {
	return &VoteTallyHub{
		CommonHub:   CommonHub{name: "VoteTally"},
		stateHelper: stateHelper,
	}
}

func (h *VoteTallyHub) Launch(ctx context.Context) {
	for {
		voteTally, err := h.FetchVoteTally()
		if err != nil {
			zlog.Error("fetching vote tally", zap.Error(err))
			time.Sleep(10 * time.Second)
			continue
		}
		h.SetLast(voteTally)
		h.EmitAll(ctx, voteTally)
		time.Sleep(100 * time.Second)
	}
}

func (h *VoteTallyHub) FetchVoteTally() (*wsmsg.VoteTally, error) {

	totalActivatedStake, err := h.stateHelper.QueryTotalActivatedStake(context.Background())
	if err != nil {
		return nil, derr.Wrap(err, "query total active stake")
	}

	producers, totalVotes, err := h.stateHelper.QueryProducers(context.Background())
	if err != nil {
		return nil, derr.Wrap(err, "query producers")
	}

	vtd := &wsmsg.VoteTallyData{
		TotalActivatedStake: totalActivatedStake,
		TotalVotes:          totalVotes,
		DecayWeight:         voteWeightToday(NowFunc),
		Producers:           producers,
	}
	voteTally := wsmsg.NewVoteTally(vtd)
	return voteTally, nil
}

const secondsInAWeek = 86400 * 7
const weeksInAYear = 52

// voteWeightToday computes the stake2vote weight for EOS, in order to compute the decaying value.
func voteWeightToday(nowFunc func() time.Time) float64 {
	now := time.Now().UTC()
	if nowFunc != nil {
		now = nowFunc()
	}

	y2k := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	elapsedSinceY2K := now.Sub(y2k)
	weeksSinceY2K := int64(elapsedSinceY2K.Seconds() / secondsInAWeek) // truncate to integer weeks
	yearsSinceY2K := float64(weeksSinceY2K) / weeksInAYear

	return math.Pow(2, yearsSinceY2K)
}
