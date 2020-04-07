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

package fluxdb

import "context"

type TestTotalActivatedStakeResponse struct {
	totalActivatedStake float64
	err                 error
}

type TestProducersResponse struct {
	producers  []Producer
	totalVotes float64
	err        error
}

type TestFluxHelper struct {
	totalActivatedStakeResponse *TestTotalActivatedStakeResponse
	producersResponse           *TestProducersResponse
}

func (c *TestFluxHelper) SetTotalActivatedStakeResponse(totalActivatedStake float64, err error) {
	c.totalActivatedStakeResponse = &TestTotalActivatedStakeResponse{
		totalActivatedStake: totalActivatedStake,
		err:                 err,
	}
}

func (c *TestFluxHelper) SetProducersResponse(producers []Producer, totalVotes float64, err error) {
	c.producersResponse = &TestProducersResponse{
		producers:  producers,
		totalVotes: totalVotes,
		err:        err,
	}
}

func (c *TestFluxHelper) QueryTotalActivatedStake(ctx context.Context) (float64, error) {
	return c.totalActivatedStakeResponse.totalActivatedStake, c.totalActivatedStakeResponse.err
}

func (c *TestFluxHelper) QueryProducers(ctx context.Context) ([]Producer, float64, error) {
	return c.producersResponse.producers, c.producersResponse.totalVotes, c.producersResponse.err
}

func NewTestFluxHelper() *TestFluxHelper {
	return &TestFluxHelper{}
}
