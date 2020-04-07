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

	"github.com/dfuse-io/derr"
	"github.com/eoscanada/eos-go"
	fluxdb "github.com/dfuse-io/dfuse-eosio/fluxdb-client"
)

type ABIGetter interface {
	GetABI(ctx context.Context, blockNum uint32, account eos.AccountName) (*eos.ABI, error)
}

type DefaultABIGetter struct {
	client fluxdb.Client
}

func NewDefaultABIGetter(client fluxdb.Client) *DefaultABIGetter {
	return &DefaultABIGetter{
		client: client,
	}
}

func (g *DefaultABIGetter) GetABI(ctx context.Context, blockNum uint32, account eos.AccountName) (*eos.ABI, error) {
	response, err := g.client.GetABI(ctx, blockNum, account)
	if err != nil {
		return nil, derr.Wrapf(err, "unable to get ABI for %s", account)
	}

	return response.ABI, nil
}
